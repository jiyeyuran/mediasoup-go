package mediasoup

import "encoding/json"

type H map[string]interface{}

// []VolumeInfo is the parameter of event "volumes" emitted by AudioLevelObserver
type VolumeInfo struct {
	// Producer *Producer
	Volume uint8
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

type CreateWebRtcTransportParams struct {
	ListenIps []TransportListenIp `json:"listenIps,omitempty"`
	EnableUdp bool                `json:"enableUdp,omitempty"`
	EnableTcp bool                `json:"enableTcp,omitempty"`
	PreferUdp bool                `json:"preferUdp,omitempty"`
	PreferTcp bool                `json:"preferTcp,omitempty"`
	AppData   interface{}         `json:"appData,omitempty"`
}

type CreatePlainRtpTransportParams struct {
	ListenIp    TransportListenIp `json:"listenIp,omitempty"`
	RtcpMux     bool              `json:"rtcpMux"` //should set explicitly
	Comedia     bool              `json:"comedia,omitempty"`
	MultiSource bool              `json:"multiSource,omitempty"`
	AppData     interface{}       `json:"appData,omitempty"`
}

type CreatePipeTransportParams struct {
	ListenIp TransportListenIp `json:"listenIp,omitempty"`
	AppData  interface{}       `json:"appData,omitempty"`
}

type PipeToRouterParams struct {
	ProducerId string            `json:"producerId,omitempty"`
	Router     *Router           `json:"router,omitempty"`
	ListenIp   TransportListenIp `json:"listenIp,omitempty"`
}

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

type internalData struct {
	RouterId       string `json:"routerId,omitempty"`
	TransportId    string `json:"transportId,omitempty"`
	ProducerId     string `json:"producerId,omitempty"`
	ConsumerId     string `json:"consumerId,omitempty"`
	DataProducerId string `json:"dataProducerId,omitempty"`
	DataConsumerId string `json:"dataConsumerId,omitempty"`
	RtpObserverId  string `json:"rtpObserverId,omitempty"`
}
