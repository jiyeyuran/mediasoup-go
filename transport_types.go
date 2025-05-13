package mediasoup

// WebRtcTransportOptions defines the options to create webrtc transport.
type WebRtcTransportOptions struct {
	// WebRtcServer is an instance of WebRtcServer. Mandatory unless listenIps is given.
	WebRtcServer *WebRtcServer `json:"-"`

	// ListenInfos specifies listening IP address or addresses in order of preference (first one
	// is the preferred one). Mandatory unless webRtcServer is given.
	ListenInfos []TransportListenInfo `json:"listenIps,omitempty"`

	// EnableUdp enables listening in UDP. Default true.
	EnableUdp *bool `json:"enableUdp,omitempty"`

	// EnableTcp enables listening in TCP. Default false.
	EnableTcp bool `json:"enableTcp,omitempty"`

	// PreferUdp indicates if UDP should be preferred. Default false.
	PreferUdp bool `json:"preferUdp,omitempty"`

	// PreferTcp indicates if TCP should be preferred. Default false.
	PreferTcp bool `json:"preferTcp,omitempty"`

	// IceConsentTimeout is ICE consent timeout in seconds. If 0 it is disabled. Default 30.
	IceConsentTimeout *uint8 `json:"iceConsentTimeout,omitempty"`

	// InitialAvailableOutgoingBitrate sets the initial available outgoing bitrate (in bps). Default 600000.
	InitialAvailableOutgoingBitrate uint32 `json:"initialAvailableOutgoingBitrate,omitempty"`

	// EnableSctp enables SCTP association creation. Default false.
	EnableSctp bool `json:"enableSctp,omitempty"`

	// NumSctpStreams configures SCTP streams.
	NumSctpStreams *NumSctpStreams `json:"numSctpStreams,omitempty"`

	// MaxSctpMessageSize is the maximum allowed size for SCTP messages sent by DataProducers. Default 262144.
	MaxSctpMessageSize uint32 `json:"maxSctpMessageSize,omitempty"`

	// SctpSendBufferSize is the maximum SCTP send buffer used by DataConsumers. Default 262144.
	SctpSendBufferSize uint32 `json:"sctpSendBufferSize,omitempty"`

	// AppData is custom application data.
	AppData H `json:"appData,omitempty"`
}

// PlainTransportOptions define options to create a PlainTransport
type PlainTransportOptions struct {
	// ListenInfo define Listening IP address.
	ListenInfo TransportListenInfo `json:"listenInfo,omitempty"`

	// RtcpListenInfo is optional listening info for RTCP.
	RtcpListenInfo *TransportListenInfo `json:"rtcpListenInfo,omitempty"`

	// RtcpMux define wether use RTCP-mux (RTP and RTCP in the same port). Default true.
	RtcpMux *bool `json:"rtcpMux,omitempty"`

	// Comedia define whether remote ip:port should be auto-detected based on first RTP/RTCP
	// packet received. If enabled, connect() method must not be called unless
	// SRTP is enabled. If so, it must be called with just remote SRTP parameters.
	// Default false.
	Comedia bool `json:"comedia,omitempty"`

	// EnableSctp define whether create a SCTP association. Default false.
	EnableSctp bool `json:"enableSctp,omitempty"`

	// NumSctpStreams define SCTP streams number.
	NumSctpStreams *NumSctpStreams `json:"numSctpStreams,omitempty"`

	// MaxSctpMessageSize define maximum allowed size for SCTP messages sent by DataProducers.
	// Default 262144.
	MaxSctpMessageSize uint32 `json:"maxSctpMessageSize,omitempty"`

	// SctpSendBufferSize define maximum SCTP send buffer used by DataConsumers.
	// Default 262144.
	SctpSendBufferSize uint32 `json:"sctpSendBufferSize,omitempty"`

	// EnableSrtp enable SRTP. For this to work, connect() must be called
	// with remote SRTP parameters. Default false.
	EnableSrtp bool `json:"enableSrtp,omitempty"`

	// SrtpCryptoSuite define the SRTP crypto suite to be used if enableSrtp is set. Default
	// 'AES_CM_128_HMAC_SHA1_80'.
	SrtpCryptoSuite SrtpCryptoSuite `json:"srtpCryptoSuite,omitempty"`

	// AppData is custom application data.
	AppData H `json:"appData,omitempty"`
}

