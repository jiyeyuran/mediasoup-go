package mediasoup

import "encoding/json"

type AudioLevelObserverOptions struct {
	/**
	 * Maximum int of entries in the 'volumes”' event. Default 1.
	 */
	MaxEntries int `json:"maxEntries"`

	/**
	 * Minimum average volume (in dBvo from -127 to 0) for entries in the
	 * 'volumes' event.	Default -80.
	 */
	Threshold int `json:"threshold"`

	/**
	 * Interval in ms for checking audio volumes. Default 1000.
	 */
	Interval int `json:"interval"`

	/**
	 * Custom application data.
	 */
	AppData interface{} `json:"appData,omitempty"`
}

func NewAudioLevelObserverOptions() AudioLevelObserverOptions {
	return AudioLevelObserverOptions{
		MaxEntries: 1,
		Threshold:  -80,
		Interval:   1000,
		AppData:    H{},
	}
}

type AudioLevelObserverVolume struct {
	/**
	 * The audio producer instance.
	 */
	Producer *Producer

	/**
	 * The average volume (in dBvo from -127 to 0) of the audio producer in the
	 * last interval.
	 */
	Volume int
}

type AudioLevelObserver struct {
	IRtpObserver
	logger Logger
}

/**
 * @emits volumes - (volumes: AudioLevelObserverVolume[])
 * @emits silence
 */
func newAudioLevelObserver(params rtpObserverParams) *AudioLevelObserver {
	o := &AudioLevelObserver{
		IRtpObserver: newRtpObserver(params),
		logger:       NewLogger("AudioLevelObserver"),
	}

	o.handleWorkerNotifications(params)

	return o
}

/**
 * Observer.
 *
 * @emits close
 * @emits pause
 * @emits resume
 * @emits addproducer - (producer: Producer)
 * @emits removeproducer - (producer: Producer)
 * @emits volumes - (volumes: AudioLevelObserverVolume[])
 * @emits silence
 */
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
				o.logger.Error(`unmarshal events failed: %s`, err)
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
			o.logger.Error(`ignoring unknown event "%s"`, event)
		}
	})
}
