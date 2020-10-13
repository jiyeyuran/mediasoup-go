package mediasoup

type DirectTransportOptions struct {
	/**
	 * Maximum allowed size for direct messages sent from DataProducers.
	 * Default 262144.
	 */
	MaxMessageSize int

	/**
	 * Custom application data.
	 */
	AppData interface{}
}

type DirectTransport struct {
	ITransport
}

func newDirectTransport(params transportParams) *DirectTransport {
	return &DirectTransport{
		ITransport: newTransport(params),
	}
}

func (transport *DirectTransport) ConsumeData(options DataConsumerOptions) (dataConsumer *DataConsumer, err error) {
	options.isDirectTransport = true

	return transport.ITransport.ConsumeData(options)
}

func (transport *DirectTransport) ProduceData(options DataProducerOptions) (dataProducer *DataProducer, err error) {
	options.isDirectTransport = true

	return transport.ITransport.ProduceData(options)
}