// SrtpParameters defines SRTP parameters.
type SrtpParameters struct {
	//Encryption and authentication transforms to be used.
	CryptoSuite SrtpCryptoSuite `json:"cryptoSuite"`

	// SRTP keying material (master key and salt) in Base64.
	KeyBase64 string `json:"keyBase64"`
}

// PipeTransportOptions define options to create a PipeTransport
type PipeTransportOptions struct {
	// ListenInfo define Listening IP address.
	ListenInfo TransportListenInfo `json:"listenInfo,omitempty"`

	// EnableSctp define whether create a SCTP association. Default false.
	EnableSctp bool `json:"enableSctp,omitempty"`

	// NumSctpStreams define SCTP streams number.
	NumSctpStreams *NumSctpStreams `json:"numSctpStreams,omitempty"`

	// MaxSctpMessageSize define maximum allowed size for SCTP messages sent by DataProducers.
	// Default 268435456.
	MaxSctpMessageSize uint32 `json:"maxSctpMessageSize,omitempty"`

	// SctpSendBufferSize define maximum SCTP send buffer used by DataConsumers.
	// Default 268435456.
	SctpSendBufferSize uint32 `json:"sctpSendBufferSize,omitempty"`

	// EnableSrtp enable SRTP. For this to work, connect() must be called
	// with remote SRTP parameters. Default false.
	EnableSrtp bool `json:"enableSrtp,omitempty"`

	// EnableRtx enable RTX and NACK for RTP retransmission. Useful if both Routers are
	// located in different hosts and there is packet lost in the link. For this
	// to work, both PipeTransports must enable this setting. Default false.
	EnableRtx bool `json:"enableRtx,omitempty"`

	// AppData is custom application data.
	AppData H `json:"appData,omitempty"`
}

// DirectTransportOptions define options to create a DirectTransport.
type DirectTransportOptions struct {
	// MaxMessageSize define maximum allowed size for direct messages sent from DataProducers.
	// Default 262144.
	MaxMessageSize uint32 `json:"maxMessageSize,omitempty"`

	// AppData is custom application data.
	AppData H `json:"appData,omitempty"`
}

// TransportType represents the transport type.
type TransportType string

const (
	TransportWebRTC TransportType = "webrtc"
	TransportPlain  TransportType = "plain"
	TransportPipe   TransportType = "pipe"
	TransportDirect TransportType = "direct"
)

// TransportListenInfo represents the transport listening information.
type TransportListenInfo struct {
	// Protocol network protocol
	Protocol TransportProtocol `json:"protocol"`

	// Ip listening IPv4 or IPv6
	Ip string `json:"ip"`

	// AnnouncedAddress announced IPv4, IPv6 or hostname (useful when running mediasoup behind NAT with private IP)
	AnnouncedAddress string `json:"announcedAddress,omitempty"`

	// Port listening port
	Port uint16 `json:"port,omitempty"`

	// PortRange listening port range. If given then Port will be ignored
	PortRange TransportPortRange `json:"portRange,omitempty"`

	// Flags socket flags
	Flags TransportSocketFlags `json:"flags,omitempty"`

	// SendBufferSize send buffer size (bytes)
	SendBufferSize uint32 `json:"sendBufferSize,omitempty"`

	// RecvBufferSize recv buffer size (bytes)
	RecvBufferSize uint32 `json:"recvBufferSize,omitempty"`
}

// TransportProtocol represents the transport protocol.
type TransportProtocol string

const (
	TransportProtocolUDP TransportProtocol = "udp"
	TransportProtocolTCP TransportProtocol = "tcp"
)

// TransportPortRange represents a port range.
type TransportPortRange struct {
	Min uint16 `json:"min"`
	Max uint16 `json:"max"`
}

// TransportSocketFlags represents UDP/TCP socket flags.
type TransportSocketFlags struct {
	IPv6Only     bool `json:"ipv6Only,omitempty"`
	UDPReusePort bool `json:"udpReusePort,omitempty"`
}

// TransportTuple represents a transport tuple.
type TransportTuple struct {
	Protocol     TransportProtocol `json:"protocol"`
	LocalAddress string            `json:"localAddress"`
	LocalPort    uint16            `json:"localPort"`
	RemoteIp     string            `json:"remoteIp,omitempty"`
	RemotePort   uint16            `json:"remotePort,omitempty"`
}

// SctpState represents the SCTP state.
type SctpState string

const (
	SctpStateNew        SctpState = "new"
	SctpStateConnecting SctpState = "connecting"
	SctpStateConnected  SctpState = "connected"
	SctpStateFailed     SctpState = "failed"
	SctpStateClosed     SctpState = "closed"
)

