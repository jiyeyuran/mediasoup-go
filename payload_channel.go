package mediasoup

import (
	"encoding/json"
	"errors"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jiyeyuran/mediasoup-go/netcodec"
)

type notifyInfo struct {
	requestData []byte
	payloadData []byte
	respCh      chan workerResponse
}

type PayloadChannel struct {
	locker              sync.Mutex
	codec               netcodec.Codec
	logger              Logger
	closed              int32
	nextId              int64
	sents               sync.Map
	listeners           sync.Map
	pendingNotification *notification
	sentChan            chan sentInfo
	notifyChan          chan notifyInfo
	closeCh             chan struct{}
}

func newPayloadChannel(codec netcodec.Codec) *PayloadChannel {
	logger := NewLogger("PayloadChannel")

	logger.Debug("constructor()")

	channel := &PayloadChannel{
		logger:     logger,
		codec:      codec,
		sentChan:   make(chan sentInfo),
		notifyChan: make(chan notifyInfo),
		closeCh:    make(chan struct{}),
	}

	return channel
}

func (c *PayloadChannel) Start() {
	go c.runWriteLoop()
	go c.runReadLoop()
}

func (c *PayloadChannel) Close() error {
	if atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		c.logger.Debug("close()")
		close(c.closeCh)
		return c.codec.Close()
	}
	return nil
}

func (c *PayloadChannel) Closed() bool {
	return atomic.LoadInt32(&c.closed) > 0
}

func (c *PayloadChannel) AddTargetHandler(targetId string, handler interface{}, options ...listenerOption) {
	c.listeners.Store(targetId, newInternalListener(handler, options...))
}

func (c *PayloadChannel) RemoveTargetHandler(targetId string) {
	c.listeners.Delete(targetId)
}

func (c *PayloadChannel) Notify(event string, internal internalData, data interface{}, payload []byte) (err error) {
	if c.Closed() {
		return NewInvalidStateError("PayloadChannel closed")
	}
	notification := workerNotification{
		Event:    event,
		Internal: internal,
		Data:     data,
	}
	requestData, _ := json.Marshal(notification)

	if len(requestData) > NS_MESSAGE_MAX_LEN {
		return errors.New("PayloadChannel request too big")
	}
	if len(payload) > NS_PAYLOAD_MAX_LEN {
		return errors.New("PayloadChannel payload too big")
	}

	return c.writeAll(requestData, payload)
}

func (c *PayloadChannel) Request(method string, internal internalData, data interface{}, payload []byte) (rsp workerResponse) {
	if c.Closed() {
		rsp.err = NewInvalidStateError("PayloadChannel closed")
		return
	}
	id := atomic.AddInt64(&c.nextId, 1)
	atomic.CompareAndSwapInt64(&c.nextId, 4294967295, 1)

	c.logger.Debug("request() [method:%s, id:%d]", method, id)

	request := workerRequest{
		Id:       id,
		Method:   method,
		Internal: internal,
		Data:     data,
	}
	requestData, _ := json.Marshal(request)

	if len(requestData) > NS_MESSAGE_MAX_LEN {
		return workerResponse{err: errors.New("PayloadChannel request too big")}
	}
	if len(payload) > NS_PAYLOAD_MAX_LEN {
		return workerResponse{err: errors.New("PayloadChannel payload too big")}
	}

	sent := sentInfo{
		method:      method,
		requestData: requestData,
		payloadData: payload,
		respCh:      make(chan workerResponse),
	}
	size := syncMapLen(c.sents)

	c.sents.Store(id, sent)
	defer c.sents.Delete(id)

	timeout := 1000 * (15 + (0.1 * float64(size)))
	timer := time.NewTimer(time.Duration(timeout) * time.Millisecond)
	defer timer.Stop()

	// send request
	select {
	case c.sentChan <- sent:
	case <-timer.C:
		rsp.err = errors.New("PayloadChannel request timeout")
	case <-c.closeCh:
		rsp.err = NewInvalidStateError("PayloadChannel closed")
	}
	if rsp.err != nil {
		return
	}

	// wait response
	select {
	case rsp = <-sent.respCh:
	case <-timer.C:
		rsp.err = errors.New("PayloadChannel request timeout")
	case <-c.closeCh:
		rsp.err = NewInvalidStateError("PayloadChannel closed")
	}
	return
}

func (c *PayloadChannel) runWriteLoop() {
	defer c.Close()

	for {
		select {
		case sentInfo := <-c.sentChan:
			if err := c.writeAll(sentInfo.requestData, sentInfo.payloadData); err != nil {
				sentInfo.respCh <- workerResponse{err: err}
				break
			}
		case <-c.closeCh:
			return
		}
	}
}

func (c *PayloadChannel) writeAll(data, payload []byte) (err error) {
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
	if notify := c.pendingNotification; c.pendingNotification != nil {
		c.pendingNotification = nil
		c.emit(notify.TargetId, notify.Event, notify.Data, payload)
		return
	}

	var msg struct {
		// response meta info
		Id       int64  `json:"id,omitempty"`
		Accepted bool   `json:"accepted,omitempty"`
		Error    string `json:"error,omitempty"`
		Reason   string `json:"reason,omitempty"`

		// notification meta info
		TargetId string `json:"targetId,omitempty"`
		Event    string `json:"event,omitempty"`

		// response or notification data
		Data json.RawMessage `json:"data,omitempty"`
	}
	if err := json.Unmarshal(payload, &msg); err != nil {
		c.logger.Error("received response unmarshal failed [id:%d], data: %s, err: %s", msg.Id, payload, err)
		return
	}

	if msg.Id > 0 {
		value, ok := c.sents.Load(msg.Id)
		if !ok {
			c.logger.Error("received response does not match any sent request [id:%d]", msg.Id)
			return
		}
		sent := value.(sentInfo)

		if msg.Accepted {
			c.logger.Debug("request succeeded [method:%s, id:%d]", sent.method, msg.Id)

			sent.respCh <- workerResponse{data: msg.Data}
		} else if len(msg.Error) > 0 {
			c.logger.Warn("request failed [method:%s, id:%d]: %s", sent.method, msg.Id, msg.Reason)

			if msg.Error == "TypeError" {
				sent.respCh <- workerResponse{err: NewTypeError(msg.Reason)}
			} else {
				sent.respCh <- workerResponse{err: errors.New(msg.Reason)}
			}
		} else {
			c.logger.Error("received response is not accepted nor rejected [method:%s, id:%s]", sent.method, msg.Id)
		}
	} else if len(msg.TargetId) > 0 && len(msg.Event) > 0 {
		c.pendingNotification = &notification{
			TargetId: msg.TargetId,
			Event:    msg.Event,
			Data:     msg.Data,
		}
	} else {
		c.logger.Error("received message is not a response nor a notification")
	}
}

// emit notifcation
func (c *PayloadChannel) emit(targetId, event string, data, payload []byte) {
	val, ok := c.listeners.Load(targetId)
	if !ok {
		c.logger.Warn("no listener on target: %s", targetId)
		return
	}
	listener := val.(*intervalListener)

	defer func() {
		if listener.once {
			c.listeners.Delete(targetId)
		}
		if r := recover(); r != nil {
			c.logger.Error("notification handler panic: %s, stack: %s", r, debug.Stack())
		}
	}()

	// may panic
	listener.Call(event, data, payload)
}
