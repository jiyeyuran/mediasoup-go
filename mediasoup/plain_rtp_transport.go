package mediasoup

import (
	"errors"

	"github.com/sirupsen/logrus"
)

type PlainRtpTransport struct {
	*baseTransport
	logger logrus.FieldLogger
	data   PlainTransportData
}

func NewPlainRtpTransport(data PlainTransportData, params CreateTransportParams) Transport {
	logger := TypeLogger("PlainRtpTransport")

	logger.Debug("constructor()")

	return &PlainRtpTransport{
		baseTransport: newTransport(params),
		logger:        logger,
		data:          data,
	}
}

func (t PlainRtpTransport) Tuple() TransportTuple {
	return t.data.Tuple
}

func (t PlainRtpTransport) RtcpTuple() TransportTuple {
	return t.data.RtcpTuple
}

/**
 * Provide the PlainRtpTransport remote parameters.
 *
 * @param {String} ip - Remote IP.
 * @param {Number} port - Remote port.
 * @param {Number} [rtcpPort] - Remote RTCP port (ignored if rtcpMux was true).
 *
 * @override
 */
func (t *PlainRtpTransport) Connect(params TransportConnectParams) (err error) {
	t.logger.Debug("connect()")

	resp := t.channel.Request("transport.connect", t.internal, params)

	return resp.Result(&t.data)
}

/**
 * Override Transport.consume() method to reject it if multiSource is set.
 *
 * @override
 * @returns {Consumer}
 */
func (t *PlainRtpTransport) Consume(params TransportConsumeParams) (*Consumer, error) {
	if t.data.MultiSource {
		return nil, errors.New("cannot call consume() with multiSource set")
	}

	return t.baseTransport.Consume(params)
}
