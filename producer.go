package mediasoup

import (
	"encoding/json"
	"sync"
	"sync/atomic"

	"github.com/go-logr/logr"
)

// ProducerOptions define options to create a producer.
type ProducerOptions struct {
	// Id is the producer id (just for Router.pipeToRouter() method).
	Id string `json:"id,omitempty"`

	// Kind is media kind ("audio" or "video").
	Kind MediaKind `json:"kind,omitempty"`

	// RtpParameters define what the endpoint is sending.
	RtpParameters RtpParameters `json:"rtpParameters,omitempty"`

	// Paused define whether the producer must start in paused mode. Default false.
	Paused bool `json:"paused,omitempty"`

	// KeyFrameRequestDelay is just used for video. Time (in ms) before asking
	// the sender for a new key frame after having asked a previous one. Default 0.
	KeyFrameRequestDelay uint32 `json:"keyFrameRequestDelay,omitempty"`

	// AppData is custom application data.
	AppData interface{} `json:"appData,omitempty"`
}

// ProducerTraceEventType define the type for "trace" event.
type ProducerTraceEventType string

const (
	ProducerTraceEventType_Rtp      ProducerTraceEventType = "rtp"
	ProducerTraceEventType_Keyframe ProducerTraceEventType = "keyframe"
	ProducerTraceEventType_Nack     ProducerTraceEventType = "nack"
	ProducerTraceEventType_Pli      ProducerTraceEventType = "pli"
	ProducerTraceEventType_Fir      ProducerTraceEventType = "fir"
)

// ProducerTraceEventData define "trace" event data.
type ProducerTraceEventData struct {
	// Type is the trace type.
	Type ProducerTraceEventType `json:"type,omitempty"`

	// Timestamp is event timestamp.
	Timestamp uint32 `json:"timestamp,omitempty"`

	// Direction is event direction, "in" | "out".
	Direction string `json:"direction,omitempty"`

	// Info is per type information.
	Info H `json:"info,omitempty"`
}

// ProducerScore define "score" event data
type ProducerScore struct {
	// Ssrc of the RTP stream.
	Ssrc uint32 `json:"ssrc,omitempty"`

	// Rid of the RTP stream.
	Rid string `json:"rid,omitempty"`

	// Score of the RTP stream.
	Score uint32 `json:"score"`
}

// ProducerVideoOrientation define "videoorientationchange" event data
type ProducerVideoOrientation struct {
	// Camera define whether the source is a video camera.
	Camera bool `json:"Camera,omitempty"`

	// Flip define whether the video source is flipped.
	Flip bool `json:"flip,omitempty"`

	// Rotation degrees (0, 90, 180 or 270).
	Rotation uint32 `json:"rotation"`
}

// ProducerStat define the statistic info of a producer
type ProducerStat struct {
	ConsumerStat
	// Jitter is the jitter buffer.
	Jitter uint32 `json:"jitter,omitempty"`
	// BitrateByLayer is a map of bitrate of each layer (such as {"0.0": 100, "1.0": 500})
	BitrateByLayer map[string]uint32 `json:"bitrateByLayer,omitempty"`
}

// ProducerType define Producer type.
type ProducerType string

const (
	ProducerType_Simple    ProducerType = "simple"
	ProducerType_Simulcast ProducerType = "simulcast"
	ProducerType_Svc       ProducerType = "svc"
)

type producerData struct {
	Kind                    MediaKind     `json:"kind,omitempty"`
	Type                    ProducerType  `json:"type,omitempty"`
	RtpParameters           RtpParameters `json:"rtpParameters,omitempty"`
	ConsumableRtpParameters RtpParameters `json:"consumableRtpParameters,omitempty"`
}

type producerParams struct {
	// internal uses with routerId, transportId, producerId
	internal       internalData
	data           producerData
	channel        *Channel
	payloadChannel *PayloadChannel
	appData        interface{}
	paused         bool
}

// Producer represents an audio or video source being injected into a mediasoup router.
// It's created on top of a transport that defines how the media packets are carried.
//
// - @emits transportclose
// - @emits score - (scores []ProducerScore)
// - @emits videoorientationchange - (videoOrientation *ProducerVideoOrientation)
// - @emits trace - (trace *ProducerTraceEventData)
// - @emits @close
type Producer struct {
	IEventEmitter
	locker                   sync.Mutex
	logger                   logr.Logger
	internal                 internalData
	data                     producerData
	channel                  *Channel
	payloadChannel           *PayloadChannel
	appData                  interface{}
	paused                   bool
	closed                   uint32
	score                    []ProducerScore
	observer                 IEventEmitter
	onClose                  func()
	onTransportClose         func()
	onPause                  func()
	onResume                 func()
	onScore                  func([]ProducerScore)
	onVideoOrientationChange func(*ProducerVideoOrientation)
	onTrace                  func(*ProducerTraceEventData)
}

