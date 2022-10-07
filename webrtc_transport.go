package mediasoup

import (
	"encoding/json"

	"github.com/go-logr/logr"
)

// WebRtcTransportOptions defines the options to create webrtc t.
type WebRtcTransportOptions struct {
	// WebRtcServer is an instance of WebRtcServer. Mandatory unless listenIps is given.
	WebRtcServer *WebRtcServer

	// ListenIps are listening IP address or addresses in order of preference (first one
	// is the preferred one). Mandatory unless webRtcServer is given.
	ListenIps []TransportListenIp `json:"listenIps,omitempty"`

	// EnableUdp enables listening in UDP. Default true.
	EnableUdp *bool `json:"enableUdp,omitempty"`

	// EnableTcp enables listening in TCP. Default false.
	EnableTcp bool `json:"enableTcp,omitempty"`

	// PreferUdp prefers UDP. Default false.
	PreferUdp bool `json:"preferUdp,omitempty"`

	// PreferUdp prefers TCP. Default false.
	PreferTcp bool `json:"preferTcp,omitempty"`

	// InitialAvailableOutgoingBitrate sets the initial available outgoing bitrate (in bps). Default 600000.
	InitialAvailableOutgoingBitrate int `json:"initialAvailableOutgoingBitrate,omitempty"`

	// EnableSctp creates a SCTP association. Default false.
	EnableSctp bool `json:"enableSctp,omitempty"`

	// NumSctpStreams set up SCTP streams.
	NumSctpStreams NumSctpStreams `json:"numSctpStreams,omitempty"`

	// MaxSctpMessageSize defines the maximum allowed size for SCTP messages sent by DataProducers. Default 262144.
	MaxSctpMessageSize int `json:"maxSctpMessageSize,omitempty"`

	// SctpSendBufferSize defines the maximum SCTP send buffer used by DataConsumers. Default 262144.
	SctpSendBufferSize int `json:"sctpSendBufferSize,omitempty"`

	// AppData is the custom application data.
	AppData interface{} `json:"appData,omitempty"`
}

type IceParameters struct {
	UsernameFragment string `json:"usernameFragment"`
	Password         string `json:"password"`
	IceLite          bool   `json:"iceLite,omitempty"`
}

type IceCandidate struct {
	Foundation string            `json:"foundation"`
	Priority   uint32            `json:"priority"`
	Ip         string            `json:"ip"`
	Protocol   TransportProtocol `json:"protocol"`
	Port       uint16            `json:"port"`
	// alway "host"
	Type string `json:"type,omitempty"`
	// "passive" | ""
	TcpType string `json:"tcpType,omitempty"`
}

type DtlsParameters struct {
	Role         DtlsRole          `json:"role,omitempty"`
	Fingerprints []DtlsFingerprint `json:"fingerprints"`
}

// DtlsFingerprint defines the hash function algorithm (as defined in the
// "Hash function Textual Names" registry initially specified in RFC 4572 Section 8)
// and its corresponding certificate fingerprint value (in lowercase hex string as
// expressed utilizing the syntax of "fingerprint" in RFC 4572 Section 5).
type DtlsFingerprint struct {
	Algorithm string `json:"algorithm"`
	Value     string `json:"value"`
}

type IceState string

const (
	IceState_New          IceState = "new"
	IceState_Connected    IceState = "connected"
	IceState_Completed    IceState = "completed"
	IceState_Disconnected IceState = "disconnected"
	IceState_Closed       IceState = "closed"
)

type DtlsRole string

const (
	DtlsRole_Auto   DtlsRole = "auto"
	DtlsRole_Client DtlsRole = "client"
	DtlsRole_Server DtlsRole = "server"
)

type DtlsState string

const (
	DtlsState_New        = "new"
	DtlsState_Connecting = "connecting"
	DtlsState_Connected  = "connected"
	DtlsState_Failed     = "failed"
	DtlsState_Closed     = "closed"
)

type WebRtcTransportSpecificStat struct {
	IceRole          string          `json:"iceRole"`
	IceState         IceState        `json:"iceState"`
	DtlsState        DtlsRole        `json:"dtlsState"`
	IceSelectedTuple *TransportTuple `json:"iceSelectedTuple,omitempty"`
}

type webrtcTransportData struct {
	// alway be "controlled"
	IceRole          string          `json:"iceRole,omitempty"`
	IceParameters    IceParameters   `json:"iceParameters,omitempty"`
	IceCandidates    []IceCandidate  `json:"iceCandidates,omitempty"`
	IceState         IceState        `json:"iceState,omitempty"`
	IceSelectedTuple *TransportTuple `json:"iceSelectedTuple,omitempty"`
	DtlsParameters   DtlsParameters  `json:"dtlsParameters,omitempty"`
	DtlsState        DtlsState       `json:"dtlsState,omitempty"`
	DtlsRemoteCert   string          `json:"dtlsRemoteCert,omitempty"`
	SctpParameters   SctpParameters  `json:"sctpParameters,omitempty"`
	SctpState        SctpState       `json:"sctpState,omitempty"`
}

