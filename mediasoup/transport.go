package mediasoup

import (
	"errors"
	"fmt"
	"runtime"
	"strings"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

type Transport interface {
	EventEmitter

	Id() string
	Closed() bool
	AppData() interface{}
	Observer() EventEmitter
	Close() error
	routerClosed()
	Dump() Response
	GetStats() Response
	Connect(TransportConnectParams) error
	Produce(TransportProduceParams) (*Producer, error)
	Consume(TransportConsumeParams) (*Consumer, error)
}

type baseTransport struct {
	EventEmitter
	logger                   logrus.FieldLogger
	internal                 Internal
	channel                  *Channel
	appData                  interface{}
	closed                   bool
	getRouterRtpCapabilities FetchRouterRtpCapabilitiesFunc
	getProducerById          FetchProducerFunc
	producers                map[string]*Producer
	consumers                map[string]*Consumer
	cnameForProducers        string
	observer                 EventEmitter
}

/**
 * new transport
 *
 * @emits routerclose
 * @emits @close
 * @emits @newproducer
 * @emits @producerclose
 */
func newTransport(params CreateTransportParams) *baseTransport {
	logger := TypeLogger("Transport")

	logger.Debug("constructor()")

	transport := &baseTransport{
		EventEmitter: NewEventEmitter(logger),
		logger:       logger,
		// - .routerId
		// - .transportId
		internal:                 params.Internal,
		channel:                  params.Channel,
		appData:                  params.AppData,
		getRouterRtpCapabilities: params.GetRouterRtpCapabilities,
		getProducerById:          params.GetProducerById,
		producers:                make(map[string]*Producer),
		consumers:                make(map[string]*Consumer),
		observer:                 NewEventEmitter(AppLogger()),
	}

	return transport
}

// Transport id
func (transport *baseTransport) Id() string {
	return transport.internal.TransportId
}

// Whether the Transport is closed.
func (transport *baseTransport) Closed() bool {
	return transport.closed
}

//App custom data.
func (transport *baseTransport) AppData() interface{} {
	return transport.appData
}

/**
 * Observer.
 *
 * @emits close
 * @emits {producer: Producer} newproducer
 * @emits {consumer: Consumer} newconsumer
 */
func (transport *baseTransport) Observer() EventEmitter {
	return transport.observer
}

// Close the Transport.
func (transport *baseTransport) Close() (err error) {
	if transport.closed {
		return
	}

	transport.logger.Debug("close()")

	transport.closed = true

	transport.RemoveAllListeners(transport.internal.TransportId)

	response := transport.channel.Request("transport.close", transport.internal, nil)

	if err = response.Err(); err != nil {
		return
	}

	for _, producer := range transport.producers {
		producer.TransportClosed()

		transport.Emit("@producerclose", producer)
	}
	transport.producers = make(map[string]*Producer)

	for _, consumer := range transport.consumers {
		consumer.TransportClosed()
	}
	transport.consumers = make(map[string]*Consumer)

	transport.Emit("@close")

	// Emit observer event.
	transport.observer.SafeEmit("close")

	return
}

/**
 * Router was closed.
 *
 * @virtual
 */
func (transport *baseTransport) routerClosed() {
	if transport.closed {
		return
	}

	transport.logger.Debug("routerClosed()")

	transport.closed = true

	// Remove notification subscriptions.
	transport.channel.RemoveAllListeners(transport.internal.TransportId)

	for _, producer := range transport.producers {
		producer.TransportClosed()

		transport.Emit("@producerclose", producer)
	}
	transport.producers = make(map[string]*Producer)

	for _, consumer := range transport.consumers {
		consumer.TransportClosed()
	}
	transport.consumers = make(map[string]*Consumer)

	transport.SafeEmit("routerclose")

	// Emit observer event.
	transport.observer.SafeEmit("close")
}

// Dump Transport.
func (transport *baseTransport) Dump() Response {
	transport.logger.Debug("dump()")

	return transport.channel.Request("transport.dump", transport.internal, nil)
}

// Get Transport stats.
func (transport *baseTransport) GetStats() Response {
	transport.logger.Debug("getStats()")

	return transport.channel.Request("transport.getStats", transport.internal, nil)
}

func (transport *baseTransport) Connect(TransportConnectParams) error {
	return errors.New("method not implemented in the subclass")
}

/**
 * Create a Producer.
 *
 * @param [id] - Producer id (just for PipeTransports).
 * @param kind - "audio"/"video".
 * @param rtpParameters - Remote RTP parameters.
 * @param [paused=false] - Whether the Consumer must start paused.
 * @param [appData={}] - Custom app data.
 */
func (transport *baseTransport) Produce(params TransportProduceParams) (producer *Producer, err error) {
	transport.logger.Debug("produce()")

	id := params.Id
	kind := params.Kind
	rtpParameters := params.RtpParameters
	paused := params.Paused
	appData := params.AppData

	if len(id) > 0 && transport.producers[id] != nil {
		err = NewTypeError(`a Producer with same id "%s" already exists`, id)
		return
	}

	if kind != "audio" && kind != "video" {
		err = NewTypeError(`invalid kind "%s"`, kind)
		return
	}

	pc, _, _, ok := runtime.Caller(1)
	// Don"t do this in PipeTransports since there we must keep CNAME value in
	// each Producer.
	if details := runtime.FuncForPC(pc); ok && details != nil &&
		!strings.Contains(details.Name(), "PipeTransport") {
		// If CNAME is given and we don"t have yet a CNAME for Producers in this
		// Transport, take it.
		if len(transport.cnameForProducers) == 0 && len(rtpParameters.Rtcp.Cname) > 0 {
			transport.cnameForProducers = rtpParameters.Rtcp.Cname
		} else if len(transport.cnameForProducers) == 0 {
			// Otherwise if we don"t have yet a CNAME for Producers and the RTP parameters
			// do not include CNAME, create a random one.
			transport.cnameForProducers = uuid.NewV4().String()[:8]
		}

		// Override Producer"s CNAME.
		rtpParameters.Rtcp.Cname = transport.cnameForProducers
	}

	routerRtpCapabilities := transport.getRouterRtpCapabilities()

	rtpMapping, err := GetProducerRtpParametersMapping(
		rtpParameters, routerRtpCapabilities)
	if err != nil {
		return
	}

	consumableRtpParameters, err := GetConsumableRtpParameters(
		kind, rtpParameters, routerRtpCapabilities, rtpMapping)
	if err != nil {
		return
	}

	internal := transport.internal
	if len(id) > 0 {
		internal.ProducerId = id
	} else {
		internal.ProducerId = uuid.NewV4().String()
	}

	reqData := map[string]interface{}{
		"kind":          kind,
		"rtpParameters": rtpParameters,
		"rtpMapping":    rtpMapping,
		"paused":        paused,
	}

	resp := transport.channel.Request("transport.produce", internal, reqData)

	var status struct {
		Type string
	}
	if err = resp.Result(&status); err != nil {
		return
	}

	producerData := ProducerData{
		Kind:                    kind,
		RtpParameters:           rtpParameters,
		Type:                    status.Type,
		ConsumableRtpParameters: consumableRtpParameters,
	}

	producer = NewProducer(internal, producerData, transport.channel, appData, paused)

	transport.producers[producer.Id()] = producer
	producer.On("@close", func() {
		delete(transport.producers, producer.Id())
	})

	transport.Emit("@newproducer", producer)

	// Emit observer event.
	transport.observer.SafeEmit("newproducer", producer)

	return
}

/**
 * Create a Consumer.
 *
 * @param producerId
 * @param rtpCapabilities - Remote RTP capabilities.
 * @param [paused=false] - Whether the Consumer must start paused.
 * @param [appData={}] - Custom app data.
 */
func (transport *baseTransport) Consume(params TransportConsumeParams) (consumer *Consumer, err error) {
	transport.logger.Debug("consume()")

	producerId := params.ProducerId
	rtpCapabilities := params.RtpCapabilities
	paused := params.Paused
	appData := params.AppData

	producer := transport.getProducerById(producerId)

	if producer == nil {
		err = fmt.Errorf(`Producer with id "%s" not found`, producerId)
		return
	}

	rtpParameters, err := GetConsumerRtpParameters(
		producer.ConsumableRtpParameters(), rtpCapabilities)
	if err != nil {
		return
	}

	internal := transport.internal
	internal.ConsumerId = uuid.NewV4().String()
	internal.ProducerId = producerId

	reqData := map[string]interface{}{
		"kind":                   producer.Kind(),
		"rtpParameters":          rtpParameters,
		"type":                   producer.Type(),
		"paused":                 paused,
		"consumableRtpEncodings": producer.ConsumableRtpParameters().Encodings,
	}

	resp := transport.channel.Request("transport.consume", internal, reqData)

	var status struct {
		Paused         bool
		ProducerPaused bool
		Score          *ConsumerScore
	}
	if err = resp.Result(&status); err != nil {
		return
	}

	data := ConsumerData{
		Kind:          producer.Kind(),
		RtpParameters: rtpParameters,
		Type:          producer.Type(),
	}

	consumer = NewConsumer(
		internal,
		data,
		transport.channel,
		appData,
		status.Paused,
		status.ProducerPaused,
		status.Score,
	)

	transport.consumers[consumer.Id()] = consumer
	consumer.On("@close", func() {
		delete(transport.consumers, consumer.Id())
	})
	consumer.On("@producerclose", func() {
		delete(transport.consumers, consumer.Id())
	})

	// Emit observer event.
	transport.observer.SafeEmit("newconsumer", producer)

	return
}
