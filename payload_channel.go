package mediasoup

import "net"

type PayloadChannel struct {
	EventEmitter
}

func newPayloadChannel(producerSocket, consumerSocket net.Conn, pid int) *PayloadChannel {
	return &PayloadChannel{}
}

func (c *PayloadChannel) Close() {

}
