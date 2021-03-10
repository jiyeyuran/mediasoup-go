package mediasoup

import (
	"encoding/json"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jiyeyuran/mediasoup-go/netstring"
)

type notification struct {
	TargetId string          `json:"targetId,omitempty"`
	Event    string          `json:"event,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
}

type PayloadChannel struct {
	IEventEmitter
	logger              Logger
	closed              int32
	producerSocket      net.Conn
	consumerSocket      net.Conn
	nextId              int64
	sents               sync.Map
	sentsLen            int64
	ongoingNotification *notification
	closeCh             chan struct{}
}

func newPayloadChannel(producerSocket, consumerSocket net.Conn) *PayloadChannel {
	logger := NewLogger("PayloadChannel")

	logger.Debug("constructor()")

	channel := &PayloadChannel{
		IEventEmitter:  NewEventEmitter(),
		logger:         logger,
		producerSocket: producerSocket,
		consumerSocket: consumerSocket,
		closeCh:        make(chan struct{}),
	}

	go channel.runReadLoop()

	return channel
}

func (c *PayloadChannel) Close() {
	if atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		c.logger.Debug("close()")

		c.producerSocket.Close()
		c.consumerSocket.Close()

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
	ns1 := netstring.Encode(rawData)
	ns2 := netstring.Encode(payload)

	if len(ns1) > NS_MESSAGE_MAX_LEN {
		err = errors.New("PayloadChannel notification too big")
		return
	}
	if len(ns2) > NS_MESSAGE_MAX_LEN {
		err = errors.New("PayloadChannel payload too big")
		return
	}

	if _, err = c.producerSocket.Write(ns1); err != nil {
		return
	}
	if _, err = c.producerSocket.Write(ns2); err != nil {
		return
	}

	return
}

func (c *PayloadChannel) Request(method string, internal interface{}, data interface{}, payload []byte) (rsp workerResponse) {
	if c.Closed() {
		rsp.err = NewInvalidStateError("PayloadChannel closed")
		return
	}

	id := int64(1)

	if atomic.LoadInt64(&c.nextId) < 4294967295 {
		id = atomic.AddInt64(&c.nextId, 1)
	} else {
		atomic.StoreInt64(&c.nextId, id)
	}

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
	ns1 := netstring.Encode(rawData)
	ns2 := netstring.Encode(payload)

	if len(ns1) > NS_MESSAGE_MAX_LEN {
		rsp.err = errors.New("PayloadChannel request too big")
		return
	}
	if len(ns2) > NS_MESSAGE_MAX_LEN {
		rsp.err = errors.New("PayloadChannel payload too big")
		return
	}

	if _, rsp.err = c.producerSocket.Write(ns1); rsp.err != nil {
		return
	}
	if _, rsp.err = c.producerSocket.Write(ns2); rsp.err != nil {
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

func (c *PayloadChannel) runReadLoop() {
	decoder := netstring.NewDecoder()

	go func() {
		for {
			select {
			case nsPayload := <-decoder.Result():
				c.processData(nsPayload)
			case <-c.closeCh:
				return
			}
		}
	}()

	buf := make([]byte, NS_PAYLOAD_MAX_LEN)

	for {
		n, err := c.consumerSocket.Read(buf)
		if err != nil {
			c.logger.Error("Channel error: %s", err)
			break
		}
		data := buf[:n]

		decoder.Feed(data)

		if decoder.Length() > NS_PAYLOAD_MAX_LEN {
			c.logger.Error("receiving buffer is full, discarding all data into it")
			decoder.Reset()
		}
	}

	c.Close()
}

func (c *PayloadChannel) processData(payload []byte) {
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
