package mediasoup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActiveSpeakerObserver(t *testing.T) {
	mediaCodecs := []*RtpCodecCapability{
		{
			Kind:      "audio",
			MimeType:  "audio/opus",
			ClockRate: 48000,
			Channels:  2,
			Parameters: RtpCodecSpecificParameters{
				Useinbandfec: 1,
			},
		},
	}

	// before all
	worker := CreateTestWorker()
	router, _ := worker.CreateRouter(RouterOptions{
		MediaCodecs: mediaCodecs,
	})
	var activeSpeakerObserver IRtpObserver
	// after all
	defer worker.Close()

	t.Run("router.createActiveSpeakerObserver() succeeds", func(t *testing.T) {
		onObserverNewRtpObserver := NewMockFunc(t)
		router.observer.Once("newrtpobserver", onObserverNewRtpObserver.Fn())

		var err error
		activeSpeakerObserver, err = router.CreateActiveSpeakerObserver()
		require.NoError(t, err)

		onObserverNewRtpObserver.ExpectCalled()
		onObserverNewRtpObserver.ExpectCalledWith(activeSpeakerObserver)
		assert.False(t, activeSpeakerObserver.Closed())
		assert.False(t, activeSpeakerObserver.Paused())
	})

	t.Run("activeSpeakerObserver.pause() and resume() succeed", func(t *testing.T) {
		activeSpeakerObserver.Pause()
		assert.True(t, activeSpeakerObserver.Paused())
		activeSpeakerObserver.Resume()
		assert.False(t, activeSpeakerObserver.Paused())
	})

	t.Run("activeSpeakerObserver.close() succeeds", func(t *testing.T) {
		// We need different a ActiveSpeakerObserver instance here.
		activeSpeakerObserver2, _ := router.CreateActiveSpeakerObserver()

		dump, _ := router.Dump()
		assert.Len(t, dump.RtpObserverIds, 2)

		activeSpeakerObserver2.Close()
		assert.True(t, activeSpeakerObserver2.Closed())

		dump, _ = router.Dump()
		assert.Len(t, dump.RtpObserverIds, 1)
	})

	t.Run(`ActiveSpeakerObserver emits "routerclose" if Router is closed`, func(t *testing.T) {
		// We need different Router and ActiveSpeakerObserver instances here.
		router2, _ := worker.CreateRouter(RouterOptions{MediaCodecs: mediaCodecs})
		activeSpeakerObserver2, _ := router2.CreateActiveSpeakerObserver()
		onObserverNewRtpObserver := NewMockFunc(t)
		activeSpeakerObserver2.On("routerclose", onObserverNewRtpObserver.Fn())
		router2.Close()
		assert.Equal(t, 1, onObserverNewRtpObserver.CalledTimes())
		assert.True(t, activeSpeakerObserver2.Closed())
	})

	t.Run(`ActiveSpeakerObserver emits "routerclose" if Worker is closed`, func(t *testing.T) {
		onObserverNewRtpObserver := NewMockFunc(t)
		activeSpeakerObserver.On("routerclose", onObserverNewRtpObserver.Fn())
		worker.Close()

		assert.Equal(t, 1, onObserverNewRtpObserver.CalledTimes())
		assert.True(t, activeSpeakerObserver.Closed())
	})
}
