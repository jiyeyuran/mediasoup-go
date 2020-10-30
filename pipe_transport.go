package mediasoup

import (
	"encoding/json"
	"fmt"

	uuid "github.com/satori/go.uuid"
)

type PipeTransportOptions struct {
	/**
	 * Listening IP address.
	 */
	ListenIp TransportListenIp `json:"listenIp,omitempty"`

	/**
	 * Create a SCTP association. Default false.
	 */
	EnableSctp bool `json:"enableSctp,omitempty"`

	/**
	 * SCTP streams number.
	 */
	NumSctpStreams NumSctpStreams `json:"numSctpStreams,omitempty"`

	/**
	 * Maximum allowed size for SCTP messages sent by DataProducers.
	 * Default 268435456.
	 */
	MaxSctpMessageSize int `json:"maxSctpMessageSize,omitempty"`

	/**
	 * Maximum SCTP send buffer used by DataConsumers.
	 * Default 268435456.
	 */
	SctpSendBufferSize int `json:"sctpSendBufferSize,omitempty"`

	/**
	 * Enable RTX and NACK for RTP retransmission. Useful if both Routers are
	 * located in different hosts and there is packet lost in the link. For this
	 * to work, both PipeTransports must enable this setting. Default false.
	 */
	EnableRtx bool `json:"enableRtx,omitempty"`

	/**
	 * Enable SRTP. Useful to protect the RTP and RTCP traffic if both Routers
	 * are located in different hosts. For this to work, connect() must be called
	 * with remote SRTP parameters. Default false.
	 */
	EnableSrtp bool `json:"enableSrtp,omitempty"`

	/**
	 * Custom application data.
	 */
	AppData interface{} `json:"appData,omitempty"`
}

type pipeTransortData struct {
	Tuple          TransportTuple  `json:"tuple,omitempty"`
	SctpParameters SctpParameters  `json:"sctpParameters,omitempty"`
	SctpState      SctpState       `json:"sctpState,omitempty"`
	Rtx            bool            `json:"rtx,omitempty"`
	SrtpParameters *SrtpParameters `json:"srtpParameters,omitempty"`
}

/**
 * PipeTransport
 * @emits sctpstatechange - (sctpState: SctpState)
 * @emits trace - (trace: TransportTraceEventData)
 */
type PipeTransport struct {
	ITransport
	logger          Logger
	internal        internalData
	data            pipeTransortData
	channel         *Channel
	payloadChannel  *PayloadChannel
	getProducerById func(string) *Producer
}

func newPipeTransport(params transportParams) ITransport {
	data := params.data.(pipeTransortData)
	params.data = transportData{
		sctpParameters: data.SctpParameters,
		sctpState:      data.SctpState,
		transportType:  TransportType_Pipe,
	}
	params.logger = NewLogger("PipeTransport")

	transport := &PipeTransport{
		ITransport:      newTransport(params),
		logger:          params.logger,
		internal:        params.internal,
		data:            data,
		channel:         params.channel,
		payloadChannel:  params.payloadChannel,
		getProducerById: params.getProducerById,
	}

	transport.handleWorkerNotifications()

	return transport
}

/**
 * Transport tuple.
 */
func (t PipeTransport) Tuple() TransportTuple {
	return t.data.Tuple
}

/**
 * SCTP parameters.
 */
func (t PipeTransport) SctpParameters() SctpParameters {
	return t.data.SctpParameters
}

/**
 * SCTP state.
 */
func (t PipeTransport) SctpState() SctpState {
	return t.data.SctpState
}

/**
 * SRTP parameters.
 */
func (t PipeTransport) SrtpParameters() *SrtpParameters {
	return t.data.SrtpParameters
}

/**
 * Observer.
 *
 * @override
 * @emits close
 * @emits newproducer - (producer: Producer)
 * @emits newconsumer - (consumer: Consumer)
 * @emits newdataproducer - (dataProducer: DataProducer)
 * @emits newdataconsumer - (dataConsumer: DataConsumer)
 * @emits sctpstatechange - (sctpState: SctpState)
 * @emits trace - (trace: TransportTraceEventData)
 */
