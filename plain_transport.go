package mediasoup

import "encoding/json"

type PlainTransportOptions struct {
	/**
	 * Listening IP address.
	 */
	ListenIp TransportListenIp

	/**
	 * Use RTCP-mux (RTP and RTCP in the same port). Default true.
	 */
	RtcpMux bool

	/**
	 * Whether remote IP:port should be auto-detected based on first RTP/RTCP
	 * packet received. If enabled, connect() method must not be called unless
	 * SRTP is enabled. If so, it must be called with just remote SRTP parameters.
	 * Default false.
	 */
	Comedia bool

	/**
	 * Create a SCTP association. Default false.
	 */
	EnableSctp bool

	/**
	 * SCTP streams number.
	 */
	NumSctpStreams NumSctpStreams

	/**
	 * Maximum allowed size for SCTP messages sent by DataProducers.
	 * Default 262144.
	 */
	MaxSctpMessageSize int

	/**
	 * Maximum SCTP send buffer used by DataConsumers.
	 * Default 262144.
	 */
	SctpSendBufferSize int

	/**
	 * Enable SRTP. For this to work, connect() must be called
	 * with remote SRTP parameters. Default false.
	 */
	EnableSrtp bool

	/**
	 * The SRTP crypto suite to be used if enableSrtp is set. Default
	 * 'AES_CM_128_HMAC_SHA1_80'.
	 */
	SrtpCryptoSuite SrtpCryptoSuite

	/**
	 * Custom application data.
	 */
	AppData interface{}
}

type PlainTransportSpecificStat struct {
	RtcpMux   bool            `json:"rtcp_mux"`
	Comedia   bool            `json:"comedia"`
	Tuple     TransportTuple  `json:"tuple"`
	RtcpTuple *TransportTuple `json:"rtcpTuple,omitempty"`
}

type plainTransportData struct {
	rtcpMux        bool
	comedia        bool
	tuple          TransportTuple
	rtcpTuple      TransportTuple
	sctpParameters SctpParameters
	sctpState      SctpState
	srtpParameters SrtpParameters
}

/**
 * PlainTransport
 * @private
 * @emits tuple - (tuple: TransportTuple)
 * @emits rtcptuple - (rtcpTuple: TransportTuple)
 * @emits sctpstatechange - (sctpState: SctpState)
 * @emits trace - (trace: TransportTraceEventData)
 */
type PlainTransport struct {
	ITransport
	logger   Logger
	internal internalData
	data     plainTransportData
	channel  *Channel
}

func newPlainTransport(params transportParams, data plainTransportData) *PlainTransport {
	params.logger = NewLogger("PlainTransport")
	params.data = transportData{
		sctpParameters: data.sctpParameters,
		sctpState:      data.sctpState,
	}

	transport := &PlainTransport{
		ITransport: newTransport(params),
		logger:     params.logger,
		internal:   params.internal,
		data:       data,
	}

	return transport
}

/**
 * Transport tuple.
 */
func (t PlainTransport) Tuple() TransportTuple {
	return t.data.tuple
}

/**
 * Transport RTCP tuple.
 */
func (t PlainTransport) RtcpTuple() TransportTuple {
	return t.data.rtcpTuple
}

/**
 * SCTP parameters.
 */
func (t PlainTransport) SctpParameters() SctpParameters {
	return t.data.sctpParameters
}

/**
 * SCTP state.
 */
func (t PlainTransport) SctpState() SctpState {
	return t.data.sctpState
}

/**
 * SRTP parameters.
 */
func (t PlainTransport) SrtpParameters() SrtpParameters {
	return t.data.srtpParameters
}

/**
 * Observer.
 *
 * @override
 * @emits close
 * @emits newproducer - (producer: Producer)
 * @emits newconsumer - (consumer: Consumer)
 * @emits newdataproducer - (dataProducer: DataProducer)
 * @emits newdataconsumer - (dataConsumer: DataConsumer)
 * @emits tuple - (tuple: TransportTuple)
 * @emits rtcptuple - (rtcpTuple: TransportTuple)
 * @emits sctpstatechange - (sctpState: SctpState)
 * @emits trace - (trace: TransportTraceEventData)
 */
func (transport *PlainTransport) Observer() IEventEmitter {
	return transport.ITransport.Observer()
}

/**
 * Close the PlainTransport.
 *
 * @override
 */
func (transport *PlainTransport) Close() {
	if transport.Closed() {
		return
	}

	if len(transport.data.sctpState) > 0 {
		transport.data.sctpState = SctpState_Closed
	}

	transport.ITransport.Close()
}

/**
 * Router was closed.
 *
 * @override
 */
func (transport *PlainTransport) routerClosed() {
	if transport.Closed() {
		return
	}

	if len(transport.data.sctpState) > 0 {
		transport.data.sctpState = SctpState_Closed
	}

	transport.ITransport.routerClosed()
}

/**
 * Provide the PlainTransport remote parameters.
 *
 * @override
 */
func (transport *PlainTransport) Connect(options TransportConnectOptions) (err error) {
	transport.logger.Debug("connect()")

	resp := transport.channel.Request("transport.connect", transport.internal, options)

	var data struct {
		Tuple          *TransportTuple
		RtcpTuple      *TransportTuple
		SrtpParameters SrtpParameters
	}
	if err = resp.Unmarshal(&data); err != nil {
		return
	}

	// Update data.
	if data.Tuple != nil {
		transport.data.tuple = *data.Tuple
	}
	if data.RtcpTuple != nil {
		transport.data.rtcpTuple = *data.RtcpTuple
	}

	transport.data.srtpParameters = data.SrtpParameters

	return nil
}

func (transport *PlainTransport) handleWorkerNotifications() {
	type eventInfo struct {
		Tuple     TransportTuple
		RtcpTuple TransportTuple
		SctpState SctpState
	}

	transport.channel.On(transport.Id(), func(event string, data []byte) {
		switch event {
		case "tuple":
			var result eventInfo
			json.Unmarshal(data, &result)

			transport.data.tuple = result.Tuple

			transport.SafeEmit("tuple", result.Tuple)

			// Emit observer event.
			transport.Observer().SafeEmit("tuple", result.Tuple)

		case "rtcptuple":
			var result eventInfo
			json.Unmarshal(data, &result)

			transport.data.rtcpTuple = result.RtcpTuple

			transport.SafeEmit("rtcptuple", result.RtcpTuple)

			// Emit observer event.
			transport.Observer().SafeEmit("rtcptuple", result.RtcpTuple)

		case "sctpstatechange":
			var result eventInfo
			json.Unmarshal(data, &result)

			transport.data.sctpState = result.SctpState

			transport.SafeEmit("sctpstatechange", result.SctpState)

			// Emit observer event.
			transport.Observer().SafeEmit("sctpstatechange", result.SctpState)

		case "trace":
			var result TransportTraceEventData
			json.Unmarshal(data, &result)

			transport.SafeEmit("trace", result)

			// Emit observer event.
			transport.Observer().SafeEmit("trace", result)

		default:
			transport.logger.Error(`ignoring unknown event "%s"`, event)
		}
	})
}
