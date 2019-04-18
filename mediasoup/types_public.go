package mediasoup

import "encoding/json"

type H map[string]interface{}

// Response from worker
type Response struct {
	data json.RawMessage
	err  error
}

func (r Response) Unmarshal(v interface{}) error {
	if r.err != nil {
		return r.err
	}
	return json.Unmarshal([]byte(r.data), v)
}

func (r Response) Data() []byte {
	return []byte(r.data)
}

func (r Response) Err() error {
	return r.err
}

// []VolumeInfo is the parameter of event "volumes" emitted by AudioLevelObserver
type VolumeInfo struct {
	Producer *Producer
	Volume   uint8
}

// VideoLayer is the parameter of event "layerschange" emitted by Consumer
type VideoLayer struct {
	SpatialLayer uint8 `json:"spatialLayer"`
}

// VideoOrientation is the parameter of event "videoorientationchange" emitted by Producer
type VideoOrientation struct {
	Camera   bool  `json:"camera,omitempty"`
	Flip     bool  `json:"flip,omitempty"`
	Rotation uint8 `json:"rotation,omitempty"`
}

type ProducerScore struct {
	Score uint8  `json:"score"`
	Ssrc  uint32 `json:"ssrc"`
	Rid   uint32 `json:"rid,omitempty"`
}

type ConsumerScore struct {
	Producer uint8 `json:"producer"`
	Consumer uint8 `json:"consumer"`
}

type PipeTransportData struct {
	Tuple TransportTuple `json:"tuple,omitempty"`
}

type PlainTransportData struct {
	RtcpMux     bool            `json:"rtcpMux,omitempty"`
	Comedia     bool            `json:"comedia,omitempty"`
	MultiSource bool            `json:"multiSource,omitempty"`
	Tuple       TransportTuple  `json:"tuple,omitempty"`
	RtcpTuple   *TransportTuple `json:"rtcpTuple,omitempty"`
}

type WebRtcTransportData struct {
	IceRole          string          `json:"iceRole,omitempty"`
	IceParameters    IceParameters   `json:"iceParameters,omitempty"`
	IceCandidates    []IceCandidate  `json:"iceCandidates,omitempty"`
	IceState         string          `json:"iceState,omitempty"`
	IceSelectedTuple *TransportTuple `json:"iceSelectedTuple,omitempty"`
	DtlsParameters   DtlsParameters  `json:"dtlsParameters,omitempty"`
	DtlsState        string          `json:"dtlsState,omitempty"`
	DtlsRemoteCert   string          `json:"dtlsRemoteCert,omitempty"`
}

type TransportTuple struct {
	LocalIp    string `json:"localIp,omitempty"`
	LocalPort  uint16 `json:"localPort,omitempty"`
	RemoteIp   string `json:"remoteIp,omitempty"`
	RemotePort uint16 `json:"remotePort,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
}

type IceParameters struct {
	UsernameFragment string `json:"usernameFragment,omitempty"`
	Password         string `json:"password,omitempty"`
	IceLite          bool   `json:"iceLite,omitempty"`
}

type IceCandidate struct {
	Foundation string `json:"foundation,omitempty"`
	Priority   uint32 `json:"priority,omitempty"`
	Ip         string `json:"ip,omitempty"`
	Port       uint16 `json:"port,omitempty"`
	Type       string `json:"type,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
	TcpType    string `json:"tcpType,omitempty"`
}

type DtlsParameters struct {
	Role         string            `json:"role,omitempty"`
	Fingerprints []DtlsFingerprint `json:"fingerprints,omitempty"`
}

type DtlsFingerprint struct {
	Algorithm string `json:"algorithm,omitempty"`
	Value     string `json:"value,omitempty"`
}

type CreateWebRtcTransportParams struct {
	ListenIps []ListenIp  `json:"listenIps,omitempty"`
	EnableUdp bool        `json:"enableUdp,omitempty"`
	EnableTcp bool        `json:"enableTcp,omitempty"`
	PreferUdp bool        `json:"preferUdp,omitempty"`
	PreferTcp bool        `json:"preferTcp,omitempty"`
	AppData   interface{} `json:"appData,omitempty"`
}

type CreatePlainRtpTransportParams struct {
	ListenIp    ListenIp    `json:"listenIp,omitempty"`
	RtcpMux     bool        `json:"rtcpMux"` //should set explicitly
	Comedia     bool        `json:"comedia,omitempty"`
	MultiSource bool        `json:"multiSource,omitempty"`
	AppData     interface{} `json:"appData,omitempty"`
}

type CreatePipeTransportParams struct {
	ListenIp ListenIp    `json:"listenIp,omitempty"`
	AppData  interface{} `json:"appData,omitempty"`
}

type PipeToRouterParams struct {
	ProducerId string   `json:"producerId,omitempty"`
	Router     *Router  `json:"router,omitempty"`
	ListenIp   ListenIp `json:"listenIp,omitempty"`
}

type ListenIp struct {
	Ip          string `json:"ip,omitempty"`
	AnnouncedIp string `json:"announcedIp,omitempty"`
}

type CreateAudioLevelObserverParams struct {
	MaxEntries uint32 `json:"maxEntries,omitempty"`
	Threshold  int    `json:"threshold,omitempty"`
	Interval   uint32 `json:"interval,omitempty"`
}

type TransportStat struct {
	Type                     string `json:"type,omitempty"`
	TransportId              string `json:"transportId,omitempty"`
	Timestamp                uint32 `json:"timestamp,omitempty"`
	BytesReceived            uint32 `json:"bytesReceived,omitempty"`
	BytesSent                uint32 `json:"bytesSent,omitempty"`
	AvailableIncomingBitrate uint32 `json:"availableIncomingBitrate,omitempty"`
	AvailableOutgoingBitrate uint32 `json:"availableOutgoingBitrate,omitempty"`
	MaxIncomingBitrate       uint32 `json:"maxIncomingBitrate,omitempty"`

	// webrtc transport
	IceRole          string          `json:"iceRole,omitempty"`
	IceState         string          `json:"iceState,omitempty"`
	DtlsState        string          `json:"dtlsState,omitempty"`
	IceSelectedTuple *TransportTuple `json:"iceSelectedTuple,omitempty"`

	// plain transport
	Tuple     *TransportTuple `json:"tuple,omitempty"`
	RtcpTuple *TransportTuple `json:"rtcpTuple,omitempty"`
}
