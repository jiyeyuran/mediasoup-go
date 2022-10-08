package mediasoup

import (
	"encoding/json"

	"github.com/go-logr/logr"
)

// PlainTransportOptions define options to create a PlainTransport
type PlainTransportOptions struct {
	// ListenIp define Listening IP address.
	ListenIp TransportListenIp `json:"listenIp,omitempty"`

	// RtcpMux define wether use RTCP-mux (RTP and RTCP in the same port). Default true.
	RtcpMux *bool `json:"rtcpMux,omitempty"`

	// Comedia define whether remote IP:port should be auto-detected based on first RTP/RTCP
	// packet received. If enabled, connect() method must not be called unless
	// SRTP is enabled. If so, it must be called with just remote SRTP parameters.
	// Default false.
	Comedia bool `json:"comedia,omitempty"`

	// EnableSctp define whether create a SCTP association. Default false.
	EnableSctp bool `json:"enableSctp,omitempty"`

	// NumSctpStreams define SCTP streams number.
	NumSctpStreams NumSctpStreams `json:"numSctpStreams,omitempty"`

	// MaxSctpMessageSize define maximum allowed size for SCTP messages sent by DataProducers.
	// Default 262144.
	MaxSctpMessageSize int `json:"maxSctpMessageSize,omitempty"`

	// SctpSendBufferSize define maximum SCTP send buffer used by DataConsumers.
	// Default 262144.
	SctpSendBufferSize int `json:"sctpSendBufferSize,omitempty"`

	// EnableSrtp enable SRTP. For this to work, connect() must be called
	// with remote SRTP parameters. Default false.
	EnableSrtp bool `json:"enableSrtp,omitempty"`

	// SrtpCryptoSuite define the SRTP crypto suite to be used if enableSrtp is set. Default
	// 'AES_CM_128_HMAC_SHA1_80'.
	SrtpCryptoSuite SrtpCryptoSuite `json:"srtpCryptoSuite,omitempty"`

	// AppData is custom application data.
	AppData interface{} `json:"appData,omitempty"`
}

// PlainTransportSpecificStat define the stat info for PlainTransport
type PlainTransportSpecificStat struct {
	RtcpMux   bool            `json:"rtcp_mux"`
	Comedia   bool            `json:"comedia"`
	Tuple     TransportTuple  `json:"tuple"`
	RtcpTuple *TransportTuple `json:"rtcpTuple,omitempty"`
}

type plainTransportData struct {
	RtcpMux        bool            `json:"rtcp_mux,omitempty"`
	Comedia        bool            `json:"comedia,omitempty"`
	Tuple          *TransportTuple `json:"tuple,omitempty"`
	RtcpTuple      *TransportTuple `json:"rtcpTuple,omitempty"`
	SctpParameters SctpParameters  `json:"sctpParameters,omitempty"`
	SctpState      SctpState       `json:"sctpState,omitempty"`
	SrtpParameters *SrtpParameters `json:"srtpParameters,omitempty"`
}

// PlainTransport represents a network path through which RTP, RTCP (optionally secured with SRTP)
// and SCTP = data.hannel is transmitted.
//
// - @emits tuple - (tuple *TransportTuple)
// - @emits rtcptuple - (rtcpTuple *TransportTuple)
// - @emits sctpstatechange - (sctpState SctpState)
// - @emits trace - (trace *TransportTraceEventData)
type PlainTransport struct {
	ITransport
	logger   logr.Logger
	internal internalData
	data     *plainTransportData
	channel  *Channel
}

func newPlainTransport(params transportParams) ITransport {
	data := params.data.(*plainTransportData)
	params.data = transportData{
		sctpParameters: data.SctpParameters,
		sctpState:      data.SctpState,
		transportType:  TransportType_Plain,
	}
	params.logger = NewLogger("PlainTransport")

	transport := &PlainTransport{
		ITransport: newTransport(params),
		logger:     params.logger,
		internal:   params.internal,
		data:       data,
		channel:    params.channel,
	}

	transport.handleWorkerNotifications()

	return transport
}

// Tuple returns transport tuple.
func (t PlainTransport) Tuple() *TransportTuple {
	return t.data.Tuple
}

// RtcpTuple returns transport RTCP tuple.
func (t PlainTransport) RtcpTuple() *TransportTuple {
	return t.data.RtcpTuple
}

// SctpParameters returns SCTP parameters.
func (t PlainTransport) SctpParameters() SctpParameters {
	return t.data.SctpParameters
}

