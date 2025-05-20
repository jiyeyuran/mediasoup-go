package mediasoup

import (
	"context"
	"fmt"
	"log/slog"

	FbsActiveSpeakerObserver "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/ActiveSpeakerObserver"
	FbsAudioLevelObserver "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/AudioLevelObserver"
	FbsNotification "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Notification"
	FbsRequest "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Request"
	FbsRouter "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Router"
	FbsRtpObserver "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/RtpObserver"
	"github.com/jiyeyuran/mediasoup-go/v2/internal/channel"
)

type rtpObserverData struct {
	Id              string
	Type            RtpObserverType
	RouterId        string
	AppData         H
	GetProducerById func(string) *Producer
}

type RtpObserver struct {
	baseListener

	data                    *rtpObserverData
	sub                     *channel.Subscription
	logger                  *slog.Logger
	channel                 *channel.Channel
	paused                  bool
	closed                  bool
	dominantSpeakerHandlers []func(AudioLevelObserverDominantSpeaker)
	volumeHandlers          []func([]AudioLevelObserverVolume)
	silenceHandlers         []func()
}

func newRtpObserver(channel *channel.Channel, logger *slog.Logger, data *rtpObserverData) *RtpObserver {
	r := &RtpObserver{
		channel: channel,
		logger:  logger.With("rtpObserverId", data.Id, "rtpObserverType", data.Type),
		data:    data,
	}
	r.handleWorkerNotifications()
	return r
}

func (r *RtpObserver) Id() string {
	return r.data.Id
}

func (r *RtpObserver) Type() RtpObserverType {
	return r.data.Type
}

func (r *RtpObserver) AppData() H {
	return r.data.AppData
}

func (r *RtpObserver) Paused() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.paused
}

func (r *RtpObserver) Closed() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.closed
}

func (r *RtpObserver) Close() error {
	return r.CloseContext(context.Background())
}

func (r *RtpObserver) CloseContext(ctx context.Context) error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.logger.DebugContext(ctx, "Close()")

	_, err := r.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodROUTER_CLOSE_RTPOBSERVER,
		HandlerId: r.data.RouterId,
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyRouter_CloseRtpObserverRequest,
			Value: &FbsRouter.CloseRtpObserverRequestT{
				RtpObserverId: r.data.Id,
			},
		},
	})
	if err != nil {
		r.mu.Unlock()
		return err
	}
	r.closed = true
	r.mu.Unlock()

	r.cleanupAfterClosed()
	return nil
}

// Pause the RtpObserver.
func (r *RtpObserver) Pause() error {
	return r.PauseContext(context.Background())
}

func (r *RtpObserver) PauseContext(ctx context.Context) error {
	r.logger.DebugContext(ctx, "Pause()")

	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodRTPOBSERVER_PAUSE,
		HandlerId: r.Id(),
	})
	if err != nil {
		return err
	}
	r.paused = true
	return nil
}

// Resume the RtpObserver.
func (r *RtpObserver) Resume() error {
	return r.ResumeContext(context.Background())
}

func (r *RtpObserver) ResumeContext(ctx context.Context) error {
	r.logger.DebugContext(ctx, "Resume()")

	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodRTPOBSERVER_RESUME,
		HandlerId: r.Id(),
	})
	if err != nil {
		return err
	}
	r.paused = false
	return nil
}

// AddProducer add a Producer to the RtpObserver.
func (r *RtpObserver) AddProducer(producerId string) error {
	return r.AddProducerContext(context.Background(), producerId)
}

func (r *RtpObserver) AddProducerContext(ctx context.Context, producerId string) error {
	r.logger.DebugContext(ctx, "AddProducer()")

	producer := r.data.GetProducerById(producerId)
	if producer == nil {
		return fmt.Errorf("producer with id %q not found", producerId)
	}

	_, err := r.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodRTPOBSERVER_ADD_PRODUCER,
		HandlerId: r.Id(),
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyRtpObserver_AddProducerRequest,
			Value: &FbsRtpObserver.AddProducerRequestT{
				ProducerId: producerId,
			},
		},
	})
	return err
}

// RemoveProducer remove a Producer from the RtpObserver.
func (r *RtpObserver) RemoveProducer(producerId string) error {
	return r.RemoveProducerContext(context.Background(), producerId)
}

func (r *RtpObserver) RemoveProducerContext(ctx context.Context, producerId string) error {
	r.logger.DebugContext(ctx, "RemoveProducer()")

	producer := r.data.GetProducerById(producerId)
	if producer == nil {
		return fmt.Errorf("producer with id %q not found", producerId)
	}

	_, err := r.channel.Request(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodRTPOBSERVER_REMOVE_PRODUCER,
		HandlerId: r.Id(),
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyRtpObserver_RemoveProducerRequest,
			Value: &FbsRtpObserver.RemoveProducerRequestT{
				ProducerId: producerId,
			},
		},
	})
	return err
}

// HandleAudioLevelObserverDominantSpeaker add handler on "dominantspeaker" event
func (r *RtpObserver) OnDominantSpeaker(handler func(AudioLevelObserverDominantSpeaker)) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.dominantSpeakerHandlers = append(r.dominantSpeakerHandlers, handler)
}

// HandleVolume add handler on "volumes" event
func (r *RtpObserver) OnVolume(handler func([]AudioLevelObserverVolume)) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.volumeHandlers = append(r.volumeHandlers, handler)
}

// HandleSilence add handler on "silence" event
func (r *RtpObserver) OnSilence(handler func()) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.silenceHandlers = append(r.silenceHandlers, handler)
}

func (r *RtpObserver) handleWorkerNotifications() {
	r.sub = r.channel.Subscribe(r.Id(), func(event FbsNotification.Event, body *FbsNotification.BodyT) {
		switch event {
		case FbsNotification.EventACTIVESPEAKEROBSERVER_DOMINANT_SPEAKER:
			notification := body.Value.(*FbsActiveSpeakerObserver.DominantSpeakerNotificationT)

			r.mu.RLock()
			handlers := r.dominantSpeakerHandlers
			r.mu.RUnlock()

			producer := r.data.GetProducerById(notification.ProducerId)
			if producer == nil {
				break
			}

			for _, handler := range handlers {
				handler(AudioLevelObserverDominantSpeaker{
					Producer: producer,
				})
			}

		case FbsNotification.EventAUDIOLEVELOBSERVER_VOLUMES:
			notification := body.Value.(*FbsAudioLevelObserver.VolumesNotificationT)

			r.mu.RLock()
			handlers := r.volumeHandlers
			r.mu.RUnlock()

			volumes := make([]AudioLevelObserverVolume, 0, len(notification.Volumes))

			for _, volume := range notification.Volumes {
				producer := r.data.GetProducerById(volume.ProducerId)
				if producer == nil {
					continue
				}
				volumes = append(volumes, AudioLevelObserverVolume{
					Producer: producer,
					Volume:   volume.Volume,
				})
			}
			for _, handler := range handlers {
				handler(volumes)
			}

		case FbsNotification.EventAUDIOLEVELOBSERVER_SILENCE:
			r.mu.RLock()
			handlers := r.silenceHandlers
			r.mu.RUnlock()
			for _, handler := range handlers {
				handler()
			}

		default:
			r.logger.Warn("ignoring unknown event in RtpObserver", "event", event)
		}
	})
}

func (r *RtpObserver) routerClosed() {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.closed = true
	r.mu.Unlock()
	r.logger.Debug("routerClosed()")

	r.cleanupAfterClosed()
}

func (r *RtpObserver) cleanupAfterClosed() {
	r.sub.Unsubscribe()
	r.notifyClosed()
}
