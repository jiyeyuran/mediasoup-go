package channel

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"log/slog"
	"strconv"
	"sync"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	FbsLog "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Log"
	FbsMessage "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Message"
	FbsNotification "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Notification"
	FbsRequest "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Request"
	FbsResponse "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Response"
)

const (
	RequestTimeout = time.Second * 5
	MaxMessageLen  = 4194308
	MaxRequestId   = 4294967295
)

type Channel struct {
	mu            sync.RWMutex
	subsMu        sync.RWMutex
	nextId        uint32
	w             io.WriteCloser
	r             io.ReadCloser
	reader        *bufio.Reader
	writeBuf      *bytes.Buffer
	readWaitGroup sync.WaitGroup
	fbsBuilder    *flatbuffers.Builder
	message       *FbsMessage.MessageT
	timerPool     *sync.Pool
	ssid          int64
	subs          map[string][]*Subscription
	responsesCh   map[uint32]chan *FbsResponse.ResponseT
	logger        *slog.Logger
	timeout       time.Duration
	closed        bool
}

func NewChannel(w io.WriteCloser, r io.ReadCloser, logger *slog.Logger) *Channel {
	return &Channel{
		w:          w,
		r:          r,
		reader:     bufio.NewReader(r),
		writeBuf:   new(bytes.Buffer),
		fbsBuilder: flatbuffers.NewBuilder(1024),
		message: &FbsMessage.MessageT{
			Data: &FbsMessage.BodyT{},
		},
		timerPool:   &sync.Pool{New: func() any { return time.NewTimer(RequestTimeout) }},
		subs:        make(map[string][]*Subscription),
		responsesCh: make(map[uint32]chan *FbsResponse.ResponseT),
		timeout:     RequestTimeout,
		logger:      logger,
	}
}

func (c *Channel) Start() {
	c.logger.Debug("Start()")

	go c.readLoop()
}

func (c *Channel) Notify(notification *FbsNotification.NotificationT) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrChannelClosed
	}

	c.logger.Debug("Notify()", "event", notification.Event, "handlerId", notification.HandlerId)

	builder := c.fbsBuilder
	message := c.message
	message.Data.Type = FbsMessage.BodyNotification
	message.Data.Value = notification
	builder.Finish(message.Pack(builder))
	payload := builder.FinishedBytes()
	message.Data.Value = nil
	builder.Reset()

	if len(payload) > MaxMessageLen {
		return ErrBodyTooLarge
	}

	err := c.write(payload)
	if err != nil {
		c.logger.Error("notify failed", "event", notification.Event, "handlerId", notification.HandlerId, "error", err)
	}
	return err
}

func (c *Channel) Request(req *FbsRequest.RequestT) (any, error) {
	c.mu.Lock()

	if c.closed {
		c.mu.Unlock()
		return nil, ErrChannelClosed
	}

	if c.nextId < MaxRequestId {
		c.nextId++
	} else {
		c.nextId = 1
	}

	req.Id = c.nextId
	if len(req.HandlerId) == 0 {
		req.HandlerId = "default"
	}
	if req.Body == nil {
		req.Body = &FbsRequest.BodyT{
			Type: FbsRequest.BodyNONE,
		}
	}

	c.logger.Debug("Request()", "requestId", req.Id, "method", req.Method, "handlerId", req.HandlerId)

	builder := c.fbsBuilder
	message := c.message
	message.Data.Type = FbsMessage.BodyRequest
	message.Data.Value = req
	builder.Finish(message.Pack(builder))
	payload := builder.FinishedBytes()
	message.Data.Value = nil
	builder.Reset()

	if len(payload) > MaxMessageLen {
		c.mu.Unlock()
		return nil, ErrBodyTooLarge
	}

	// Create new literal Inbox and map to a chan msg.
	mch := make(chan *FbsResponse.ResponseT, 1)
	c.responsesCh[req.Id] = mch
	defer func() {
		c.mu.Lock()
		delete(c.responsesCh, req.Id)
		c.mu.Unlock()
	}()

	if err := c.write(payload); err != nil {
		c.mu.Unlock()
		return nil, err
	}

	c.mu.Unlock()

	timer := c.timerPool.Get().(*time.Timer)
	timer.Reset(c.timeout)

	defer func() {
		if !timer.Stop() {
			<-timer.C
		}
		c.timerPool.Put(timer)
	}()

	select {
	case m, ok := <-mch:
		if !ok {
			return nil, ErrChannelClosed
		}
		if m.Accepted {
			if m.Body == nil {
				return nil, nil
			}
			return m.Body.Value, nil
		}
		c.logger.Error("request failed", "id", m.Id, "error", m.Error, "reason", m.Reason)
		return nil, errors.New(m.Error + ": " + m.Reason)

	case <-timer.C:
		return nil, ErrChannelRequestTimeout
	}
}

