package mediasoup

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/jiyeyuran/mediasoup-go/mediasoup/netstring"
	"github.com/sirupsen/logrus"
)

const (
	// netstring length for a 4194304 bytes payload.
	NS_MESSAGE_MAX_LEN = 4194313
	NS_PAYLOAD_MAX_LEN = 4194304
)

type sentInfo struct {
	id         int64
	method     string
	responseCh chan Response
}

type Channel struct {
	EventEmitter
	socket       net.Conn
	logger       logrus.FieldLogger
	workerLogger logrus.FieldLogger
	closed       bool
	nextId       int64
	sents        map[int64]sentInfo
	closeCh      chan struct{}
}

func NewChannel(socket net.Conn, pid int) *Channel {
	logger := TypeLogger(fmt.Sprintf("Channel[pid:%d]", pid))
	workerLogger := TypeLogger(fmt.Sprintf("worker[pid:%d]", pid))

	channel := &Channel{
		EventEmitter: NewEventEmitter(logger),
		socket:       socket,
		logger:       logger,
		workerLogger: workerLogger,
		sents:        make(map[int64]sentInfo),
		closeCh:      make(chan struct{}),
	}

	go channel.runReadLoop()

	logger.Debugln("constructor()")

	return channel
}

func (c *Channel) Close() {
	if c.closed {
		return
	}

	c.logger.Debugln("close()")

	c.socket.Close()

	c.closed = true
}

func (c *Channel) Request(
	method string,
	internal interface{},
	data ...interface{},
) (rsp Response) {
	if c.nextId < 4294967295 {
		c.nextId++
	} else {
		c.nextId = 1
	}

	id := c.nextId

	c.logger.Debugf("request() [method:%s, id:%d]", method, id)

	if c.closed {
		rsp.err = NewInvalidStateError("Channel closed")
		return
	}

	sent := sentInfo{
		id:         id,
		method:     method,
		responseCh: make(chan Response),
	}
	c.sents[id] = sent

	defer delete(c.sents, id)

	req := struct {
		Id       int64       `json:"id"`
		Method   string      `json:"method,omitempty"`
		Internal interface{} `json:"internal,omitempty"`
		Data     interface{} `json:"data,omitempty"`
	}{
		Id:       id,
		Method:   method,
		Internal: internal,
	}
	if len(data) > 0 {
		req.Data = data[0]
	}
	rawData, _ := json.Marshal(req)

	ns := netstring.Encode(rawData)
	if len(ns) > NS_MESSAGE_MAX_LEN {
		rsp.err = errors.New("Channel request too big")
		return
	}

	_, rsp.err = c.socket.Write(ns)
	if rsp.err != nil {
		return
	}

	timeout := 1000 * (15 + (0.1 * float64(len(c.sents))))
	timer := time.NewTimer(time.Duration(timeout) * time.Millisecond)
	defer timer.Stop()

	select {
	case rsp = <-sent.responseCh:
		return
	case <-timer.C:
		rsp.err = errors.New("Channel request timeout")
	case <-c.closeCh:
		rsp.err = errors.New("Channel closed")
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
		n, err := c.socket.Read(buf)
		if err != nil {
			c.logger.Errorf("Channel error: %s", err)
			break
		}
		data := buf[:n]

		decoder.Feed(data)

		if decoder.Length() > NS_PAYLOAD_MAX_LEN {
			c.logger.Errorln("receiving buffer is full, discarding all data into it")
			decoder.Reset()
		}
	}

	c.closed = true
	close(c.closeCh)
}

func (c *Channel) processNSPayload(nsPayload []byte) {
	switch nsPayload[0] {
	case '{':
		c.processMessage(nsPayload)
	case 'D':
		c.workerLogger.Debugf("%s", nsPayload)
	case 'W':
		c.workerLogger.Warnf("%s", nsPayload)
	case 'E':
		c.workerLogger.Errorf("%s", nsPayload)
	default:
		c.workerLogger.Errorf("unexpected data: %s", nsPayload)
	}
}

func (c *Channel) processMessage(nsPayload []byte) {
	var msg struct {
		Id       int64
		TargetId string
	}
	json.Unmarshal(nsPayload, &msg)

	if msg.Id > 0 {
		sent, ok := c.sents[msg.Id]
		if !ok {
			c.logger.Errorf("received response does not match any sent request [id:%d]", msg.Id)
			return
		}
		var msg struct {
			Id       int64
			Accepted bool
			Data     json.RawMessage
			Error    string
			Reason   string
		}
		json.Unmarshal(nsPayload, &msg)

		if msg.Accepted {
			c.logger.Debugf("request succeeded [method:%s, id:%d]", sent.method, sent.id)

			sent.responseCh <- Response{data: msg.Data}
		} else if len(msg.Error) > 0 {
			c.logger.Warnf("request failed [method:%s, id:%d]: %s",
				sent.method, sent.id, msg.Reason)

			sent.responseCh <- Response{err: NewTypeError(msg.Reason)}
		}
	} else if len(msg.TargetId) > 0 {
		var notification struct {
			TargetId string
			Event    string
			Data     json.RawMessage
		}
		json.Unmarshal(nsPayload, &notification)

		go c.SafeEmit(notification.TargetId, notification.Event, notification.Data)
	} else {
		c.logger.Errorln("received message is not a response nor a notification")
	}
}
