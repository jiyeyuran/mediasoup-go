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

type VolumeInfo struct {
	Producer *Producer `json:"producer,omitempty"`
	Volume   uint8     `json:"volume,omitempty"`
}

type ProducerData struct {
	Kind                    string
	Type                    string
	RtpParameters           RtpProducerCapabilities
	ConsumableRtpParameters RtpConsumerCapabilities
}

type ProducerScore struct {
	Score uint8  `json:"score,omitempty"`
	Ssrc  uint32 `json:"ssrc,omitempty"`
	Rid   uint32 `json:"rid,omitempty"`
}

type VideoOrientation struct {
	Camera   bool  `json:"camera,omitempty"`
	Flip     bool  `json:"flip,omitempty"`
	Rotation uint8 `json:"rotation,omitempty"`
}

type ConsumerData struct {
	Kind          string
	Type          string
	RtpParameters RtpConsumerCapabilities
}

type ConsumerScore struct {
	Producer uint8 `json:"producer,omitempty"`
	Consumer uint8 `json:"consumer,omitempty"`
}

type VideoLayer struct {
	SpatialLayer uint8 `json:"spatialLayer"`
}

type FetchRouterRtpCapabilitiesFunc func() RtpCapabilities

type CreateProducerParams struct {
	Id            string                  `json:"id,omitempty"`
	Kind          string                  `json:"kind,omitempty"`
	RtpParameters RtpProducerCapabilities `json:"rtpParameters,omitempty"`
	Paused        bool                    `json:"paused,omitempty"`
	AppData       interface{}             `json:"appData,omitempty"`
}

type CreateConsumerParams struct {
	ProducerId      string                  `json:"producerId,omitempty"`
	RtpCapabilities RtpConsumerCapabilities `json:"rtpCapabilities,omitempty"`
	Paused          bool                    `json:"paused,omitempty"`
	AppData         interface{}             `json:"appData,omitempty"`
}

type CreateTransportParams struct {
	Internal                 Internal
	Channel                  *Channel
	AppData                  interface{}
	GetRouterRtpCapabilities FetchRouterRtpCapabilitiesFunc
	GetProducerById          FetchProducerFunc
}

type TransportConnectParams struct {
	// pipe and plain transport
	Ip   string `json:"ip,omitempty"`
	Port int    `json:"port,omitempty"`
	// plain transport
	RtcpPort int `json:"rtcpPort,omitempty"`
	// webrtc transport
	DtlsParamters *DtlsParamters `json:"dtlsParamters,omitempty"`
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

type WebrtcTransportData struct {
	IceRole          string         `json:"iceRole,omitempty"`
	IceParameters    IceParameters  `json:"iceParameters,omitempty"`
	IceCandidates    []IceCandidate `json:"iceCandidates,omitempty"`
	IceState         string         `json:"iceState,omitempty"`
	IceSelectedTuple TransportTuple `json:"iceSelected_tuple,omitempty"`
	DtlsState        string         `json:"dtlsState,omitempty"`
	DtlsRemoteCert   string         `json:"dtlsRemoteCert,omitempty"`
}

type TransportTuple struct {
	LocalIp    string `json:"localIp,omitempty"`
	LocalPort  string `json:"localPort,omitempty"`
	RemoteIp   string `json:"remoteIp,omitempty"`
	RemotePort string `json:"remotePort,omitempty"`
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

type DtlsParamters struct {
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