// RtpListenerDump represents RTP listener dump information.
type RtpListenerDump struct {
	SSRCTable []KeyValue[int, string] `json:"ssrcTable"`
	MIDTable  []KeyValue[int, string] `json:"midTable"`
	RIDTable  []KeyValue[int, string] `json:"ridTable"`
}

// SctpListenerDump represents SCTP listener dump information.
type SctpListenerDump struct {
	StreamIDTable []KeyValue[int, string] `json:"streamIdTable"`
}

// RecvRtpHeaderExtensions represents received RTP header extensions.
type RecvRtpHeaderExtensions struct {
	MID               *byte `json:"mid,omitempty"`
	RID               *byte `json:"rid,omitempty"`
	RRID              *byte `json:"rrid,omitempty"`
	AbsSendTime       *byte `json:"absSendTime,omitempty"`
	TransportWideCC01 *byte `json:"transportWideCc01,omitempty"`
}

// TransportTraceEventType represents valid types for 'trace' events.
type TransportTraceEventType string

const (
	TransportTraceEventProbation TransportTraceEventType = "probation"
	TransportTraceEventBWE       TransportTraceEventType = "bwe"
)

// TransportTraceEventData represents 'trace' event data.
type TransportTraceEventData struct {
	Type      TransportTraceEventType `json:"type"`
	Timestamp uint64                  `json:"timestamp"`
	Direction string                  `json:"direction"` // "in" or "out"
	Info      H                       `json:"info"`
}

type TransportDump struct {
	Id                      string                     `json:"id"`
	Type                    TransportType              `json:"type"`
	Direct                  bool                       `json:"direct,omitempty"`
	ProducerIds             []string                   `json:"producerIds"`
	ConsumerIds             []string                   `json:"consumerIds"`
	MapSsrcConsumerId       []KeyValue[uint32, string] `json:"mapSsrcConsumerId"`
	MapRtxSsrcConsumerId    []KeyValue[uint32, string] `json:"mapRtxSsrcConsumerId"`
	DataProducerIds         []string                   `json:"dataProducerIds"`
	DataConsumerIds         []string                   `json:"dataConsumerIds"`
	RecvRtpHeaderExtensions *RecvRtpHeaderExtensions   `json:"recvRtpHeaderExtensions"`
	RtpListener             *RtpListener               `json:"rtpListener"`
	MaxMessageSize          uint32                     `json:"maxMessageSize,omitempty"`
	SctpParameters          *SctpParameters            `json:"SctpParameters,omitempty"`
	SctpState               SctpState                  `json:"sctpState,omitempty"`
	SctpListener            *SctpListener              `json:"sctpListener,omitempty"`
	TraceEventTypes         []TransportTraceEventType  `json:"traceEventTypes"`

	*PlainTransportDump
	*WebRtcTransportDump
	*PipeTransportDump
}

type PlainTransportDump struct {
	RtcpMux        bool            `json:"rtcpMux,omitempty"`
	Comedia        bool            `json:"comedia,omitempty"`
	Tuple          TransportTuple  `json:"tuple"`
	RtcpTuple      *TransportTuple `json:"rtcpTuple,omitempty"`
	SrtpParameters *SrtpParameters `json:"srtpParameters,omitempty"`
}

type PipeTransportDump struct {
	Tuple          TransportTuple  `json:"tuple"`
	Rtx            bool            `json:"rtx,omitempty"`
	SrtpParameters *SrtpParameters `json:"srtpParameters,omitempty"`
}

type WebRtcTransportDump struct {
	IceRole          string          `json:"iceRole,omitempty"`
	IceParameters    IceParameters   `json:"iceParameters"`
	IceCandidates    []IceCandidate  `json:"iceCandidates"`
	IceState         IceState        `json:"iceState,omitempty"`
	IceSelectedTuple *TransportTuple `json:"iceSelectedTuple,omitempty"`
	DtlsParameters   DtlsParameters  `json:"dtlsParameters"`
	DtlsState        DtlsState       `json:"dtlsState,omitempty"`
}

type RtpListener struct {
	SsrcTable []KeyValue[uint32, string] `json:"ssrcTable,omitempty"`
	MidTable  []KeyValue[string, string] `json:"midTable,omitempty"`
	RidTable  []KeyValue[string, string] `json:"ridTable,omitempty"`
}