func newProducer(params producerParams) *Producer {
	logger := NewLogger("Producer")

	logger.V(1).Info("constructor()", "internal", params.internal)

	producer := &Producer{
		IEventEmitter:  NewEventEmitter(),
		logger:         logger,
		internal:       params.internal,
		data:           params.data,
		channel:        params.channel,
		payloadChannel: params.payloadChannel,
		appData:        params.appData,
		paused:         params.paused,
		observer:       NewEventEmitter(),
	}

	producer.handleWorkerNotifications()

	return producer
}

// Id returns producer id
func (producer *Producer) Id() string {
	return producer.internal.ProducerId
}

// Closed returns whether the Producer is closed.
func (producer *Producer) Closed() bool {
	return atomic.LoadUint32(&producer.closed) > 0
}

// Kind returns media kind.
func (producer *Producer) Kind() MediaKind {
	return producer.data.Kind
}

// RtpParameters returns RTP parameters.
func (producer *Producer) RtpParameters() RtpParameters {
	return producer.data.RtpParameters
}

// Type returns producer type.
func (producer *Producer) Type() ProducerType {
	return producer.data.Type
}

// ConsumableRtpParameters returns consumable RTP parameters.
func (producer *Producer) ConsumableRtpParameters() RtpParameters {
	return producer.data.ConsumableRtpParameters
}

// Paused returns whether the Producer is paused.
func (producer *Producer) Paused() bool {
	producer.locker.Lock()
	defer producer.locker.Unlock()

	return producer.paused
}

// Score returns producer score list.
func (producer *Producer) Score() []ProducerScore {
	return producer.score
}

// AppData returns app custom data.
func (producer *Producer) AppData() interface{} {
	return producer.appData
}

// Observer.
//
// - @emits close
// - @emits pause
// - @emits resume
// - @emits score - (scores []ProducerScore)
// - @emits videoorientationchange - (videoOrientation *ProducerVideoOrientation)
// - @emits trace - (trace *ProducerTraceEventData)
func (producer *Producer) Observer() IEventEmitter {
	return producer.observer
}

// Close the producer.
func (producer *Producer) Close() (err error) {
	if atomic.CompareAndSwapUint32(&producer.closed, 0, 1) {
		producer.logger.V(1).Info("close()")

		// Remove notification subscriptions.
		producer.channel.Unsubscribe(producer.Id())
		producer.payloadChannel.Unsubscribe(producer.Id())

		reqData := H{"producerId": producer.internal.ProducerId}
		response := producer.channel.Request("transport.closeProducer", producer.internal, reqData)

		if err = response.Err(); err != nil {
			producer.logger.Error(err, "producer close error failed")
		}

		producer.Emit("@close")
		producer.RemoveAllListeners()

		producer.close()
	}

	return
}

// close send "close" event.
func (producer *Producer) close() {
	// Emit observer event.
	producer.observer.SafeEmit("close")
	producer.observer.RemoveAllListeners()

	if handler := producer.onClose; handler != nil {
		handler()
	}
}

// transportClosed is called when transport was closed.
func (producer *Producer) transportClosed() {
	if atomic.CompareAndSwapUint32(&producer.closed, 0, 1) {
		producer.logger.V(1).Info("transportClosed()")

		// Remove notification subscriptions.
		producer.channel.Unsubscribe(producer.Id())
		producer.payloadChannel.Unsubscribe(producer.Id())

		producer.SafeEmit("transportclose")
		producer.RemoveAllListeners()

		if handler := producer.onTransportClose; handler != nil {
			handler()
		}

		producer.close()
	}
}

// Dump producer.
func (producer *Producer) Dump() (dump ProducerDump, err error) {
	producer.logger.V(1).Info("dump()")

	resp := producer.channel.Request("producer.dump", producer.internal)
	err = resp.Unmarshal(&dump)

	return
}

// GetStats returns producer stats.
func (producer *Producer) GetStats() (stats []*ProducerStat, err error) {
	producer.logger.V(1).Info("getStats()")

	resp := producer.channel.Request("producer.getStats", producer.internal)
	err = resp.Unmarshal(&stats)

	return
}