// WebRtcTransport represents a network path negotiated by both, a WebRTC endpoint and mediasoup,
// via ICE and DTLS procedures. A WebRTC transport may be used to receive media, to send media or
// to both receive and send. There is no limitation in mediasoup. However, due to their design,
// mediasoup-client and libmediasoupclient require separate WebRTC transports for sending and
// receiving.
//
// The WebRTC transport implementation of mediasoup is ICE Lite, meaning that it does not initiate
// ICE connections but expects ICE Binding Requests from endpoints.
//
// - @emits icestatechange - (iceState IceState)
// - @emits iceselectedtuplechange - (tuple *TransportTuple)
// - @emits dtlsstatechange - (dtlsState DtlsState)
// - @emits sctpstatechange - (sctpState SctpState)
// - @emits trace - (trace TransportTraceEventData)
type WebRtcTransport struct {
	ITransport
	logger         logr.Logger
	internal       internalData
	data           *webrtcTransportData
	channel        *Channel
	payloadChannel *PayloadChannel
}

func newWebRtcTransport(params transportParams) ITransport {
	data, _ := params.data.(*webrtcTransportData)
	params.data = transportData{
		sctpParameters: data.SctpParameters,
		sctpState:      data.SctpState,
		transportType:  TransportType_Webrtc,
	}
	params.logger = NewLogger("WebRtcTransport")

	transport := &WebRtcTransport{
		ITransport:     newTransport(params),
		logger:         params.logger,
		internal:       params.internal,
		data:           data,
		channel:        params.channel,
		payloadChannel: params.payloadChannel,
	}

	transport.handleWorkerNotifications()

	return transport
}

// IceRole returns ICE role.
func (t WebRtcTransport) IceRole() string {
	return t.data.IceRole
}

// IceParameters returns ICE parameters.
func (t WebRtcTransport) IceParameters() IceParameters {
	return t.data.IceParameters
}

// returns IceCandidates ICE candidates.
func (t WebRtcTransport) IceCandidates() []IceCandidate {
	return t.data.IceCandidates
}

// IceState returns ICE state.
func (t WebRtcTransport) IceState() IceState {
	return t.data.IceState
}

// IceSelectedTuple returns ICE selected tuple.
func (t WebRtcTransport) IceSelectedTuple() *TransportTuple {
	return t.data.IceSelectedTuple
}

// DtlsParameters returns DTLS parameters.
func (t WebRtcTransport) DtlsParameters() DtlsParameters {
	return t.data.DtlsParameters
}

// DtlsState returns DTLS state.
func (t WebRtcTransport) DtlsState() DtlsState {
	return t.data.DtlsState
}

// DtlsRemoteCert returns remote certificate in PEM format
func (t WebRtcTransport) DtlsRemoteCert() string {
	return t.data.DtlsRemoteCert
}

// SctpParameters returns SCTP parameters.
func (t WebRtcTransport) SctpParameters() SctpParameters {
	return t.data.SctpParameters
}

// SctpState returns SRTP parameters.
func (t WebRtcTransport) SctpState() SctpState {
	return t.data.SctpState
}

// Observer returns an EventEmitter object.
//
// - @emits close
// - @emits newproducer - (producer *Producer)
// - @emits newconsumer - (consumer *Consumer)
// - @emits newdataproducer - (dataProducer *DataProducer)
// - @emits newdataconsumer - (dataConsumer *DataConsumer)
// - @emits icestatechange - (iceState IceState)
// - @emits iceselectedtuplechange - (tuple *TransportTuple)
// - @emits dtlsstatechange - (dtlsState DtlsState)
// - @emits sctpstatechange - (sctpState SctpState)
// - @emits trace - (trace TransportTraceEventData)
func (t *WebRtcTransport) Observer() IEventEmitter {
	return t.ITransport.Observer()
}

// Close the WebRtcTransport.
func (t *WebRtcTransport) Close() {
	if t.Closed() {
		return
	}

	t.data.IceState = IceState_Closed
	t.data.IceSelectedTuple = nil
	t.data.DtlsState = DtlsState_Closed

	if len(t.data.SctpState) > 0 {
		t.data.SctpState = SctpState_Closed
	}

	t.ITransport.Close()
}

