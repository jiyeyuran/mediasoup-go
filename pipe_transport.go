package mediasoup

type PipeTransportSpecificStat struct {
	Tuple TransportTuple `json:"tuple"`
}

type PipeTransport struct {
	ITransport
}

func newPipeTransport(params transportParams) *PipeTransport {
	return &PipeTransport{
		ITransport: newTransport(params),
	}
}

func (transport *PipeTransport) Produce(options ProducerOptions) (producer *Producer, err error) {
	options.isPipeTransport = true

	return transport.ITransport.Produce(options)
}
