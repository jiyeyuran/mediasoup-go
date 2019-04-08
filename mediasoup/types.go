package mediasoup

import "encoding/json"

type H map[string]interface{}

type Internal struct {
	RouterId      string `json:"routerId,omitempty"`
	TransportId   string `json:"transportId,omitempty"`
	ProducerId    string `json:"producerId,omitempty"`
	ConsumerId    string `json:"consumerId,omitempty"`
	RtpObserverId string `json:"rtpObserverId,omitempty"`
}

type Response struct {
	data json.RawMessage
	err  error
}

func (r Response) Result(v interface{}) error {
	if r.err != nil {
		return r.err
	}
	return json.Unmarshal([]byte(r.data), v)
}

func (r Response) Err() error {
	return r.err
}

type FetchProducerFunc func(producerId string) *Producer

type FetchRouterRtpCapabilitiesFunc func() RtpCapabilities

type VolumeInfo struct {
	Producer *Producer `json:"producer,omitempty"`
	Volume   uint8     `json:"volume,omitempty"`
}

type VideoLayer struct {
	SpatialLayer uint8 `json:"spatialLayer"`
}

type VideoOrientation struct {
	Camera   bool  `json:"camera,omitempty"`
	Flip     bool  `json:"flip,omitempty"`
	Rotation uint8 `json:"rotation,omitempty"`
}

type RouterData struct {
	RtpCapabilities RtpCapabilities
}

type ProducerData struct {
	Kind                    string
	Type                    string
	RtpParameters           RtpRemoteCapabilities
	ConsumableRtpParameters RtpRemoteCapabilities
}

type ConsumerData struct {
	Kind          string
	Type          string
	RtpParameters RtpRemoteCapabilities
}

type ProducerScore struct {
	Score uint8  `json:"score,omitempty"`
	Ssrc  uint32 `json:"ssrc,omitempty"`
	Rid   uint32 `json:"rid,omitempty"`
}

type ConsumerScore struct {
	Producer uint8 `json:"producer,omitempty"`
	Consumer uint8 `json:"consumer,omitempty"`
}

type transportProduceParams struct {
	Id            string                `json:"id,omitempty"`
	Kind          string                `json:"kind,omitempty"`
	RtpParameters RtpRemoteCapabilities `json:"rtpParameters,omitempty"`
	Paused        bool                  `json:"paused,omitempty"`
	AppData       interface{}           `json:"appData,omitempty"`
}

type transportConsumeParams struct {
	ProducerId      string                `json:"producerId,omitempty"`
	RtpCapabilities RtpRemoteCapabilities `json:"rtpCapabilities,omitempty"`
	Paused          bool                  `json:"paused,omitempty"`
	AppData         interface{}           `json:"appData,omitempty"`
}

type createTransportParams struct {
	Internal                 Internal
	Channel                  *Channel
	AppData                  interface{}
	GetRouterRtpCapabilities FetchRouterRtpCapabilitiesFunc
	GetProducerById          FetchProducerFunc
}

type transportConnectParams struct {
	// pipe and plain transport
	Ip   string `json:"ip,omitempty"`
	Port int    `json:"port,omitempty"`
	// plain transport
	RtcpPort int `json:"rtcpPort,omitempty"`
	// webrtc transport
	DtlsParameters *DtlsParameters `json:"dtlsParameters,omitempty"`
}

type PipeTransportData struct {
	Tuple TransportTuple `json:"tuple,omitempty"`
}

type PlainTransportData struct {
	RtcpMux     bool           `json:"rtcpMux,omitempty"`
	Comedia     string         `json:"comedia,omitempty"`
	MultiSource bool           `json:"multiSource,omitempty"`
	Tuple       TransportTuple `json:"tuple,omitempty"`
	RtcpTuple   TransportTuple `json:"rtcpTuple,omitempty"`
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
	LocalPort  int    `json:"localPort,omitempty"`
	RemoteIp   string `json:"remoteIp,omitempty"`
	RemotePort int    `json:"remotePort,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
}

type IceParameters struct {
	UsernameFragment string `json:"usernameFragment,omitempty"`
	Password         string `json:"password,omitempty"`
	IceLite          bool   `json:"iceLite,omitempty"`
}

type IceCandidate struct {
	Foundation uint32 `json:"foundation,omitempty"`
	Priority   uint32 `json:"priority,omitempty"`
	Ip         string `json:"ip,omitempty"`
	Port       int    `json:"port,omitempty"`
	Type       string `json:"type,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
	TcpType    string `json:"tcpType,omitempty"`
}

type DtlsParameters struct {
	Role         string           `json:"role,omitempty"`
	Fingerprints DtlsFingerprints `json:"fingerprints,omitempty"`
}

type DtlsFingerprints struct {
	Sha1   string `json:"sha-1,omitempty"`
	Sha224 string `json:"sha-224,omitempty"`
	Sha256 string `json:"sha-256,omitempty"`
	Sha384 string `json:"sha-384,omitempty"`
	Sha512 string `json:"sha-512,omitempty"`
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
	RtcpMux     bool        `json:"rtcpMux,omitempty"`
	Comedia     string      `json:"comedia,omitempty"`
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
	MaxEntries int `json:"maxEntries,omitempty"`
	Threshold  int `json:"threshold,omitempty"`
	Interval   int `json:"interval,omitempty"`
}