func (c *Channel) Subscribe(subj string, cb Handler) *Subscription {
	c.logger.Debug("Subscribe()", "subject", subj)

	sub := &Subscription{
		Subject: subj,
		mcb:     cb,
		conn:    c,
	}
	sub.pCond = sync.NewCond(&sub.mu)

	c.subsMu.Lock()
	subs := append(c.subs[subj], sub)
	c.ssid++
	sub.sid = c.ssid
	c.subs[subj] = subs
	c.subsMu.Unlock()

	return sub
}

func (c *Channel) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}
	c.logger.Debug("Close()")

	c.doClose()
}

func (c *Channel) doClose() {
	c.closed = true

	c.w.Close()
	c.r.Close()

	// wait for readLoop to finish
	c.readWaitGroup.Wait()

	c.subsMu.RLock()
	allSubs := c.subs
	c.subsMu.RUnlock()
	for _, subs := range allSubs {
		for _, sub := range subs {
			sub.Unsubscribe()
		}
	}

	for _, ch := range c.responsesCh {
		close(ch)
	}
}

// Closed tests if Channel has been closed.
func (c *Channel) Closed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.closed
}

func (c *Channel) waitForMsgs(s *Subscription) {
	var closed bool
	var delivered, max uint64

	for {
		s.mu.Lock()
		if s.pHead == nil && !s.closed {
			s.pCond.Wait()
		}
		// Pop the msg off the list
		m := s.pHead
		if m != nil {
			s.pHead = m.next
			if s.pHead == nil {
				s.pTail = nil
			}
		}
		mcb := s.mcb
		max = s.max
		closed = s.closed
		if !s.closed {
			s.delivered++
			delivered = s.delivered
		}
		s.mu.Unlock()

		if closed {
			break
		}

		// Deliver the message.
		if m != nil && (max == 0 || delivered <= max) {
			mcb(m.Data.Event, m.Data.Body)
		}
		// If we have hit the max for delivered msgs, remove sub.
		if max > 0 && delivered >= max {
			c.removeSub(s)
			break
		}
	}
}

// unsubscribe performs the low level unsubscribe to the server.
// Use Subscription.Unsubscribe()
func (c *Channel) unsubscribe(sub *Subscription, max int) error {
	var maxStr string
	if max > 0 {
		sub.mu.Lock()
		sub.max = uint64(max)
		if sub.delivered < sub.max {
			maxStr = strconv.Itoa(max)
		}
		sub.mu.Unlock()
	}

	c.subsMu.RLock()
	_, ok := c.subs[sub.Subject]
	c.subsMu.RUnlock()
	// Already unsubscribed
	if !ok {
		return nil
	}

	if len(maxStr) == 0 {
		c.removeSub(sub)
	}

	return nil
}

// Lock for c should be held here upon entry
func (c *Channel) removeSub(s *Subscription) {
	c.subsMu.Lock()
	var subs []*Subscription
	for _, sub := range c.subs[s.Subject] {
		if sub.sid != s.sid {
			subs = append(subs, sub)
		}
	}
	if len(subs) > 0 {
		c.subs[s.Subject] = subs
	} else {
		delete(c.subs, s.Subject)
	}
	c.subsMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	// Mark as invalid
	s.closed = true
	s.pCond.Broadcast()
}