// routerClosed called when router was closed.
func (t *WebRtcTransport) routerClosed() {
	if t.Closed() {
		return
	}

	t.data.IceState = IceState_Closed
	t.data.IceSelectedTuple = nil
	t.data.DtlsState = DtlsState_Closed

	if len(t.data.SctpState) > 0 {
		t.data.SctpState = SctpState_Closed
	}

	t.ITransport.routerClosed()
}

// webRtcServerClosed called when closing the associated WebRtcServer.
func (t *WebRtcTransport) webRtcServerClosed() {
	if t.Closed() {
		return
	}
	t.data.IceState = IceState_Closed
	t.data.IceSelectedTuple = nil
	t.data.DtlsState = DtlsState_Closed

	if len(t.data.SctpState) > 0 {
		t.data.SctpState = SctpState_Closed
	}
}

// Connect provides the WebRtcTransport remote parameters.
func (t *WebRtcTransport) Connect(options TransportConnectOptions) (err error) {
	t.logger.V(1).Info("connect()")

	reqData := TransportConnectOptions{DtlsParameters: options.DtlsParameters}
	resp := t.channel.Request("transport.connect", t.internal, reqData)

	var result struct {
		DtlsLocalRole DtlsRole
	}
	if err = resp.Unmarshal(&result); err != nil {
		return
	}

	// Update data.
	t.data.DtlsParameters.Role = result.DtlsLocalRole

	return
}

// RestartIce restarts ICE.
func (t *WebRtcTransport) RestartIce() (iceParameters IceParameters, err error) {
	t.logger.V(1).Info("restartIce()")

	resp := t.channel.Request("transport.restartIce", t.internal)

	var result struct {
		IceParameters IceParameters
	}
	if err = resp.Unmarshal(&result); err != nil {
		return
	}

	t.data.IceParameters = result.IceParameters

	return result.IceParameters, nil
}

// handleWorkerNotifications handle WebRtcTransport's notifications from worker.
func (t *WebRtcTransport) handleWorkerNotifications() {
	logger := t.logger

	t.channel.Subscribe(t.Id(), func(event string, data []byte) {
		switch event {
		case "icestatechange":
			var result struct {
				IceState IceState
			}
			if err := json.Unmarshal([]byte(data), &result); err != nil {
				logger.Error(err, "failed to unmarshal icestatechange", "data", json.RawMessage(data))
				return
			}

			t.SafeEmit("icestatechange", result.IceState)

			// Emit observer event.
			t.Observer().SafeEmit("icestatechange", result.IceState)

		case "iceselectedtuplechange":
			var result struct {
				IceSelectedTuple *TransportTuple
			}
			if err := json.Unmarshal([]byte(data), &result); err != nil {
				logger.Error(err, "failed to unmarshal iceselectedtuplechange", "data", json.RawMessage(data))
				return
			}

			t.data.IceSelectedTuple = result.IceSelectedTuple

			t.SafeEmit("iceselectedtuplechange", result.IceSelectedTuple)

			// Emit observer event.
			t.Observer().SafeEmit("iceselectedtuplechange", result.IceSelectedTuple)

		case "dtlsstatechange":
			var result struct {
				DtlsState      DtlsState
				DtlsRemoteCert string
			}
			if err := json.Unmarshal([]byte(data), &result); err != nil {
				logger.Error(err, "failed to unmarshal dtlsstatechange", "data", json.RawMessage(data))
				return
			}

			t.data.DtlsState = result.DtlsState

			if result.DtlsState == "connected" {
				t.data.DtlsRemoteCert = result.DtlsRemoteCert
			}

			t.SafeEmit("dtlsstatechange", result.DtlsState)

			// Emit observer event.
			t.Observer().SafeEmit("dtlsstatechange", result.DtlsState)

		case "sctpstatechange":
			var result struct {
				SctpState SctpState
			}
			if err := json.Unmarshal([]byte(data), &result); err != nil {
				logger.Error(err, "failed to unmarshal sctpstatechange", "data", json.RawMessage(data))
				return
			}

			t.data.SctpState = result.SctpState

			t.SafeEmit("sctpstatechange", result.SctpState)

			// Emit observer event.
			t.Observer().SafeEmit("sctpstatechange", result.SctpState)

		case "trace":
			var result *TransportTraceEventData
			if err := json.Unmarshal([]byte(data), &result); err != nil {
				logger.Error(err, "failed to unmarshal trace", "data", json.RawMessage(data))
				return
			}

			t.SafeEmit("trace", result)

			// Emit observer event.
			t.Observer().SafeEmit("trace", result)

		default:
			t.logger.Error(nil, "ignoring unknown event", "event", event)
		}
	})
}