type SctpListener struct {
	StreamIdTable []KeyValue[uint16, string] `json:"streamIdTable,omitempty"`
}

type TransportStat struct {
	BaseTransportStat
	*WebRtcTransportStat
	*PlainTransportStat
	*PipeTransportStat
}

// BaseTransportStat represents base transport statistics.
type BaseTransportStat struct {
	Type                     string    `json:"type"`
	TransportId              string    `json:"transportId"`
	Timestamp                uint64    `json:"timestamp"`
	SctpState                SctpState `json:"sctpState"`
	BytesReceived            uint64    `json:"bytesReceived"`
	RecvBitrate              uint32    `json:"recvBitrate"`
	BytesSent                uint64    `json:"bytesSent"`
	SendBitrate              uint32    `json:"sendBitrate"`
	RtpBytesReceived         uint64    `json:"rtpBytesReceived"`
	RtpRecvBitrate           uint32    `json:"rtpRecvBitrate"`
	RtpBytesSent             uint64    `json:"rtpBytesSent"`
	RtpSendBitrate           uint32    `json:"rtpSendBitrate"`
	RtxBytesReceived         uint64    `json:"rtxBytesReceived"`
	RtxRecvBitrate           uint32    `json:"rtxRecvBitrate"`
	RtxBytesSent             uint64    `json:"rtxBytesSent"`
	RtxSendBitrate           uint32    `json:"rtxSendBitrate"`
	ProbationBytesSent       uint64    `json:"probationBytesSent"`
	ProbationSendBitrate     uint32    `json:"probationSendBitrate"`
	AvailableOutgoingBitrate *uint32   `json:"availableOutgoingBitrate,omitempty"`
	AvailableIncomingBitrate *uint32   `json:"availableIncomingBitrate,omitempty"`
	MaxIncomingBitrate       *uint32   `json:"maxIncomingBitrate,omitempty"`
	MaxOutgoingBitrate       *uint32   `json:"maxOutgoingBitrate,omitempty"`
	MinOutgoingBitrate       *uint32   `json:"minOutgoingBitrate,omitempty"`
	RtpPacketLossReceived    *float64  `json:"rtpPacketLossReceived,omitempty"`
	RtpPacketLossSent        *float64  `json:"rtpPacketLossSent,omitempty"`
}

type WebRtcTransportStat struct {
	IceRole          string          `json:"iceRole"`
	IceState         IceState        `json:"iceState"`
	DtlsState        DtlsState       `json:"dtlsState"`
	IceSelectedTuple *TransportTuple `json:"iceSelectedTuple,omitempty"`
}

type PlainTransportStat struct {
	RtcpMux   bool            `json:"rtcp_mux"`
	Comedia   bool            `json:"comedia"`
	Tuple     TransportTuple  `json:"tuple"`
	RtcpTuple *TransportTuple `json:"rtcpTuple,omitempty"`
}

type PipeTransportStat struct {
	Tuple TransportTuple `json:"tuple"`
}

type TransportListenIp struct {
	// Listening IPv4 or IPv6.
	Ip string `json:"ip,omitempty"`
	// Announced IPv4 or IPv6 (useful when running mediasoup behind NAT with private IP).
	AnnouncedIp string `json:"announcedIp,omitempty"`
}

type IceParameters struct {
	UsernameFragment string `json:"usernameFragment"`
	Password         string `json:"password"`
	IceLite          bool   `json:"iceLite,omitempty"`
}

type IceCandidate struct {
	Foundation string            `json:"foundation"`
	Priority   uint32            `json:"priority"`
	Address    string            `json:"address"`
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
	IceStateNew          IceState = "new"
	IceStateConnected    IceState = "connected"
	IceStateCompleted    IceState = "completed"
	IceStateDisconnected IceState = "disconnected"
	IceStateClosed       IceState = "closed"
)

type DtlsRole string

const (
	DtlsRoleAuto   DtlsRole = "auto"
	DtlsRoleClient DtlsRole = "client"
	DtlsRoleServer DtlsRole = "server"
)

type DtlsState string

const (
	DtlsStateNew        DtlsState = "new"
	DtlsStateConnecting DtlsState = "connecting"
	DtlsStateConnected  DtlsState = "connected"
	DtlsStateFailed     DtlsState = "failed"
	DtlsStateClosed     DtlsState = "closed"
)

// SrtpCryptoSuite defines SRTP crypto suite.
type SrtpCryptoSuite string

