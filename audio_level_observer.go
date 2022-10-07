package mediasoup

import (
	"encoding/json"

	"github.com/go-logr/logr"
)

// AudioLevelObserverOptions define options to create an AudioLevelObserver.
type AudioLevelObserverOptions struct {
	// MaxEntries is maximum int of entries in the 'volumes”' event. Default 1.
	MaxEntries int `json:"maxEntries"`

	// Threshold is minimum average volume (in dBvo from -127 to 0) for entries in the
	// "volumes" event.	Default -80.
	Threshold int `json:"threshold"`

	// Interval in ms for checking audio volumes. Default 1000.
	Interval int `json:"interval"`

	// AppData is custom application data.
	AppData interface{} `json:"appData,omitempty"`
}

type AudioLevelObserverVolume struct {
	// Producer is the audio producer instance.
	Producer *Producer

	// Volume is the average volume (in dBvo from -127 to 0) of the audio producer in the
	// last interval.
	Volume int
}

// AudioLevelObserver monitors the volume of the selected audio producers. It just handles audio
// producers (if AddProducer() is called with a video producer it will fail).
//
// Audio levels are read from an RTP header extension. No decoding of audio data is done. See
// RFC6464 for more information.
//
// - @emits volumes - (volumes []AudioLevelObserverVolume)
// - @emits silence
type AudioLevelObserver struct {
	IRtpObserver
	logger logr.Logger
}

func newAudioLevelObserver(params rtpObserverParams) *AudioLevelObserver {
	o := &AudioLevelObserver{
		IRtpObserver: newRtpObserver(params),
		logger:       NewLogger("AudioLevelObserver"),
	}

	o.handleWorkerNotifications(params)

	return o
}

// Observer.
//
// - @emits close
// - @emits pause
// - @emits resume
// - @emits addproducer - (producer *Producer)
// - @emits removeproducer - (producer *Producer)
// - @emits volumes - (volumes []AudioLevelObserverVolume)
// - @emits silence
func (o *AudioLevelObserver) Observer() IEventEmitter {
	return o.IRtpObserver.Observer()
}

func (o *AudioLevelObserver) handleWorkerNotifications(params rtpObserverParams) {
	rtpObserverId := params.internal.RtpObserverId
	getProducerById := params.getProducerById

	type eventInfo struct {
		ProducerId string `json:"producerId,omitempty"`
		Volume     int    `json:"volume,omitempty"`
	}

	params.channel.On(rtpObserverId, func(event string, data []byte) {
		switch event {
		case "volumes":
			// Get the corresponding Producer instance and remove entries with
			// no Producer (it may have been closed in the meanwhile).
			var volumes []AudioLevelObserverVolume

			events := []eventInfo{}

			if err := json.Unmarshal(data, &events); err != nil {
				o.logger.Error(err, "unmarshal events volumes", "data", json.RawMessage(data))
				break
			}

			for _, row := range events {
				if producer := getProducerById(row.ProducerId); producer != nil {
					volumes = append(volumes, AudioLevelObserverVolume{
						Producer: producer,
						Volume:   row.Volume,
					})
				}
			}

			if len(volumes) > 0 {
				o.SafeEmit("volumes", volumes)

				// Emit observer event.
				o.Observer().SafeEmit("volumes", volumes)
			}
		case "silence":
			o.SafeEmit("silence")

			// Emit observer event.
			o.Observer().SafeEmit("silence")
		default:
			o.logger.Error(nil, "ignoring unknown event", "event", event)
		}
	})
}
