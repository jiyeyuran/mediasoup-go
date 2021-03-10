package mediasoup

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jiyeyuran/mediasoup-go/netstring"
)

const (
	// netstring length for a 4194304 bytes payload.
	NS_MESSAGE_MAX_LEN = 4194313
	NS_PAYLOAD_MAX_LEN = 4194304
)

// Response from worker
type workerResponse struct {
	data json.RawMessage
	err  error
}

func (r workerResponse) Unmarshal(v interface{}) error {
	if r.err != nil {
		return r.err
	}
	if len(r.data) == 0 {
		return nil
	}
	return json.Unmarshal([]byte(r.data), v)
}

func (r workerResponse) Data() []byte {
	return []byte(r.data)
}

func (r workerResponse) Err() error {
	return r.err
}

type sentInfo struct {
	id     int64
	method string
	respCh chan workerResponse
}

type Channel struct {
	IEventEmitter
	logger         Logger
	closed         int32
	producerSocket net.Conn
	consumerSocket net.Conn
	pid            int
	nextId         int64
	sents          sync.Map
	sentsLen       int64
	closeCh        chan struct{}
}

func newChannel(producerSocket, consumerSocket net.Conn, pid int) *Channel {
	logger := NewLogger("Channel")

	logger.Debug("constructor()")

	channel := &Channel{
		IEventEmitter:  NewEventEmitter(),
		logger:         logger,
		producerSocket: producerSocket,
		consumerSocket: consumerSocket,
		pid:            pid,
		closeCh:        make(chan struct{}),
	}

	go channel.runReadLoop()

	return channel
}

func (c *Channel) Close() {
	if atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		c.logger.Debug("close()")

		c.producerSocket.Close()
		c.consumerSocket.Close()

		close(c.closeCh)
		c.RemoveAllListeners()
	}
}

func (c *Channel) Closed() bool {
	return atomic.LoadInt32(&c.closed) > 0
}

func (c *Channel) Request(method string, internal interface{}, data ...interface{}) (rsp workerResponse) {
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

	req := H{
		"id":       id,
		"method":   method,
		"internal": internal,
	}
	if len(data) > 0 {
		req["data"] = data[0]
	}
	rawData, _ := json.Marshal(req)

	ns := netstring.Encode(rawData)

	if len(ns) > NS_MESSAGE_MAX_LEN {
		rsp.err = errors.New("Channel request too big")
		return
	}

	if _, rsp.err = c.producerSocket.Write(ns); rsp.err != nil {
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

func (c *Channel) runReadLoop() {
	decoder := netstring.NewDecoder()

	go func() {
		for {
			select {
			case nsPayload := <-decoder.Result():
				c.processNSPayload(nsPayload)
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

func (c *Channel) processNSPayload(nsPayload []byte) {
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
		TargetId string `json:"targetId,omitempty"`
		Event    string `json:"event,omitempty"`
		// common data
		Data json.RawMessage `json:"data,omitempty"`
	}
	json.Unmarshal(nsPayload, &msg)

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
		c.SafeEmit(msg.TargetId, msg.Event, msg.Data)
	} else {
		c.logger.Error("received message is not a response nor a notification")
	}
}
