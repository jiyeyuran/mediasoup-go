package mediasoup

type PipeTransportSpecificStat struct {
	Tuple TransportTuple `json:"tuple"`
}

type PipeTransport struct {
	IEventEmitter
}