// Pause the producer.
func (producer *Producer) Pause() (err error) {
	producer.locker.Lock()
	defer producer.locker.Unlock()

	producer.logger.V(1).Info("pause()")

	wasPaused := producer.paused

	response := producer.channel.Request("producer.pause", producer.internal)

	if err = response.Err(); err != nil {
		return
	}

	producer.paused = true

	// Emit observer event.
	if !wasPaused {
		producer.observer.SafeEmit("pause")

		if handler := producer.onPause; handler != nil {
			handler()
		}
	}

	return
}

// Resume the producer.
func (producer *Producer) Resume() (err error) {
	producer.locker.Lock()
	defer producer.locker.Unlock()

	producer.logger.V(1).Info("resume()")

	wasPaused := producer.paused

	result := producer.channel.Request("producer.resume", producer.internal)

	if err = result.Err(); err != nil {
		return
	}

	producer.paused = false

	// Emit observer event.
	if wasPaused {
		producer.observer.SafeEmit("resume")

		if handler := producer.onResume; handler != nil {
			handler()
		}
	}

	return
}

// EnableTraceEvent enable "trace" event.
func (producer *Producer) EnableTraceEvent(types ...ProducerTraceEventType) error {
	producer.logger.V(1).Info("enableTraceEvent()")

	if types == nil {
		types = []ProducerTraceEventType{}
	}

	result := producer.channel.Request("producer.enableTraceEvent", producer.internal, H{"types": types})

	return result.Err()
}

// Send RTP packet (just valid for Producers created on a DirectTransport).
func (producer *Producer) Send(rtpPacket []byte) error {
	return producer.payloadChannel.Notify("producer.send", producer.internal, "", rtpPacket)
}

// OnClose set handler on "close" event
func (producer *Producer) OnClose(handler func()) {
	producer.onClose = handler
}

// OnTransportClose set handler on "transportclose" event
func (producer *Producer) OnTransportClose(handler func()) {
	producer.onTransportClose = handler
}

// OnPause set handler on "pause" event
func (producer *Producer) OnPause(handler func()) {
	producer.onPause = handler
}

// OnResume set handler on "resume" event
func (producer *Producer) OnResume(handler func()) {
	producer.onResume = handler
}

// OnScore set handler on "score" event
func (producer *Producer) OnScore(handler func(score []ProducerScore)) {
	producer.onScore = handler
}

// OnVideoOrientationChange set handler on "videoorientationchange" event
func (producer *Producer) OnVideoOrientationChange(handler func(videoOrientation *ProducerVideoOrientation)) {
	producer.onVideoOrientationChange = handler
}

// OnTrace set handler on "trace" event
func (producer *Producer) OnTrace(handler func(trace *ProducerTraceEventData)) {
	producer.onTrace = handler
}

func (producer *Producer) handleWorkerNotifications() {
	logger := producer.logger

	producer.channel.Subscribe(producer.Id(), func(event string, data []byte) {
		switch event {
		case "score":
			score := []ProducerScore{}

			if err := json.Unmarshal([]byte(data), &score); err != nil {
				logger.Error(err, "failed to unmarshal score", "data", json.RawMessage(data))
				return
			}

			producer.score = score

			producer.SafeEmit("score", score)

			// Emit observer event.
			producer.observer.SafeEmit("score", score)

			if handler := producer.onScore; handler != nil {
				handler(score)
			}

		case "videoorientationchange":
			orientation := &ProducerVideoOrientation{}

			if err := json.Unmarshal([]byte(data), &orientation); err != nil {
				logger.Error(err, "failed to unmarshal orientation", "data", json.RawMessage(data))
				return
			}

			producer.SafeEmit("videoorientationchange", orientation)

			// Emit observer event.
			producer.observer.SafeEmit("videoorientationchange", orientation)

			if handler := producer.onVideoOrientationChange; handler != nil {
				handler(orientation)
			}

		case "trace":
			var trace *ProducerTraceEventData

			if err := json.Unmarshal([]byte(data), &trace); err != nil {
				logger.Error(err, "failed to unmarshal trace", "data", json.RawMessage(data))
				return
			}

			producer.SafeEmit("trace", trace)

			// Emit observer event.
			producer.observer.SafeEmit("trace", trace)

			if handler := producer.onTrace; handler != nil {
				handler(trace)
			}

		default:
			logger.Error(nil, "ignoring unknown event", "event", event)
		}
	})
}
