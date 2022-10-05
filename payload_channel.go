package mediasoup

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
	"github.com/jiyeyuran/mediasoup-go/netcodec"
)

type notifyInfo struct {
	requestData []byte
	payloadData []byte
	respCh      chan workerResponse
}

type PayloadChannel struct {
	IEventEmitter
	locker              sync.Mutex
	codec               netcodec.Codec
	logger              logr.Logger
	closed              int32
	nextId              int64
	sents               sync.Map
	pendingNotification *notification
	sentChan            chan sentInfo
	closeCh             chan struct{}
	useHandlerID        bool
}

func newPayloadChannel(codec netcodec.Codec, useHandlerID bool) *PayloadChannel {
	logger := NewLogger("PayloadChannel")

	logger.V(1).Info("constructor()")

	channel := &PayloadChannel{
		IEventEmitter: NewEventEmitter(),
		logger:        logger,
		codec:         codec,
		sentChan:      make(chan sentInfo),
		closeCh:       make(chan struct{}),
		useHandlerID:  useHandlerID,
	}

	return channel
}

func (c *PayloadChannel) Start() {
	go c.runWriteLoop()
	go c.runReadLoop()
}

func (c *PayloadChannel) Close() error {
	if atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		c.logger.V(1).Info("close()")
		close(c.closeCh)
		return c.codec.Close()
	}
	return nil
}

func (c *PayloadChannel) Closed() bool {
	return atomic.LoadInt32(&c.closed) > 0
}

func (c *PayloadChannel) Notify(event string, internal internalData, data string, payload []byte) (err error) {
	if c.Closed() {
		return NewInvalidStateError("PayloadChannel closed")
	}

	var request []byte

	if c.useHandlerID {
		request = []byte(fmt.Sprintf("n:%s:%s:%s", event, internal.HandlerID(event), data))
	} else {
		rawData, _ := json.Marshal(H{"ppid": data})
		notification := workerNotification{
			Event:    event,
			Internal: internal,
			Data:     rawData,
		}
		request, _ = json.Marshal(notification)
	}

	if len(request) > NS_MESSAGE_MAX_LEN {
		return errors.New("PayloadChannel notification too big")
	}
	if len(payload) > NS_PAYLOAD_MAX_LEN {
		return errors.New("PayloadChannel payload too big")
	}

	return c.writeAll(request, payload)
}

func (c *PayloadChannel) Request(method string, internal internalData, data string, payload []byte) (rsp workerResponse) {
	if c.Closed() {
		rsp.err = NewInvalidStateError("PayloadChannel closed")
		return
	}
	id := atomic.AddInt64(&c.nextId, 1)
	atomic.CompareAndSwapInt64(&c.nextId, 4294967295, 1)

	c.logger.V(1).Info("request()", "method", method, "id", id)

	var request []byte

	if c.useHandlerID {
		handlerID := internal.HandlerID(method)
		request = []byte(fmt.Sprintf("r:%d:%s:%s:%s", id, method, handlerID, data))
	} else {
		rawData, _ := json.Marshal(H{"ppid": data})
		request, _ = json.Marshal(workerRequest{
			Id:       id,
			Method:   method,
			Internal: internal,
			Data:     rawData,
		})
	}

	if len(request) > NS_MESSAGE_MAX_LEN {
		return workerResponse{err: errors.New("PayloadChannel request too big")}
	}
	if len(payload) > NS_PAYLOAD_MAX_LEN {
		return workerResponse{err: errors.New("PayloadChannel payload too big")}
	}

	sent := sentInfo{
		method:  method,
		request: request,
		payload: payload,
		respCh:  make(chan workerResponse),
	}
	c.sents.Store(id, sent)
	defer c.sents.Delete(id)

	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()

	// send request
	select {
	case c.sentChan <- sent:
	case <-timer.C:
		rsp.err = fmt.Errorf("PayloadChannel request timeout, id: %d, method: %s", id, method)
	case <-c.closeCh:
		rsp.err = NewInvalidStateError("PayloadChannel closed, id: %d, method: %s", id, method)
	}
	if rsp.err != nil {
		return
	}

	// wait response
	select {
	case rsp = <-sent.respCh:
	case <-timer.C:
		rsp.err = fmt.Errorf("PayloadChannel response timeout, id: %d, method: %s", id, method)
	case <-c.closeCh:
		rsp.err = NewInvalidStateError("PayloadChannel closed, id: %d, method: %s", id, method)
	}
	return
}

func (c *PayloadChannel) runWriteLoop() {
	defer c.Close()

	for {
		select {
		case sentInfo := <-c.sentChan:
			if err := c.writeAll(sentInfo.request, sentInfo.payload); err != nil {
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
			c.logger.Error(err, "PayloadChannel error failed")
			break
		}
		c.processPayload(payload)
	}
}

func (c *PayloadChannel) processPayload(payload []byte) {
	if notify := c.pendingNotification; notify != nil {
		c.pendingNotification = nil
		go c.SafeEmit(notify.TargetId, notify.Event, notify.Data, payload)
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
		c.logger.Error(err, "received response unmarshal failed", "id", msg.Id, "payload", payload)
		return
	}

	if msg.Id > 0 {
		value, ok := c.sents.Load(msg.Id)
		if !ok {
			c.logger.Error(nil, "received response does not match any sent request", "id", msg.Id)
			return
		}
		sent := value.(sentInfo)

		if msg.Accepted {
			c.logger.V(1).Info("request succeeded", "method", sent.method, "id", msg.Id)

			sent.respCh <- workerResponse{data: msg.Data}
		} else if len(msg.Error) > 0 {
			c.logger.Error(errors.New(msg.Reason), "request failed", "method", sent.method, "id", msg.Id)

			if msg.Error == "TypeError" {
				sent.respCh <- workerResponse{err: NewTypeError(msg.Reason)}
			} else {
				sent.respCh <- workerResponse{err: errors.New(msg.Reason)}
			}
		} else {
			c.logger.Error(nil, "received response is not accepted nor rejected", "method", sent.method, "id", msg.Id)
		}
	} else if len(msg.TargetId) > 0 && len(msg.Event) > 0 {
		c.pendingNotification = &notification{
			TargetId: msg.TargetId,
			Event:    msg.Event,
			Data:     msg.Data,
		}
	} else {
		c.logger.Error(nil, "received message is not a response nor a notification")
	}
}
