package mediasoup

import (
	"testing"
	"time"

	FbsActiveSpeakerObserver "github.com/jiyeyuran/mediasoup-go/internal/FBS/ActiveSpeakerObserver"
	FbsAudioLevelObserver "github.com/jiyeyuran/mediasoup-go/internal/FBS/AudioLevelObserver"
	FbsNotification "github.com/jiyeyuran/mediasoup-go/internal/FBS/Notification"
	"github.com/stretchr/testify/assert"
)

func TestRtpObserverPauseAndResume(t *testing.T) {
	router := createRouter(nil)

	o, _ := router.CreateActiveSpeakerObserver()
	err := o.Pause()
	assert.NoError(t, err)
	assert.True(t, o.Paused())
	err = o.Resume()
	assert.NoError(t, err)
	assert.False(t, o.Paused())
}

func TestRtpOberverAddAndRemoveProducer(t *testing.T) {
	router := createRouter(nil)
	producer := createAudioProducer(createWebRtcTransport(router))

	o, _ := router.CreateActiveSpeakerObserver()

	err := o.AddProducer(producer.Id())
	assert.NoError(t, err)

	err = o.RemoveProducer(producer.Id())
	assert.NoError(t, err)
}

func TestRtpObserverClose(t *testing.T) {
	t.Run("close normally", func(t *testing.T) {
		mock := new(MockedHandler)
		defer mock.AssertExpectations(t)

		mock.On("OnClose").Times(1)

		worker := newTestWorker()
		router := createRouter(worker)

		o, _ := router.CreateActiveSpeakerObserver()
		o.OnClose(mock.OnClose)

		err := o.Close()
		assert.NoError(t, err)
		assert.True(t, o.Closed())
	})

	t.Run("router closed", func(t *testing.T) {
		mock := new(MockedHandler)
		defer mock.AssertExpectations(t)

		mock.On("OnClose").Times(1)

		worker := newTestWorker()
		router := createRouter(worker)

		o, _ := router.CreateActiveSpeakerObserver()
		o.OnClose(mock.OnClose)

		router.Close()
		assert.True(t, o.Closed())
	})

	t.Run("worker closed", func(t *testing.T) {
		mock := new(MockedHandler)
		defer mock.AssertExpectations(t)

		mock.On("OnClose").Times(1)

		worker := newTestWorker()
		router := createRouter(worker)

		o, _ := router.CreateActiveSpeakerObserver()
		o.OnClose(mock.OnClose)

		worker.Close()

		assert.True(t, o.Closed())
	})
}

func TestRtpObserverNotification(t *testing.T) {
	mock := new(MockedHandler)
	defer mock.AssertExpectations(t)

	router := createRouter(nil)
	producer := createAudioProducer(createWebRtcTransport(router))

	speaker := AudioLevelObserverDominantSpeaker{ProducerId: producer.Id()}
	volumes := []AudioLevelObserverVolume{{ProducerId: producer.Id(), Volume: -10}}

	mock.On("OnDominantSpeaker", speaker)
	mock.On("OnSilence")
	mock.On("OnVolume", volumes)

	o, _ := router.CreateActiveSpeakerObserver()
	o.OnDominantSpeaker(mock.OnDominantSpeaker)
	o.OnSilence(mock.OnSilence)
	o.OnVolume(mock.OnVolume)

	o.channel.ProcessNotificationForTesting(&FbsNotification.NotificationT{
		HandlerId: o.Id(),
		Event:     FbsNotification.EventACTIVESPEAKEROBSERVER_DOMINANT_SPEAKER,
		Body: &FbsNotification.BodyT{
			Type: FbsNotification.BodyActiveSpeakerObserver_DominantSpeakerNotification,
			Value: &FbsActiveSpeakerObserver.DominantSpeakerNotificationT{
				ProducerId: speaker.ProducerId,
			},
		},
	})
	o.channel.ProcessNotificationForTesting(&FbsNotification.NotificationT{
		HandlerId: o.Id(),
		Event:     FbsNotification.EventAUDIOLEVELOBSERVER_SILENCE,
	})
	o.channel.ProcessNotificationForTesting(&FbsNotification.NotificationT{
		HandlerId: o.Id(),
		Event:     FbsNotification.EventAUDIOLEVELOBSERVER_VOLUMES,
		Body: &FbsNotification.BodyT{
			Type: FbsNotification.BodyAudioLevelObserver_VolumesNotification,
			Value: &FbsAudioLevelObserver.VolumesNotificationT{
				Volumes: []*FbsAudioLevelObserver.VolumeT{
					{ProducerId: volumes[0].ProducerId, Volume: volumes[0].Volume},
				},
			},
		},
	})
	time.Sleep(time.Millisecond)
}
