package mediasoup

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
)

type AudioLevelObserver struct {
	*RtpObserver
	logger logrus.FieldLogger
}

func NewAudioLevelObserver(
	internal Internal,
	channel *Channel,
	getProducerById FetchProducerFunc,
) *AudioLevelObserver {
	return &AudioLevelObserver{
		RtpObserver: NewRtpObserver(internal, channel, getProducerById),
		logger:      TypeLogger("AudioLevelObserver"),
	}
}

func (audioLevelObserver AudioLevelObserver) handleWorkerNotifications() {
	audioLevelObserver.channel.On(
		audioLevelObserver.internal.RtpObserverId,
		func(event string, data json.RawMessage) {
			switch event {
			case "volumes":
				// Get the corresponding Producer instance and remove entries with
				// no Producer (it may have been closed in the meanwhile).
				var volumes []VolumeInfo
				var notifications []struct {
					ProducerId string
					Volume     float64
				}

				json.Unmarshal([]byte(data), &notifications)

				for _, notification := range notifications {
					producer := audioLevelObserver.getProducerById(notification.ProducerId)
					if producer != nil {
						volumes = append(volumes, VolumeInfo{
							Producer: producer,
							Volume:   notification.Volume,
						})
					}
				}

				if len(volumes) > 0 {
					audioLevelObserver.SafeEmit("volumes", volumes)
				}
			case "silence":
				audioLevelObserver.SafeEmit("silence")
			default:
				audioLevelObserver.logger.Error(`ignoring unknown event "%s"`, event)
			}
		},
	)
}
