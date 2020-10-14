package mediasoup

type PipeTransportSpecificStat struct {
	Tuple TransportTuple `json:"tuple"`
}

type PipeTransport struct {
	ITransport
	logger Logger
}

func newPipeTransport(params transportParams) *PipeTransport {
	params.logger = NewLogger("PipeTransport")
	params.data.isPipeTransport = true

	transport := &PipeTransport{
		ITransport: newTransport(params),
		logger:     params.logger,
	}

	return transport
}