const (
	AEAD_AES_256_GCM        SrtpCryptoSuite = "AEAD_AES_256_GCM"
	AEAD_AES_128_GCM        SrtpCryptoSuite = "AEAD_AES_128_GCM"
	AES_CM_128_HMAC_SHA1_80 SrtpCryptoSuite = "AES_CM_128_HMAC_SHA1_80"
	AES_CM_128_HMAC_SHA1_32 SrtpCryptoSuite = "AES_CM_128_HMAC_SHA1_32"
)

type TransportConnectOptions struct {
	// pipe and plain transport
	Ip             string          `json:"ip,omitempty"`
	Port           *uint16         `json:"port,omitempty"`
	SrtpParameters *SrtpParameters `json:"srtpParameters,omitempty"`
	// plain transport
	RtcpPort *uint16 `json:"rtcpPort,omitempty"`
	// webrtc transport
	DtlsParameters *DtlsParameters `json:"dtlsParameters,omitempty"`
}

type TransportData struct {
	*WebRtcTransportData
	*PlainTransportData
	*PipeTransportData
}

func (t *TransportData) clone() *TransportData {
	if t == nil {
		return nil
	}
	return &TransportData{
		WebRtcTransportData: t.WebRtcTransportData.clone(),
		PlainTransportData:  t.PlainTransportData.clone(),
		PipeTransportData:   t.PipeTransportData.clone(),
	}
}

type WebRtcTransportData struct {
	// IceRole alway be "controlled"
	IceRole          string
	IceParameters    IceParameters
	IceCandidates    []IceCandidate
	IceState         IceState
	IceSelectedTuple *TransportTuple
	DtlsParameters   DtlsParameters
	DtlsState        DtlsState
	DtlsRemoteCert   string
	SctpParameters   *SctpParameters
	SctpState        SctpState
}

func (d *WebRtcTransportData) clone() *WebRtcTransportData {
	if d == nil {
		return nil
	}
	iceCandidates := make([]IceCandidate, len(d.IceCandidates))
	copy(iceCandidates, d.IceCandidates)

	fingerprints := make([]DtlsFingerprint, len(d.DtlsParameters.Fingerprints))
	copy(fingerprints, d.DtlsParameters.Fingerprints)

	return &WebRtcTransportData{
		IceRole:          d.IceRole,
		IceParameters:    d.IceParameters,
		IceCandidates:    iceCandidates,
		IceState:         d.IceState,
		IceSelectedTuple: ifElse(d.IceSelectedTuple != nil, func() *TransportTuple { return ref(*d.IceSelectedTuple) }),
		DtlsParameters: DtlsParameters{
			Role:         d.DtlsParameters.Role,
			Fingerprints: fingerprints,
		},
		DtlsState:      d.DtlsState,
		DtlsRemoteCert: d.DtlsRemoteCert,
		SctpParameters: ifElse(d.SctpParameters != nil, func() *SctpParameters { return ref(*d.SctpParameters) }),
		SctpState:      d.SctpState,
	}
}

type PlainTransportData struct {
	Tuple          TransportTuple
	RtcpTuple      *TransportTuple
	SctpParameters *SctpParameters
	SctpState      SctpState
	SrtpParameters *SrtpParameters
}

func (d *PlainTransportData) clone() *PlainTransportData {
	if d == nil {
		return nil
	}
	return &PlainTransportData{
		Tuple:          d.Tuple,
		RtcpTuple:      ifElse(d.RtcpTuple != nil, func() *TransportTuple { return ref(*d.RtcpTuple) }),
		SctpParameters: ifElse(d.SctpParameters != nil, func() *SctpParameters { return ref(*d.SctpParameters) }),
		SctpState:      d.SctpState,
		SrtpParameters: ifElse(d.SrtpParameters != nil, func() *SrtpParameters { return ref(*d.SrtpParameters) }),
	}
}

type PipeTransportData struct {
	Tuple          TransportTuple
	SctpParameters *SctpParameters
	SctpState      SctpState
	SrtpParameters *SrtpParameters
}

func (d *PipeTransportData) clone() *PipeTransportData {
	if d == nil {
		return nil
	}
	return &PipeTransportData{
		Tuple:          d.Tuple,
		SctpParameters: ifElse(d.SctpParameters != nil, func() *SctpParameters { return ref(*d.SctpParameters) }),
		SctpState:      d.SctpState,
		SrtpParameters: ifElse(d.SrtpParameters != nil, func() *SrtpParameters { return ref(*d.SrtpParameters) }),
	}
}
