package mediasoup

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
)

type Producer struct {
	EventEmitter
	logger   logrus.FieldLogger
	internal Internal
	data     *ProducerData
	channel  *Channel
	appData  interface{}
	paused   bool
	closed   bool
	score    []ProducerScore
	observer EventEmitter
}

/**
 * New Producer.
 *
 * @emits transportclose
 * @emits {Array<Object>} score
 * @emits {Object} videoorientationchange
 * @emits @close
 */
func NewProducer(
	internal Internal,
	data *ProducerData,
	channel *Channel,
	appData interface{},
	paused bool,
) *Producer {
	logger := TypeLogger("Producer")

	logger.Debug("constructor()")

	producer := &Producer{
		EventEmitter: NewEventEmitter(logger),
		logger:       logger,
		// - .routerId
		// - .transportId
		// - .producerId
		internal: internal,
		data:     data,
		channel:  channel,
		appData:  appData,
		paused:   paused,
		observer: NewEventEmitter(AppLogger()),
	}

	producer.handleWorkerNotifications()

	return producer
}

// Producer id
func (producer *Producer) Id() string {
	return producer.internal.ProducerId
}

// Whether the Producer is closed.
func (producer *Producer) Closed() bool {
	return producer.closed
}

// Media kind.
func (producer *Producer) Kind() string {
	return producer.data.Kind
}

// RTP parameters.
func (producer *Producer) RtpParameters() RtpProducerCapabilities {
	return producer.data.RtpParameters
}

// Producer type.
// It can be 'simple', 'simulcast' or 'svc'.
func (producer *Producer) Type() string {
	return producer.data.Type
}

// Consumable RTP parameters.
func (producer *Producer) ConsumableRtpParameters() RtpConsumerCapabilities {
	return producer.data.ConsumableRtpParameters
}

// Whether the Producer is paused.
func (producer *Producer) Paused() bool {
	return producer.paused
}

// Producer score list.
func (producer *Producer) Score() []ProducerScore {
	return producer.score
}

//App custom data.
func (producer *Producer) AppData() interface{} {
	return producer.appData
}

/**
 * Observer.
 *
 * @emits close
 * @emits pause
 * @emits resume
 * @emits {[]ProducerScore} score
 * @emits {Object} videoorientationchange
 */
func (producer *Producer) Observer() EventEmitter {
	return producer.observer
}

// Close the Producer.
func (producer *Producer) Close() (err error) {
	if producer.closed {
		return
	}

	producer.closed = true

	producer.logger.Debug("close()")

	producer.RemoveAllListeners(producer.internal.ProducerId)

	response := producer.channel.Request("producer.close", producer.internal, nil)

	if err = response.Err(); err != nil {
		return
	}

	producer.Emit("@close")

	// Emit observer event.
	producer.observer.SafeEmit("close")

	return
}

// Transport was closed.
func (producer *Producer) TransportClosed() {
	if producer.closed {
		return
	}

	producer.closed = true

	producer.logger.Debug("transportClosed()")

	producer.SafeEmit("transportclose")

	// Emit observer event.
	producer.observer.SafeEmit("transportclose")
}

// Dump Producer.
func (producer *Producer) Dump() Response {
	producer.logger.Debug("dump()")

	return producer.channel.Request("producer.dump", producer.internal, nil)
}

// Get Producer stats.
func (producer *Producer) GetStats() Response {
	producer.logger.Debug("getStats()")

	return producer.channel.Request("producer.getStats", producer.internal, nil)
}

// Pause the Producer.
func (producer *Producer) Pause() (err error) {
	producer.logger.Debug("pause()")

	wasPaused := producer.paused

	response := producer.channel.Request("producer.pause", producer.internal, nil)

	if err = response.Err(); err != nil {
		return
	}

	producer.paused = true

	// Emit observer event.
	if !wasPaused {
		producer.observer.SafeEmit("pause")
	}

	return
}

// Resume the Producer.
func (producer *Producer) Resume() (err error) {
	producer.logger.Debug("resume()")

	wasPaused := producer.paused

	response := producer.channel.Request("producer.resume", producer.internal, nil)

	if err = response.Err(); err != nil {
		return
	}

	producer.paused = false

	// Emit observer event.
	if wasPaused {
		producer.observer.SafeEmit("resume")
	}

	return
}

func (producer *Producer) handleWorkerNotifications() {
	producer.On(producer.internal.ProducerId, func(event string, data json.RawMessage) {
		switch event {
		case "score":
			producer.score = []ProducerScore{}

			json.Unmarshal([]byte(data), &producer.score)

			producer.SafeEmit("score", producer.score)

			// Emit observer event.
			producer.observer.SafeEmit("score", producer.score)

		case "videoorientationchange":
			orientation := VideoOrientation{}

			json.Unmarshal([]byte(data), &orientation)

			producer.SafeEmit("videoorientationchange", orientation)

			// Emit observer event.
			producer.observer.SafeEmit("videoorientationchange", orientation)

		default:
			producer.logger.Error(`ignoring unknown event "%s"`, event)
		}
	})
}
