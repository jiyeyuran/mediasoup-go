package mediasoup

// WebRtcServerOptions represents the configuration options for a WebRTC server.
// The WebRTC server is a singleton that allows endpoints to share the same
// UDP/TCP ports and ICE candidates.
type WebRtcServerOptions struct {
	// ListenInfos contains an array of TransportListenInfo objects representing
	// the network interfaces and protocols to listen on. This is required.
	ListenInfos []*TransportListenInfo `json:"listenInfos,omitempty"`

	// AppData is custom application data that can be attached to the WebRTC server.
	// The app can access this data at any time. This is optional.
	AppData H `json:"appData,omitempty"`
}

type WebRtcServerDump struct {
	Id                        string                `json:"id,omitempty"`
	UdpSockets                []IpPort              `json:"udpSockets,omitempty"`
	TcpServers                []IpPort              `json:"tcpServers,omitempty"`
	WebRtcTransportIds        []string              `json:"webRtcTransportIds,omitempty"`
	LocalIceUsernameFragments []IceUserNameFragment `json:"localIceUsernameFragments,omitempty"`
	TupleHashes               []TupleHash           `json:"tupleHashes,omitempty"`
}

type IpPort struct {
	Ip   string `json:"ip,omitempty"`
	Port uint16 `json:"port,omitempty"`
}

type IceUserNameFragment struct {
	LocalIceUsernameFragment string `json:"localIceUsernameFragment,omitempty"`
	WebRtcTransportId        string `json:"webRtcTransportId,omitempty"`
}

type TupleHash struct {
	TupleHash         uint64 `json:"tupleHash,omitempty"`
	WebRtcTransportId string `json:"webRtcTransportId,omitempty"`
}