func (transport *PipeTransport) Observer() IEventEmitter {
	return transport.ITransport.Observer()
}

/**
 * Close the PipeTransport.
 *
 * @override
 */
func (transport *PipeTransport) Close() {
	if transport.Closed() {
		return
	}

	if len(transport.data.SctpState) > 0 {
		transport.data.SctpState = SctpState_Closed
	}

	transport.ITransport.Close()
}

/**
 * Router was closed.
 *
 * @override
 */
func (transport *PipeTransport) routerClosed() {
	if transport.Closed() {
		return
	}

	if len(transport.data.SctpState) > 0 {
		transport.data.SctpState = SctpState_Closed
	}

	transport.ITransport.routerClosed()
}

/**
 * Provide the PlainTransport remote parameters.
 *
 * @override
 */
func (transport *PipeTransport) Connect(options TransportConnectOptions) (err error) {
	transport.logger.Debug("connect()")

	reqData := TransportConnectOptions{
		Ip:             options.Ip,
		Port:           options.Port,
		SrtpParameters: options.SrtpParameters,
	}
	resp := transport.channel.Request("transport.connect", transport.internal, reqData)

	var data struct {
		Tuple TransportTuple
	}
	if err = resp.Unmarshal(&data); err != nil {
		return
	}

	// Update data.
	transport.data.Tuple = data.Tuple

	return nil
}

/**
 * Create a pipe Consumer.
 *
 * @override
 */
func (transport *PipeTransport) Consume(options ConsumerOptions) (consumer *Consumer, err error) {
	transport.logger.Debug("consume()")

	producerId := options.ProducerId
	appData := options.AppData

	producer := transport.getProducerById(producerId)

	if producer == nil {
		err = fmt.Errorf(`Producer with id "%s" not found`, producerId)
		return
	}

	rtpParameters := getPipeConsumerRtpParameters(producer.ConsumableRtpParameters(), transport.data.Rtx)
	internal := transport.internal
	internal.ConsumerId = uuid.NewV4().String()
	internal.ProducerId = producerId

	reqData := H{
		"kind":                   producer.Kind(),
		"rtpParameters":          rtpParameters,
		"type":                   "pipe",
		"consumableRtpEncodings": producer.ConsumableRtpParameters().Encodings,
	}
	resp := transport.channel.Request("transport.consume", internal, reqData)

	var status struct {
		Paused         bool
		ProducerPaused bool
	}
	if err = resp.Unmarshal(&status); err != nil {
		return
	}

	consumerData := consumerData{
		Kind:          producer.Kind(),
		RtpParameters: rtpParameters,
		Type:          "pipe",
	}
	consumer = newConsumer(consumerParams{
		internal:       internal,
		data:           consumerData,
		channel:        transport.channel,
		payloadChannel: transport.payloadChannel,
		appData:        appData,
		paused:         status.Paused,
		producerPaused: status.ProducerPaused,
	})

	baseTransport := transport.ITransport.(*Transport)

	baseTransport.consumers.Store(consumer.Id(), consumer)
	consumer.On("@close", func() {
		baseTransport.consumers.Delete(consumer.Id())
	})
	consumer.On("@producerclose", func() {
		baseTransport.consumers.Delete(consumer.Id())
	})

	// Emit observer event.
	transport.Observer().SafeEmit("newconsumer", consumer)

	return
}

func (transport *PipeTransport) handleWorkerNotifications() {
	transport.channel.On(transport.Id(), func(event string, data []byte) {
		switch event {
		case "sctpstatechange":
			var result struct {
				SctpState SctpState
			}
			json.Unmarshal(data, &result)

			transport.data.SctpState = result.SctpState

			transport.SafeEmit("sctpstatechange", result.SctpState)

			// Emit observer event.
			transport.Observer().SafeEmit("sctpstatechange", result.SctpState)

		case "trace":
			var result TransportTraceEventData
			json.Unmarshal(data, &result)

			transport.SafeEmit("trace", result)

			// Emit observer event.
			transport.Observer().SafeEmit("trace", result)

		default:
			transport.logger.Error(`ignoring unknown event "%s"`, event)
		}
	})
}
