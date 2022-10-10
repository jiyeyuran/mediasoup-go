package mediasoup

import (
	"encoding/json"

	"github.com/go-logr/logr"
)

type ActiveSpeakerObserverOptions struct {
	Interval int         `json:"interval"`
	AppData  interface{} `json:"appData,omitempty"`
}

type ActiveSpeakerObserverActivity struct {
	// Producer is the dominant audio producer instance.
	Producer *Producer
}

// ActiveSpeakerObserver monitors the speech activity of the selected audio producers. It just
// handles audio producers (if addProducer() is called with a video producer it will fail).
//
// Implementation of Dominant Speaker Identification for Multipoint Videoconferencing by Ilana
// Volfin and Israel Cohen. This implementation uses the RTP Audio Level extension from RFC-6464
// for the input signal. This has been ported from DominantSpeakerIdentification.java in Jitsi.
// Audio levels used for speech detection are read from an RTP header extension. No decoding of
// audio data is done. See RFC6464 for more information.
//
// - @emits dominantspeaker - (activity *ActiveSpeakerObserverActivity)
type ActiveSpeakerObserver struct {
	IRtpObserver
	logger            logr.Logger
	onDominantSpeaker func(speaker *ActiveSpeakerObserverActivity)
}

func newActiveSpeakerObserver(params rtpObserverParams) *ActiveSpeakerObserver {
	o := &ActiveSpeakerObserver{
		IRtpObserver: newRtpObserver(params),
		logger:       NewLogger("ActiveSpeakerObserver"),
	}
	o.handleWorkerNotifications(params)

	return o
}

// Observer.
//
// - @emits dominantspeaker - (activity *ActiveSpeakerObserverActivity)
func (o *ActiveSpeakerObserver) Observer() IEventEmitter {
	return o.IRtpObserver.Observer()
}

// OnDominantSpeaker set handler on "dominantspeaker" event
func (o *ActiveSpeakerObserver) OnDominantSpeaker(handler func(speaker *ActiveSpeakerObserverActivity)) {
	o.onDominantSpeaker = handler
}

func (o *ActiveSpeakerObserver) handleWorkerNotifications(params rtpObserverParams) {
	rtpObserverId := params.internal.RtpObserverId
	getProducerById := params.getProducerById

	type eventInfo struct {
		ProducerId string `json:"producerId,omitempty"`
	}

	params.channel.Subscribe(rtpObserverId, func(event string, data []byte) {
		switch event {
		case "dominantspeaker":
			event := eventInfo{}

			if err := json.Unmarshal(data, &event); err != nil {
				o.logger.Error(err, "unmarshal dominantspeaker failed", "data", json.RawMessage(data))
				break
			}

			dominantSpeaker := &ActiveSpeakerObserverActivity{
				Producer: getProducerById(event.ProducerId),
			}
			o.SafeEmit("dominantspeaker", dominantSpeaker)

			// Emit observer event.
			o.Observer().SafeEmit("dominantspeaker", dominantSpeaker)

			if handler := o.onDominantSpeaker; handler != nil {
				handler(dominantSpeaker)
			}

		default:
			o.logger.Error(nil, "ignoring unknown event", "event", event)
		}
	})
}
