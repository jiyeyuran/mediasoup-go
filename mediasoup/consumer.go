package mediasoup

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
)

type Consumer struct {
	EventEmitter
	logger         logrus.FieldLogger
	internal       Internal
	data           ConsumerData
	channel        *Channel
	appData        interface{}
	paused         bool
	closed         bool
	producerPaused bool
	score          ConsumerScore
	// Current video layers (just for video with simulcast or SVC).
	currentLayers *VideoLayer
	observer      EventEmitter
}

/**
 * New Consumer
 *
 * @emits transportclose
 * @emits consumerclose
 * @emits consumerpause
 * @emits consumerresume
 * @emits {consumer: Number, consumer: Number} score
 * @emits {spatialLayer: Number|Null} layerschange
 * @emits @close
 * @emits @consumerclose
 */
func NewConsumer(
	internal Internal,
	data ConsumerData,
	channel *Channel,
	appData interface{},
	paused bool,
	producerPaused bool,
	score ConsumerScore,
) *Consumer {
	logger := TypeLogger("Consumer")

	logger.Debug("constructor()")

	consumer := &Consumer{
		EventEmitter: NewEventEmitter(logger),
		logger:       logger,
		// - .routerId
		// - .transportId
		// - .consumerId
		// - .consumerId
		internal:       internal,
		data:           data,
		channel:        channel,
		appData:        appData,
		paused:         paused,
		producerPaused: producerPaused,
		score:          score,
		observer:       NewEventEmitter(AppLogger()),
	}

	consumer.handleWorkerNotifications()

	return consumer
}

// Consumer id
func (consumer *Consumer) Id() string {
	return consumer.internal.ConsumerId
}

// Associated Consumer id.
func (consumer *Consumer) ConsumerId() string {
	return consumer.internal.ConsumerId
}

// Whether the Consumer is closed.
func (consumer *Consumer) Closed() bool {
	return consumer.closed
}

// Media kind.
func (consumer *Consumer) Kind() string {
	return consumer.data.Kind
}

// RTP parameters.
func (consumer *Consumer) RtpParameters() RtpConsumerCapabilities {
	return consumer.data.RtpParameters
}

// Consumer type.
// It can be "simple", "simulcast" or "svc".
func (consumer *Consumer) Type() string {
	return consumer.data.Type
}

// Whether the Consumer is paused.
func (consumer *Consumer) Paused() bool {
	return consumer.paused
}

// Whether the associate Producer is paused.
func (consumer *Consumer) ProducerPaused() bool {
	return consumer.producerPaused
}

// Consumer score with consumer and consumer keys.
func (consumer *Consumer) Score() ConsumerScore {
	return consumer.score
}

// Current video layers.
func (consumer *Consumer) CurrentLayers() *VideoLayer {
	return consumer.currentLayers
}

// App custom data.
func (consumer *Consumer) AppData() interface{} {
	return consumer.appData
}

/**
 * Observer.
 *
 * @emits close
 * @emits pause
 * @emits resume
 * @emits {consumer: Number, consumer: Number} score
 * @emits {spatialLayer: Number|Null} layerschange
 */
func (consumer *Consumer) Observer() EventEmitter {
	return consumer.observer
}

// Close the Consumer.
func (consumer *Consumer) Close() (err error) {
	if consumer.closed {
		return
	}

	consumer.closed = true

	consumer.logger.Debug("close()")

	consumer.RemoveAllListeners(consumer.internal.ConsumerId)

	response := consumer.channel.Request("consumer.close", consumer.internal, nil)

	if err = response.Err(); err != nil {
		return
	}

	consumer.Emit("@close")

	// Emit observer event.
	consumer.observer.SafeEmit("close")

	return
}

// Transport was closed.
func (consumer *Consumer) TransportClosed() {
	if consumer.closed {
		return
	}

	consumer.closed = true

	consumer.logger.Debug("transportClosed()")

	consumer.SafeEmit("transportclose")

	// Emit observer event.
	consumer.observer.SafeEmit("close")
}

