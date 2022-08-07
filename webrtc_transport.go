package mediasoup

import (
	"encoding/json"
	"sync"
)

type WebRtcTransportOptions struct {
	/**
	 * Instance of WebRtcServer. Mandatory unless listenIps is given.
	 */
	WebRtcServer *WebRtcServer

	/**
	 * Listening IP address or addresses in order of preference (first one is the
	 * preferred one). Mandatory unless webRtcServer is given.
	 */
	ListenIps []TransportListenIp `json:"listenIps,omitempty"`

	/**
	 * Listen in UDP. Default true.
	 */
	EnableUdp *bool `json:"enableUdp,omitempty"`

	/**
	 * Listen in TCP. Default false.
	 */
	EnableTcp bool `json:"enableTcp,omitempty"`

	/**
	 * Prefer UDP. Default false.
	 */
	PreferUdp bool `json:"preferUdp,omitempty"`

	/**
	 * Prefer TCP. Default false.
	 */
	PreferTcp bool `json:"preferTcp,omitempty"`

	/**
	 * Initial available outgoing bitrate (in bps). Default 600000.
	 */
	InitialAvailableOutgoingBitrate uint32 `json:"initialAvailableOutgoingBitrate,omitempty"`

	/**
	 * Create a SCTP association. Default false.
	 */
	EnableSctp bool `json:"enableSctp,omitempty"`

	/**
	 * SCTP streams uint32.
	 */
	NumSctpStreams NumSctpStreams `json:"numSctpStreams,omitempty"`

	/**
	 * Maximum allowed size for SCTP messages sent by DataProducers.
	 * Default 262144.
	 */
	MaxSctpMessageSize int `json:"maxSctpMessageSize,omitempty"`

	/**
	 * Maximum SCTP send buffer used by DataConsumers.
	 * Default 262144.
	 */
	SctpSendBufferSize int `json:"sctpSendBufferSize,omitempty"`

	/**
	 * Custom application data.
	 */
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
	// "passive" | undefined
	TcpType string `json:"tcpType,omitempty"`
}

type DtlsParameters struct {
	Role         DtlsRole          `json:"role,omitempty"`
	Fingerprints []DtlsFingerprint `json:"fingerprints"`
}

/**
 * The hash function algorithm (as defined in the "Hash function Textual Names"
 * registry initially specified in RFC 4572 Section 8) and its corresponding
 * certificate fingerprint value (in lowercase hex string as expressed utilizing
 * the syntax of "fingerprint" in RFC 4572 Section 5).
 */
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
	locker sync.Mutex
	// alway be 'controlled'
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

func (data *webrtcTransportData) SetIceParameters(iceParameters IceParameters) {
	data.locker.Lock()
	defer data.locker.Unlock()
	data.IceParameters = iceParameters
}

func (data *webrtcTransportData) SetIceState(iceState IceState) {
	data.locker.Lock()
	defer data.locker.Unlock()
	data.IceState = iceState
}

func (data *webrtcTransportData) SetIceSelectedTuple(tuple *TransportTuple) {
	data.locker.Lock()
	defer data.locker.Unlock()
	data.IceSelectedTuple = tuple
}

func (data *webrtcTransportData) SetDtlsParametersRole(role DtlsRole) {
	data.locker.Lock()
	defer data.locker.Unlock()
	data.DtlsParameters.Role = role
}

func (data *webrtcTransportData) SetDtlsState(dtlsState DtlsState) {
	data.locker.Lock()
	defer data.locker.Unlock()
	data.DtlsState = dtlsState
}

func (data *webrtcTransportData) SetDtlsRemoteCert(dtlsRemoteCert string) {
	data.locker.Lock()
	defer data.locker.Unlock()
	data.DtlsRemoteCert = dtlsRemoteCert
}

func (data *webrtcTransportData) GetSctpState() (sctpState SctpState) {
	data.locker.Lock()
	defer data.locker.Unlock()
	return data.SctpState
}

func (data *webrtcTransportData) SetSctpState(sctpState SctpState) {
	data.locker.Lock()
	defer data.locker.Unlock()
	data.SctpState = sctpState
}

/**
 * WebRtcTransport
 * @emits icestatechange - (iceState: IceState)
 * @emits iceselectedtuplechange - (iceSelectedTuple: TransportTuple)
 * @emits dtlsstatechange - (dtlsState: DtlsState)
 * @emits sctpstatechange - (sctpState: SctpState)
 * @emits trace - (trace: TransportTraceEventData)
 */
type WebRtcTransport struct {
	ITransport
	logger         Logger
	internal       internalData
	data           *webrtcTransportData
	channel        *Channel
	payloadChannel *PayloadChannel
}

