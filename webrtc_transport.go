package mediasoup

type WebRtcTransportOptions struct {
	/**
	 * Listening IP address or addresses in order of preference (first one is the
	 * preferred one).
	 */
	ListenIps []TransportListenIp

	/**
	 * Listen in UDP. Default true.
	 */
	EnableUdp bool

	/**
	 * Listen in TCP. Default false.
	 */
	EnableTcp bool

	/**
	 * Prefer UDP. Default false.
	 */
	PreferUdp bool

	/**
	 * Prefer TCP. Default false.
	 */
	PreferTcp bool

	/**
	 * Initial available outgoing bitrate (in bps). Default 600000.
	 */
	InitialAvailableOutgoingBitrate uint32

	/**
	 * Create a SCTP association. Default false.
	 */
	EnableSctp bool

	/**
	 * SCTP streams uint32.
	 */
	NumSctpStreams NumSctpStreams

	/**
	 * Maximum allowed size for SCTP messages sent by DataProducers.
	 * Default 262144.
	 */
	MaxSctpMessageSize uint32

	/**
	 * Maximum SCTP send buffer used by DataConsumers.
	 * Default 262144.
	 */
	SctpSendBufferSize uint32

	/**
	 * Custom application data.
	 */
	AppData interface{}
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
	Port       uint32            `json:"port"`
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
	IceState_Connected             = "connected"
	IceState_Completed             = "completed"
	IceState_Disconnected          = "disconnected"
	IceState_Closed                = "closed"
)

type DtlsRole string

const (
	DtlsRole_Auto   DtlsRole = "auto"
	DtlsRole_Client          = "client"
	DtlsRole_Server          = "server"
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

type WebRtcTransport struct {
	ITransport
}

func newWebRtcTransport(params transportParams) *WebRtcTransport {
	return &WebRtcTransport{
		ITransport: newTransport(params),
	}
}
