package mediasoup

import (
	"context"
	"testing"
	"time"

	FbsActiveSpeakerObserver "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/ActiveSpeakerObserver"
	FbsAudioLevelObserver "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/AudioLevelObserver"
	FbsNotification "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
		mymock := new(MockedHandler)
		defer mymock.AssertExpectations(t)

		mymock.On("OnClose", mock.IsType(context.Background())).Once()

		worker := newTestWorker()
		router := createRouter(worker)

		o, _ := router.CreateActiveSpeakerObserver()
		o.OnClose(mymock.OnClose)

		err := o.Close()
		assert.NoError(t, err)
		assert.True(t, o.Closed())
	})

	t.Run("router closed", func(t *testing.T) {
		mymock := new(MockedHandler)
		defer mymock.AssertExpectations(t)

		mymock.On("OnClose", mock.IsType(context.Background())).Once()

		worker := newTestWorker()
		router := createRouter(worker)

		o, _ := router.CreateActiveSpeakerObserver()
		o.OnClose(mymock.OnClose)

		router.Close()
		assert.True(t, o.Closed())
	})

	t.Run("worker closed", func(t *testing.T) {
		mymock := new(MockedHandler)
		defer mymock.AssertExpectations(t)

		mymock.On("OnClose", mock.IsType(context.Background())).Once()

		worker := newTestWorker()
		router := createRouter(worker)

		o, _ := router.CreateActiveSpeakerObserver()
		o.OnClose(mymock.OnClose)

		worker.Close()

		assert.True(t, o.Closed())
	})
}

func TestRtpObserverNotification(t *testing.T) {
	mymock := new(MockedHandler)
	defer mymock.AssertExpectations(t)

	router := createRouter(nil)
	producer := createAudioProducer(createWebRtcTransport(router))

	speaker := AudioLevelObserverDominantSpeaker{Producer: producer}
	volumes := []AudioLevelObserverVolume{{Producer: producer, Volume: -10}}

	mymock.On("OnDominantSpeaker", speaker)
	mymock.On("OnSilence")
	mymock.On("OnVolume", volumes)

	o, _ := router.CreateActiveSpeakerObserver()
	o.OnDominantSpeaker(mymock.OnDominantSpeaker)
	o.OnSilence(mymock.OnSilence)
	o.OnVolume(mymock.OnVolume)

	o.channel.ProcessNotificationForTesting(&FbsNotification.NotificationT{
		HandlerId: o.Id(),
		Event:     FbsNotification.EventACTIVESPEAKEROBSERVER_DOMINANT_SPEAKER,
		Body: &FbsNotification.BodyT{
			Type: FbsNotification.BodyActiveSpeakerObserver_DominantSpeakerNotification,
			Value: &FbsActiveSpeakerObserver.DominantSpeakerNotificationT{
				ProducerId: speaker.Producer.Id(),
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
					{ProducerId: volumes[0].Producer.Id(), Volume: volumes[0].Volume},
				},
			},
		},
	})
	time.Sleep(time.Millisecond)
}