// Dump Consumer.
func (consumer *Consumer) Dump() Response {
	consumer.logger.Debug("dump()")

	return consumer.channel.Request("consumer.dump", consumer.internal, nil)
}

// Get Consumer stats.
func (consumer *Consumer) GetStats() Response {
	consumer.logger.Debug("getStats()")

	return consumer.channel.Request("consumer.getStats", consumer.internal, nil)
}

// Pause the Consumer.
func (consumer *Consumer) Pause() (err error) {
	consumer.logger.Debug("pause()")

	wasPaused := consumer.paused || consumer.producerPaused

	response := consumer.channel.Request("consumer.pause", consumer.internal, nil)

	if err = response.Err(); err != nil {
		return
	}

	consumer.paused = true

	// Emit observer event.
	if !wasPaused {
		consumer.observer.SafeEmit("pause")
	}

	return
}

// Resume the Consumer.
func (consumer *Consumer) Resume() (err error) {
	consumer.logger.Debug("resume()")

	wasPaused := consumer.paused || consumer.producerPaused

	response := consumer.channel.Request("consumer.resume", consumer.internal, nil)

	if err = response.Err(); err != nil {
		return
	}

	consumer.paused = false

	// Emit observer event.
	if wasPaused && !consumer.producerPaused {
		consumer.observer.SafeEmit("resume")
	}

	return
}

// Set preferred video layers.
func (consumer *Consumer) SetPreferredLayers(spatialLayer, temporalLayer uint8) (err error) {
	consumer.logger.Debug("setPreferredLayers()")

	response := consumer.channel.Request(
		"consumer.setPreferredLayers",
		consumer.internal,
		map[string]uint8{
			"spatialLayer":  spatialLayer,
			"temporalLayer": temporalLayer,
		},
	)

	return response.Err()
}

// Request a key frame to the Producer.
func (consumer *Consumer) RequestKeyFrame() error {
	consumer.logger.Debug("requestKeyFrame()")

	response := consumer.channel.Request("consumer.requestKeyFrame", consumer.internal, nil)

	return response.Err()
}

func (consumer *Consumer) handleWorkerNotifications() {
	consumer.On(consumer.internal.ConsumerId, func(event string, data json.RawMessage) {
		switch event {
		case "producerclose":
			if consumer.closed {
				break
			}

			consumer.closed = true

			consumer.channel.RemoveAllListeners(consumer.internal.ConsumerId)

			consumer.Emit("@producerclose")
			consumer.SafeEmit("producerclose")

			// Emit observer event.
			consumer.observer.SafeEmit("close")

		case "producerpause":
			if consumer.producerPaused {
				break
			}

			wasPaused := consumer.paused || consumer.producerPaused

			consumer.producerPaused = true

			consumer.SafeEmit("producerpause")

			// Emit observer event.
			if !wasPaused {
				consumer.observer.SafeEmit("pause")
			}

		case "producerresume":
			if !consumer.producerPaused {
				break
			}

			wasPaused := consumer.paused || consumer.producerPaused

			consumer.producerPaused = false

			consumer.SafeEmit("producerresume")

			// Emit observer event.
			if !wasPaused && !consumer.paused {
				consumer.observer.SafeEmit("resume")
			}

		case "score":
			var score ConsumerScore

			json.Unmarshal([]byte(data), &score)

			consumer.score = score

			consumer.SafeEmit("score", score)

			// Emit observer event.
			consumer.observer.SafeEmit("score", score)

		case "layerschange":
			var layer VideoLayer

			json.Unmarshal([]byte(data), &layer)

			consumer.currentLayers = &layer

			consumer.SafeEmit("layerschange", layer)

			// Emit observer event.
			consumer.observer.SafeEmit("layerschange", layer)

		default:
			consumer.logger.Error(`ignoring unknown event "%s"`, event)
		}
	})
}
