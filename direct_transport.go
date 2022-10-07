package mediasoup

import "github.com/go-logr/logr"

// DirectTransportOptions define options to create a DirectTransport.
type DirectTransportOptions struct {
	// MaxMessageSize define maximum allowed size for direct messages sent from DataProducers.
	// Default 262144.
	MaxMessageSize uint32 `json:"maxMessageSize,omitempty"`

	// AppData is custom application data.
	AppData interface{} `json:"appData,omitempty"`
}

type directTransportData struct{}

// DirectTransport represents a direct connection between the mediasoup golang process and a Router
// instance in a mediasoup-worker subprocess.
//
// A direct transport can be used to directly send and receive data messages from/to golang by means
// of DataProducers and DataConsumers of type 'direct' created on a direct transport. Direct
// messages sent by a DataProducer in a direct transport can be consumed by endpoints connected
// through a SCTP capable transport (WebRtcTransport, PlainTransport, PipeTransport) and also by the
// golang application by means of a DataConsumer created on a DirectTransport (and vice-versa:
// messages sent over SCTP/DataChannel can be consumed by the golang application by means of a
// DataConsumer created on a DirectTransport).

// A direct transport can also be used to inject and directly consume RTP and RTCP packets in golang
// by using the producer.Send(rtpPacket) and consumer.On('rtp') API (plus
// directTransport.SendRtcp(rtcpPacket) and directTransport.On('rtcp') API).
//
// - @emits rtcp - (packet []byte)
// - @emits trace - (trace *TransportTraceEventData)
type DirectTransport struct {
	ITransport
	logger         logr.Logger
	internal       internalData
	channel        *Channel
	payloadChannel *PayloadChannel
}

func newDirectTransport(params transportParams) ITransport {
	params.data = transportData{
		transportType: TransportType_Direct,
	}
	params.logger = NewLogger("DirectTransport")

	transport := &DirectTransport{
		ITransport:     newTransport(params),
		logger:         params.logger,
		internal:       params.internal,
		channel:        params.channel,
		payloadChannel: params.payloadChannel,
	}

	transport.handleWorkerNotifications()

	return transport
}

// Observer.
//
// - @emits close
// - @emits newdataproducer - (dataProducer *DataProducer)
// - @emits newdataconsumer - (dataProducer *DataProducer)
// - @emits rtcp - (packet []byte)
// - @emits trace - (trace: *TransportTraceEventData)
func (transport *DirectTransport) Observer() IEventEmitter {
	return transport.ITransport.Observer()
}

// Connect is a NO-OP method in DirectTransport.
func (transport *DirectTransport) Connect(TransportConnectOptions) error {
	transport.logger.V(1).Info("connect()")
	return nil
}

// SetMaxIncomingBitrate always returns error.
func (transport *DirectTransport) SetMaxIncomingBitrate(bitrate int) error {
	return NewUnsupportedError("SetMaxIncomingBitrate() not implemented in DirectTransport")
}

// SendRtcp send RTCP packet.
func (transport *DirectTransport) SendRtcp(rtcpPacket []byte) error {
	return transport.payloadChannel.Notify("transport.sendRtcp", transport.internal, "", rtcpPacket)
}

func (transport *DirectTransport) handleWorkerNotifications() {
	transport.channel.On(transport.Id(), func(event string, data *TransportTraceEventData) {
		switch event {
		case "trace":
			transport.SafeEmit("trace", data)

			// Emit observer event.
			transport.Observer().SafeEmit("trace", data)

		default:
			transport.logger.Error(nil, "ignoring unknown event in channel listener", "event", event)
		}
	})

	transport.payloadChannel.On(transport.Id(), func(event string, data, payload []byte) {
		switch event {
		case "rtcp":
			if transport.Closed() {
				return
			}

			transport.SafeEmit("rtcp", payload)

			// Emit observer event.
			transport.Observer().SafeEmit("rtcp", payload)

		default:
			transport.logger.Error(nil, "ignoring unknown event in payload channel listener", "event", event)
		}
	})
}
