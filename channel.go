package mediasoup

import (
	"net"
)

type Channel struct {
	EventEmitter
}

func NewChannel(producerSocket, consumerSocket net.Conn, pid int) *Channel {
	return &Channel{}
}

func (c *Channel) Close() {

}

func (c *Channel) Request(method string, internal interface{}, data ...interface{}) (rsp Response) {
	return
}
