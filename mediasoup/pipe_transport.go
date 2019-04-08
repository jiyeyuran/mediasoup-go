package mediasoup

import (
	"fmt"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

type PipeTransport struct {
	*baseTransport
	logger logrus.FieldLogger
	data   PipeTransportData
}

func NewPipeTransport(data PipeTransportData, params CreateTransportParams) Transport {
	logger := TypeLogger("PipeTransport")

	logger.Debug("constructor()")

	return &PipeTransport{
		baseTransport: newTransport(params),
		logger:        logger,
		data:          data,
	}
}

func (t PipeTransport) Tuple() TransportTuple {
	return t.data.Tuple
}

/**
 * Provide the PipeTransport remote parameters.
 *
 * @param {String} ip - Remote IP.
 * @param {Number} port - Remote port.
 *
 * @override
 */
func (t *PipeTransport) Connect(params TransportConnectParams) (err error) {
	t.logger.Debug("connect()")

	resp := t.channel.Request("transport.connect", t.internal, params)

	return resp.Result(&t.data)
}

/**
 * Create a pipe Consumer.
 *
 * @param {String} producerId
 * @param {Object} [appData={}] - Custom app data.
 *
 * @override
 */
func (t *PipeTransport) Consume(params TransportConsumeParams) (consumer *Consumer, err error) {
	t.logger.Debug("consume()")

	producerId, appData := params.ProducerId, params.AppData

	producer := t.getProducerById(producerId)

	if producer == nil {
		err = fmt.Errorf(`Producer with id "%s" not found`, producerId)
	}

	rtpParameters := GetPipeConsumerRtpParameters(producer.ConsumableRtpParameters())

	internal := t.internal
	internal.ConsumerId = uuid.NewV4().String()
	internal.ProducerId = producerId

	reqData := map[string]interface{}{
		"kind":                   producer.Kind(),
		"rtpParameters":          rtpParameters,
		"type":                   "pipe",
		"consumableRtpEncodings": producer.ConsumableRtpParameters().Encodings,
	}

	resp := t.channel.Request("transport.consume", internal, reqData)

	var status struct {
		Paused         bool
		ProducerPaused bool
	}
	if err = resp.Result(&status); err != nil {
		return
	}

	data := ConsumerData{
		Kind:          producer.Kind(),
		RtpParameters: rtpParameters,
		Type:          "pipe",
	}

	consumer = NewConsumer(
		internal,
		data,
		t.channel,
		appData,
		status.Paused,
		status.ProducerPaused,
		nil,
	)

	t.consumers[consumer.Id()] = consumer
	consumer.On("@close", func() {
		delete(t.consumers, consumer.Id())
	})
	consumer.On("@producerclose", func() {
		delete(t.consumers, consumer.Id())
	})

	// Emit observer event.
	t.observer.SafeEmit("newconsumer", producer)

	return
}
