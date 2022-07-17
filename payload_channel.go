package mediasoup

import (
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jiyeyuran/mediasoup-go/netcodec"
)

type notification struct {
	TargetId string          `json:"targetId,omitempty"`
	Event    string          `json:"event,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
}

type PayloadChannel struct {
	IEventEmitter
	locker              sync.Mutex
	codec               netcodec.Codec
	logger              Logger
	closed              int32
	nextId              int64
	sents               sync.Map
	sentsLen            int64
	ongoingNotification *notification
	closeCh             chan struct{}
}

func newPayloadChannel(codec netcodec.Codec) *PayloadChannel {
	logger := NewLogger("PayloadChannel")

	logger.Debug("constructor()")

	channel := &PayloadChannel{
		IEventEmitter: NewEventEmitter(),
		logger:        logger,
		codec:         codec,
		closeCh:       make(chan struct{}),
	}

	return channel
}

func (c *PayloadChannel) Start() {
	go c.runReadLoop()
}

func (c *PayloadChannel) Close() {
	if atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		c.logger.Debug("close()")

		close(c.closeCh)
		c.RemoveAllListeners()
	}
}

func (c *PayloadChannel) Closed() bool {
	return atomic.LoadInt32(&c.closed) > 0
}

func (c *PayloadChannel) Notify(event string, internal interface{}, data interface{}, payload []byte) (err error) {
	if c.Closed() {
		err = NewInvalidStateError("PayloadChannel closed")
		return
	}
	notification := H{
		"event":    event,
		"internal": internal,
		"data":     data,
	}
	rawData, _ := json.Marshal(notification)

	return c.writeAll(rawData, payload)
}

func (c *PayloadChannel) Request(method string, internal interface{}, data interface{}, payload []byte) (rsp workerResponse) {
	if c.Closed() {
		rsp.err = NewInvalidStateError("PayloadChannel closed")
		return
	}
	id := atomic.AddInt64(&c.nextId, 1)
	atomic.CompareAndSwapInt64(&c.nextId, 4294967295, 1)

	c.logger.Debug("request() [method:%s, id:%d]", method, id)

	sent := sentInfo{
		id:     id,
		method: method,
		respCh: make(chan workerResponse),
	}
	c.sents.Store(id, sent)

	size := atomic.AddInt64(&c.sentsLen, 1)

	defer func() {
		c.sents.Delete(id)
		atomic.AddInt64(&c.sentsLen, -1)
	}()

	rawData, _ := json.Marshal(H{
		"id":       id,
		"method":   method,
		"internal": internal,
		"data":     data,
	})

	if rsp.err = c.writeAll(rawData, payload); rsp.err != nil {
		return
	}

	timeout := 1000 * (15 + (0.1 * float64(size)))
	timer := time.NewTimer(time.Duration(timeout) * time.Millisecond)
	defer timer.Stop()

	select {
	case rsp = <-sent.respCh:
		return
	case <-timer.C:
		rsp.err = errors.New("Channel request timeout")
	case <-c.closeCh:
		rsp.err = NewInvalidStateError("Channel closed")
	}

	return
}

func (c *PayloadChannel) writeAll(data, payload []byte) (err error) {
	if len(payload) > NS_MESSAGE_MAX_LEN {
		return errors.New("PayloadChannel data too big")
	}
	if len(payload) > NS_MESSAGE_MAX_LEN {
		return errors.New("PayloadChannel payload too big")
	}

	c.locker.Lock()
	defer c.locker.Unlock()

	if err = c.codec.WritePayload(data); err != nil {
		return
	}
	if len(payload) > 0 {
		if err = c.codec.WritePayload(payload); err != nil {
			return
		}
	}
	return
}

func (c *PayloadChannel) runReadLoop() {
	defer c.Close()

	for {
		payload, err := c.codec.ReadPayload()
		if err != nil {
			c.logger.Error("PayloadChannel error: %s", err)
			break
		}
		c.processPayload(payload)
	}
}

func (c *PayloadChannel) processPayload(payload []byte) {
	if c.ongoingNotification != nil {
		notification := c.ongoingNotification
		c.SafeEmit(notification.TargetId, notification.Event, notification.Data, payload)
		c.ongoingNotification = nil
		return
	}

	var msg struct {
		// response
		Id       int64  `json:"id,omitempty"`
		Accepted bool   `json:"accepted,omitempty"`
		Error    string `json:"error,omitempty"`
		Reason   string `json:"reason,omitempty"`
		// notification
		TargetId string `json:"targetId,omitempty"`
		Event    string `json:"event,omitempty"`
		// common data
		Data json.RawMessage `json:"data,omitempty"`
	}
	json.Unmarshal(payload, &msg)

	if msg.Id > 0 {
		value, ok := c.sents.Load(msg.Id)
		if !ok {
			c.logger.Error("received response does not match any sent request [id:%d]", msg.Id)
			return
		}
		sent := value.(sentInfo)

		if msg.Accepted {
			c.logger.Debug("request succeeded [method:%s, id:%d]", sent.method, sent.id)

			sent.respCh <- workerResponse{data: msg.Data}
		} else if len(msg.Error) > 0 {
			c.logger.Warn("request failed [method:%s, id:%d]: %s", sent.method, sent.id, msg.Reason)

			if msg.Error == "TypeError" {
				sent.respCh <- workerResponse{err: NewTypeError(msg.Reason)}
			} else {
				sent.respCh <- workerResponse{err: errors.New(msg.Reason)}
			}
		} else {
			c.logger.Error("received response is not accepted nor rejected [method:%s, id:%s]", sent.method, sent.id)
		}
	} else if len(msg.TargetId) > 0 && len(msg.Event) > 0 {
		c.ongoingNotification = &notification{
			TargetId: msg.TargetId,
			Event:    msg.Event,
			Data:     msg.Data,
		}
	} else {
		c.logger.Error("received message is not a response nor a notification")
	}
}
