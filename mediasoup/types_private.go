package mediasoup

type internalData struct {
	RouterId      string `json:"routerId,omitempty"`
	TransportId   string `json:"transportId,omitempty"`
	ProducerId    string `json:"producerId,omitempty"`
	ConsumerId    string `json:"consumerId,omitempty"`
	RtpObserverId string `json:"rtpObserverId,omitempty"`
}

type routerData struct {
	RtpCapabilities RtpCapabilities
}

type producerData struct {
	Kind                    string
	Type                    string
	RtpParameters           RtpRemoteCapabilities
	ConsumableRtpParameters RtpRemoteCapabilities
}

type consumerData struct {
	Kind          string
	Type          string
	RtpParameters RtpRemoteCapabilities
}

type transportProduceParams struct {
	Id            string                `json:"id,omitempty"`
	Kind          string                `json:"kind,omitempty"`
	RtpParameters RtpRemoteCapabilities `json:"rtpParameters,omitempty"`
	Paused        bool                  `json:"paused,omitempty"`
	AppData       interface{}           `json:"appData,omitempty"`
}

type transportConsumeParams struct {
	ProducerId      string          `json:"producerId,omitempty"`
	RtpCapabilities RtpCapabilities `json:"rtpCapabilities,omitempty"`
	Paused          bool            `json:"paused,omitempty"`
	AppData         interface{}     `json:"appData,omitempty"`
}

type createTransportParams struct {
	Internal                 internalData
	Channel                  *Channel
	AppData                  interface{}
	GetRouterRtpCapabilities fetchRouterRtpCapabilitiesFunc
	GetProducerById          fetchProducerFunc
}

type transportConnectParams struct {
	// pipe and plain transport
	Ip   string `json:"ip,omitempty"`
	Port uint16 `json:"port,omitempty"`
	// plain transport
	RtcpPort uint16 `json:"rtcpPort,omitempty"`
	// webrtc transport
	DtlsParameters *DtlsParameters `json:"dtlsParameters,omitempty"`
}

type fetchProducerFunc func(producerId string) *Producer

type fetchRouterRtpCapabilitiesFunc func() RtpCapabilities
