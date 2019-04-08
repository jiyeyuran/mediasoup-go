package mediasoup

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
)

type AudioLevelObserver struct {
	*baseRtpObserver
	logger logrus.FieldLogger
}

func NewAudioLevelObserver(
	internal Internal,
	channel *Channel,
	getProducerById FetchProducerFunc,
) *AudioLevelObserver {
	o := &AudioLevelObserver{
		baseRtpObserver: newRtpObserver(internal, channel),
		logger:          TypeLogger("AudioLevelObserver"),
	}

	o.handleWorkerNotifications(internal.RtpObserverId, getProducerById)

	return o
}

func (o *AudioLevelObserver) handleWorkerNotifications(
	rtpObserverId string,
	getProducerById FetchProducerFunc,
) {
	o.baseRtpObserver.channel.On(rtpObserverId,
		func(event string, data json.RawMessage) {
			switch event {
			case "volumes":
				// Get the corresponding Producer instance and remove entries with
				// no Producer (it may have been closed in the meanwhile).
				var volumes []VolumeInfo
				var notifications []struct {
					ProducerId string
					Volume     uint8
				}

				json.Unmarshal([]byte(data), &notifications)

				for _, notification := range notifications {
					producer := getProducerById(notification.ProducerId)

					if producer != nil {
						volumes = append(volumes, VolumeInfo{
							Producer: producer,
							Volume:   notification.Volume,
						})
					}
				}

				if len(volumes) > 0 {
					o.SafeEmit("volumes", volumes)
				}
			case "silence":
				o.SafeEmit("silence")
			default:
				o.logger.Errorf(`ignoring unknown event "%s"`, event)
			}
		},
	)
}
