package mediasoup

type DirectTransportOptions struct {
	/**
	 * Maximum allowed size for direct messages sent from DataProducers.
	 * Default 262144.
	 */
	MaxMessageSize uint32 `json:"maxMessageSize,omitempty"`

	/**
	 * Custom application data.
	 */
	AppData interface{} `json:"appData,omitempty"`
}

type directTransportData struct{}

/**
 * DirectTransport
 * @emits rtcp - (packet: []byte)
 * @emits trace - (trace: TransportTraceEventData)
 */
type DirectTransport struct {
	ITransport
	logger         Logger
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

/**
 * Observer.
 *
 * @override
 * @emits close
 * @emits newdataproducer - (dataProducer: DataProducer)
 * @emits newdataconsumer - (dataProducer: DataProducer)
 * @emits trace - (trace: TransportTraceEventData)
 */
func (transport *DirectTransport) Observer() IEventEmitter {
	return transport.ITransport.Observer()
}

/**
 * NO-OP method in DirectTransport.
 *
 * @override
 */
func (transport *DirectTransport) Connect(TransportConnectOptions) error {
	transport.logger.Debug("connect()")

	return nil
}

/**
 * @override
 */
func (transport *DirectTransport) setMaxIncomingBitrate(bitrate int) error {
	return NewUnsupportedError("setMaxIncomingBitrate() not implemented in DirectTransport")
}

/**
 * Send RTCP packet.
 */
func (transport *DirectTransport) SendRtcp(rtcpPacket []byte) error {
	return transport.payloadChannel.Notify("transport.sendRtcp", transport.internal, nil, rtcpPacket)
}

func (transport *DirectTransport) handleWorkerNotifications() {
	transport.channel.On(transport.Id(), func(event string, data TransportTraceEventData) {
		switch event {
		case "trace":
			transport.SafeEmit("trace", data)

			// Emit observer event.
			transport.Observer().SafeEmit("trace", data)

		default:
			transport.logger.Error(`ignoring unknown event "%s" in channel listener`, event)
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
			transport.logger.Error(`ignoring unknown event "%s" in payload channel listener`, event)
		}
	})
}
