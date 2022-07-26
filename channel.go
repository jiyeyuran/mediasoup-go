package mediasoup

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jiyeyuran/mediasoup-go/netcodec"
)

type Channel struct {
	logger    Logger
	codec     netcodec.Codec
	closed    int32
	pid       int
	nextId    int64
	sents     sync.Map
	listeners sync.Map
	sentChan  chan sentInfo
	closeCh   chan struct{}
}

func newChannel(codec netcodec.Codec, pid int) *Channel {
	logger := NewLogger("Channel")

	logger.Debug("constructor()")

	channel := &Channel{
		logger:   logger,
		codec:    codec,
		pid:      pid,
		sentChan: make(chan sentInfo),
		closeCh:  make(chan struct{}),
	}

	return channel
}

func (c *Channel) Start() {
	go c.runWriteLoop()
	go c.runReadLoop()
}

func (c *Channel) Close() error {
	if atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		c.logger.Debug("close()")
		close(c.closeCh)
		return c.codec.Close()
	}
	return nil
}

func (c *Channel) Closed() bool {
	return atomic.LoadInt32(&c.closed) > 0
}

func (c *Channel) AddTargetHandler(targetId string, handler interface{}, options ...listenerOption) {
	c.listeners.Store(targetId, newInternalListener(handler, options...))
}

func (c *Channel) RemoveTargetHandler(targetId string) {
	c.listeners.Delete(targetId)
}

func (c *Channel) Request(method string, internal internalData, data ...interface{}) (rsp workerResponse) {
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
	}
	if len(data) > 0 {
		request.Data = data[0]
	}
	requestData, _ := json.Marshal(request)

	if len(requestData) > NS_MESSAGE_MAX_LEN {
		return workerResponse{err: errors.New("Channel request too big")}
	}

	sent := sentInfo{
		method:      method,
		requestData: requestData,
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
		rsp.err = errors.New("Channel request timeout")
	case <-c.closeCh:
		rsp.err = NewInvalidStateError("Channel closed")
	}
	if rsp.err != nil {
		return
	}

	// wait response
	select {
	case rsp = <-sent.respCh:
	case <-timer.C:
		rsp.err = errors.New("Channel request timeout")
	case <-c.closeCh:
		rsp.err = NewInvalidStateError("Channel closed")
	}
	return
}

func (c *Channel) runWriteLoop() {
	defer c.Close()

	for {
		select {
		case sentInfo := <-c.sentChan:
			if err := c.codec.WritePayload(sentInfo.requestData); err != nil {
				sentInfo.respCh <- workerResponse{err: err}
				break
			}
		case <-c.closeCh:
			return
		}
	}
}

func (c *Channel) runReadLoop() {
	defer c.Close()

	for {
		payload, err := c.codec.ReadPayload()
		if err != nil {
			c.logger.Error("Channel error: %s", err)
			break
		}
		c.processPayload(payload)
	}
}

func (c *Channel) processPayload(nsPayload []byte) {
	switch nsPayload[0] {
	case '{':
		c.processMessage(nsPayload)
	case 'D':
		c.logger.Debug("[pid:%d] %s", c.pid, nsPayload[1:])
	case 'W':
		c.logger.Warn("[pid:%d] %s", c.pid, nsPayload[1:])
	case 'E':
		c.logger.Error("[pid:%d] %s", c.pid, nsPayload[1:])
	case 'X':
		fmt.Printf("%s\n", nsPayload[1:])
	default:
		c.logger.Warn("[pid:%d] unexpected data: %s", nsPayload[1:])
	}
}

func (c *Channel) processMessage(nsPayload []byte) {
	var msg struct {
		// response
		Id       int64  `json:"id,omitempty"`
		Accepted bool   `json:"accepted,omitempty"`
		Error    string `json:"error,omitempty"`
		Reason   string `json:"reason,omitempty"`
		// notification
		TargetId interface{} `json:"targetId,omitempty"`
		Event    string      `json:"event,omitempty"`
		// common data
		Data json.RawMessage `json:"data,omitempty"`
	}
	if err := json.Unmarshal(nsPayload, &msg); err != nil {
		c.logger.Error("received response, failed to unmarshal to json: %s", err)
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
	} else if msg.TargetId != nil && len(msg.Event) > 0 {
		var targetId string
		// The type of msg.TargetId should be string or float64
		switch v := msg.TargetId.(type) {
		case string:
			targetId = v
		case float64:
			targetId = fmt.Sprintf("%0.0f", v)
		default:
			targetId = fmt.Sprintf("%v", v)
		}
		c.emit(targetId, msg.Event, msg.Data)
	} else {
		c.logger.Error("received message is not a response nor a notification")
	}
}

// emit notifcation
func (c *Channel) emit(targetId, event string, data []byte) {
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
	listener.Call(event, data)
}