func newWebRtcTransport(params transportParams) ITransport {
	data := params.data.(*webrtcTransportData)
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

/**
 * ICE role.
 */
func (t WebRtcTransport) IceRole() string {
	return t.data.IceRole
}

/**
 * ICE parameters.
 */
func (t WebRtcTransport) IceParameters() IceParameters {
	return t.data.IceParameters
}

/**
 * ICE candidates.
 */
func (t WebRtcTransport) IceCandidates() []IceCandidate {
	return t.data.IceCandidates
}

/**
 * ICE state.
 */
func (t WebRtcTransport) IceState() IceState {
	return t.data.IceState
}

/**
 * ICE selected tuple.
 */
func (t WebRtcTransport) IceSelectedTuple() *TransportTuple {
	return t.data.IceSelectedTuple
}

/**
 * DTLS parameters.
 */
func (t WebRtcTransport) DtlsParameters() DtlsParameters {
	return t.data.DtlsParameters
}

/**
 * DTLS state.
 */
func (t WebRtcTransport) DtlsState() DtlsState {
	return t.data.DtlsState
}

/**
 * Remote certificate in PEM format.
 */
func (t WebRtcTransport) DtlsRemoteCert() string {
	return t.data.DtlsRemoteCert
}

/**
 * SCTP parameters.
 */
func (t WebRtcTransport) SctpParameters() SctpParameters {
	return t.data.SctpParameters
}

/**
 * SRTP parameters.
 */
func (t WebRtcTransport) SctpState() SctpState {
	return t.data.SctpState
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
 * @emits icestatechange - (iceState: IceState)
 * @emits iceselectedtuplechange - (iceSelectedTuple: TransportTuple)
 * @emits dtlsstatechange - (dtlsState: DtlsState)
 * @emits sctpstatechange - (sctpState: SctpState)
 * @emits trace - (trace: TransportTraceEventData)
 */
func (transport *WebRtcTransport) Observer() IEventEmitter {
	return transport.ITransport.Observer()
}

/**
 * Close the WebRtcTransport.
 *
 * @override
 */
func (transport *WebRtcTransport) Close() {
	if transport.Closed() {
		return
	}

	transport.data.SetIceSelectedTuple(nil)
	transport.data.SetIceState(IceState_Closed)
	transport.data.SetDtlsState(DtlsState_Closed)

	if len(transport.data.GetSctpState()) > 0 {
		transport.data.SetSctpState(SctpState_Closed)
	}

	transport.ITransport.Close()
}

/**
 * Router was closed.
 *
 * @override
 */
func (transport *WebRtcTransport) routerClosed() {
	if transport.Closed() {
		return
	}

	transport.data.SetIceSelectedTuple(nil)
	transport.data.SetIceState(IceState_Closed)
	transport.data.SetDtlsState(DtlsState_Closed)

	if len(transport.data.GetSctpState()) > 0 {
		transport.data.SetSctpState(SctpState_Closed)
	}

	transport.ITransport.routerClosed()
}

// webRtcServerClosed called when closing the associated WebRtcServer.
func (transport *WebRtcTransport) webRtcServerClosed() {
	if transport.Closed() {
		return
	}
	transport.data.IceState = IceState_Closed
	transport.data.IceSelectedTuple = nil
	transport.data.DtlsState = DtlsState_Closed

	if len(transport.data.SctpState) > 0 {
		transport.data.SctpState = SctpState_Closed
	}
}

/**
 * Provide the PlainTransport remote parameters.
 *
 * @override
 */
func (transport *WebRtcTransport) Connect(options TransportConnectOptions) (err error) {
	transport.logger.Debug("connect()")

	reqData := TransportConnectOptions{DtlsParameters: options.DtlsParameters}
	resp := transport.channel.Request("transport.connect", transport.internal, reqData)

	var data struct {
		DtlsLocalRole DtlsRole
	}
	if err = resp.Unmarshal(&data); err != nil {
		return
	}

	// Update data.
	transport.data.SetDtlsParametersRole(data.DtlsLocalRole)

	return
}

/**
 * Restart ICE.
 */
func (transport *WebRtcTransport) RestartIce() (iceParameters IceParameters, err error) {
	transport.logger.Debug("restartIce()")

	resp := transport.channel.Request("transport.restartIce", transport.internal)

	var data struct {
		IceParameters IceParameters
	}
	if err = resp.Unmarshal(&data); err == nil {
		transport.data.SetIceParameters(data.IceParameters)
	}

	return data.IceParameters, err
}

func (transport *WebRtcTransport) handleWorkerNotifications() {
	transport.channel.On(transport.Id(), func(event string, data []byte) {
		switch event {
		case "icestatechange":
			var result struct {
				IceState IceState
			}
			json.Unmarshal(data, &result)

			transport.SafeEmit("icestatechange", result.IceState)

			// Emit observer event.
			transport.Observer().SafeEmit("icestatechange", result.IceState)

		case "iceselectedtuplechange":
			var result struct {
				IceSelectedTuple TransportTuple
			}
			json.Unmarshal(data, &result)

			transport.data.SetIceSelectedTuple(&result.IceSelectedTuple)

			transport.SafeEmit("iceselectedtuplechange", result.IceSelectedTuple)

			// Emit observer event.
			transport.Observer().SafeEmit("iceselectedtuplechange", result.IceSelectedTuple)

		case "dtlsstatechange":
			var result struct {
				DtlsState      DtlsState
				DtlsRemoteCert string
			}
			json.Unmarshal(data, &result)

			transport.data.SetDtlsState(result.DtlsState)

			if result.DtlsState == "connected" {
				transport.data.SetDtlsRemoteCert(result.DtlsRemoteCert)
			}

			transport.SafeEmit("dtlsstatechange", result.DtlsState)

			// Emit observer event.
			transport.Observer().SafeEmit("dtlsstatechange", result.DtlsState)

		case "sctpstatechange":
			var result struct {
				SctpState SctpState
			}
			json.Unmarshal(data, &result)

			transport.data.SetSctpState(result.SctpState)

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
