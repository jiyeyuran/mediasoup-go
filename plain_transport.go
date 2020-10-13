package mediasoup

type PlainTransportSpecificStat struct {
	RtcpMux   bool            `json:"rtcp_mux"`
	Comedia   bool            `json:"comedia"`
	Tuple     TransportTuple  `json:"tuple"`
	RtcpTuple *TransportTuple `json:"rtcpTuple,omitempty"`
}

type PlainTransport struct {
	ITransport
}

func newPlainTransport(params transportParams) *PlainTransport {
	return &PlainTransport{
		ITransport: newTransport(params),
	}
}