func (c *Channel) write(payload []byte) error {
	c.writeBuf.Reset()

	size := uint32(len(payload))
	sizeBuf := [4]byte{}
	binary.NativeEndian.PutUint32(sizeBuf[:], size)
	c.writeBuf.Write(sizeBuf[:])
	c.writeBuf.Write(payload)

	_, err := c.w.Write(c.writeBuf.Bytes())
	return err
}

func (c *Channel) readLoop() {
	c.readWaitGroup.Add(1)
	defer func() {
		c.readWaitGroup.Done()
		c.mu.Lock()
		defer c.mu.Unlock()
		if !c.closed {
			c.doClose()
		}
	}()

	sizeBuf := [4]byte{}

	for {
		if _, err := io.ReadFull(c.reader, sizeBuf[:]); err != nil {
			break
		}
		size := int(binary.NativeEndian.Uint32(sizeBuf[:]))
		readBuf := make([]byte, size)
		if _, err := io.ReadFull(c.reader, readBuf); err != nil {
			break
		}
		c.processPayload(readBuf)
	}
}

func (c *Channel) processPayload(payload []byte) {
	message := FbsMessage.GetRootAsMessage(payload, 0).UnPack()

	switch message.Data.Type {
	case FbsMessage.BodyResponse:
		c.processResponse(message.Data.Value.(*FbsResponse.ResponseT))

	case FbsMessage.BodyNotification:
		c.processNotification(message.Data.Value.(*FbsNotification.NotificationT), 0)

	case FbsMessage.BodyLog:
		c.processLog(message.Data.Value.(*FbsLog.LogT))
	}
}

func (c *Channel) processResponse(response *FbsResponse.ResponseT) {
	c.mu.Lock()
	mch, ok := c.responsesCh[response.Id]
	c.mu.Unlock()

	if ok {
		mch <- response
	} else {
		c.logger.Error("received an unhandled response", "id", response.Id, "accepted", response.Accepted, "error", response.Error)
	}
}

func (c *Channel) processNotification(notification *FbsNotification.NotificationT, retried int) {
	c.subsMu.RLock()
	subs, ok := c.subs[notification.HandlerId]
	c.subsMu.RUnlock()
	if !ok {
		if retried >= 10 {
			c.logger.Error("received an unhandled notification", "subject", notification.HandlerId, "event", notification.Event)
			return
		}
		// It may happen that we receive a response from the worker followed by
		// a notification from the worker. If we emit the notification immediately
		// it may reach its target **before** the response, destroying the ordered
		/// delivery. So we must wait a bit here.
		// See https://github.com/versatica/mediasoup/issues/510
		time.Sleep(time.Microsecond)
		c.processNotification(notification, retried+1)
		return
	}

	for _, sub := range subs {
		sub.once.Do(func() {
			go c.waitForMsgs(sub)
		})
		m := &Msg{Data: notification}
		sub.mu.Lock()
		// Push onto the async pList
		if sub.pHead == nil {
			sub.pHead = m
			sub.pTail = m
			if sub.pCond != nil {
				sub.pCond.Signal()
			}
		} else {
			sub.pTail.next = m
			sub.pTail = m
		}
		sub.mu.Unlock()
	}
}

func (c *Channel) processLog(log *FbsLog.LogT) {
	payload := log.Data

	switch tp := payload[0]; tp {
	case 'D':
		c.logger.Debug(payload[1:])

	case 'W':
		c.logger.Warn(payload[1:])

	case 'E':
		c.logger.Error(payload[1:])

	case 'X':
		c.logger.Info(string(payload[1:]))
	}
}

func (c *Channel) ProcessNotificationForTesting(notification *FbsNotification.NotificationT) {
	c.processNotification(notification, 0)
}