// SctpState returns SCTP state.
func (t PlainTransport) SctpState() SctpState {
	return t.data.SctpState
}

// SrtpParameters returns SRTP parameters.
func (t PlainTransport) SrtpParameters() *SrtpParameters {
	return t.data.SrtpParameters
}

// Observer.
//
// - @emits close
// - @emits newproducer - (producer *Producer)
// - @emits newconsumer - (consumer *Consumer)
// - @emits newdataproducer - = data.roducer *DataProducer
// - @emits newdataconsumer - = data.onsumer *DataConsumer
// - @emits tuple - (tuple TransportTuple)
// - @emits rtcptuple - (rtcpTuple TransportTuple)
// - @emits sctpstatechange - (sctpState SctpState)
// - @emits trace - (trace *TransportTraceEventData)
func (transport *PlainTransport) Observer() IEventEmitter {
	return transport.ITransport.Observer()
}

// Close the PlainTransport.
func (transport *PlainTransport) Close() {
	if transport.Closed() {
		return
	}

	if len(transport.data.SctpState) > 0 {
		transport.data.SctpState = SctpState_Closed
	}

	transport.ITransport.Close()
}

// routerClosed is called when router was closed.
func (transport *PlainTransport) routerClosed() {
	if transport.Closed() {
		return
	}

	if len(transport.data.SctpState) > 0 {
		transport.data.SctpState = SctpState_Closed
	}

	transport.ITransport.routerClosed()
}

// Connect provide the PlainTransport remote parameters.
func (transport *PlainTransport) Connect(options TransportConnectOptions) (err error) {
	transport.logger.V(1).Info("connect()")

	reqData := TransportConnectOptions{
		Ip:             options.Ip,
		Port:           options.Port,
		RtcpPort:       options.RtcpPort,
		SrtpParameters: options.SrtpParameters,
	}
	resp := transport.channel.Request("transport.connect", transport.internal, reqData)

	var data struct {
		Tuple          *TransportTuple
		RtcpTuple      *TransportTuple
		SrtpParameters *SrtpParameters
	}
	if err = resp.Unmarshal(&data); err != nil {
		return
	}

	// Update data.
	if data.Tuple != nil {
		transport.data.Tuple = data.Tuple
	}
	if data.RtcpTuple != nil {
		transport.data.RtcpTuple = data.RtcpTuple
	}

	transport.data.SrtpParameters = data.SrtpParameters

	return nil
}

func (transport *PlainTransport) handleWorkerNotifications() {
	logger := transport.logger

	transport.channel.Subscribe(transport.Id(), func(event string, data []byte) {
		switch event {
		case "tuple":
			var result struct {
				Tuple *TransportTuple
			}
			if err := json.Unmarshal([]byte(data), &result); err != nil {
				logger.Error(err, "failed to unmarshal tuple", "data", json.RawMessage(data))
				return
			}

			transport.data.Tuple = result.Tuple

			transport.SafeEmit("tuple", result.Tuple)

			// Emit observer event.
			transport.Observer().SafeEmit("tuple", result.Tuple)

		case "rtcptuple":
			var result struct {
				RtcpTuple *TransportTuple
			}
			if err := json.Unmarshal([]byte(data), &result); err != nil {
				logger.Error(err, "failed to unmarshal rtcptuple", "data", json.RawMessage(data))
				return
			}

			transport.data.RtcpTuple = result.RtcpTuple

			transport.SafeEmit("rtcptuple", result.RtcpTuple)

			// Emit observer event.
			transport.Observer().SafeEmit("rtcptuple", result.RtcpTuple)

		case "sctpstatechange":
			var result struct {
				SctpState SctpState
			}
			if err := json.Unmarshal([]byte(data), &result); err != nil {
				logger.Error(err, "failed to unmarshal sctpstatechange", "data", json.RawMessage(data))
				return
			}

			transport.data.SctpState = result.SctpState

			transport.SafeEmit("sctpstatechange", result.SctpState)

			// Emit observer event.
			transport.Observer().SafeEmit("sctpstatechange", result.SctpState)

		case "trace":
			var result *TransportTraceEventData

			if err := json.Unmarshal([]byte(data), &result); err != nil {
				logger.Error(err, "failed to unmarshal trace", "data", json.RawMessage(data))
				return
			}

			transport.SafeEmit("trace", result)

			// Emit observer event.
			transport.Observer().SafeEmit("trace", result)

		default:
			logger.Error(nil, "ignoring unknown event", "event", event)
		}
	})
}
