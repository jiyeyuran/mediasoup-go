package mediasoup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAudioLevelObserver(t *testing.T) {
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
	var AudioLevelObserver IRtpObserver
	// after all
	defer worker.Close()

	t.Run("router.createAudioLevelObserver() succeeds", func(t *testing.T) {
		onObserverNewRtpObserver := NewMockFunc(t)
		router.observer.Once("newrtpobserver", onObserverNewRtpObserver.Fn())

		var err error
		AudioLevelObserver, err = router.CreateAudioLevelObserver()
		require.NoError(t, err)

		onObserverNewRtpObserver.ExpectCalled()
		onObserverNewRtpObserver.ExpectCalledWith(AudioLevelObserver)
		assert.False(t, AudioLevelObserver.Closed())
		assert.False(t, AudioLevelObserver.Paused())
	})

	t.Run("AudioLevelObserver.pause() and resume() succeed", func(t *testing.T) {
		AudioLevelObserver.Pause()
		assert.True(t, AudioLevelObserver.Paused())
		AudioLevelObserver.Resume()
		assert.False(t, AudioLevelObserver.Paused())
	})

	t.Run("AudioLevelObserver.close() succeeds", func(t *testing.T) {
		// We need different a AudioLevelObserver instance here.
		AudioLevelObserver2, _ := router.CreateAudioLevelObserver()

		dump, _ := router.Dump()
		assert.Len(t, dump.RtpObserverIds, 2)

		AudioLevelObserver2.Close()
		assert.True(t, AudioLevelObserver2.Closed())

		dump, _ = router.Dump()
		assert.Len(t, dump.RtpObserverIds, 1)
	})

	t.Run(`AudioLevelObserver emits "routerclose" if Router is closed`, func(t *testing.T) {
		// We need different Router and AudioLevelObserver instances here.
		router2, _ := worker.CreateRouter(RouterOptions{MediaCodecs: mediaCodecs})
		AudioLevelObserver2, _ := router2.CreateAudioLevelObserver()
		onObserverNewRtpObserver := NewMockFunc(t)
		AudioLevelObserver2.On("routerclose", onObserverNewRtpObserver.Fn())
		router2.Close()
		assert.Equal(t, 1, onObserverNewRtpObserver.CalledTimes())
		assert.True(t, AudioLevelObserver2.Closed())
	})

	t.Run(`AudioLevelObserver emits "routerclose" if Worker is closed`, func(t *testing.T) {
		onObserverNewRtpObserver := NewMockFunc(t)
		AudioLevelObserver.On("routerclose", onObserverNewRtpObserver.Fn())
		worker.Close()

		assert.Equal(t, 1, onObserverNewRtpObserver.CalledTimes())
		assert.True(t, AudioLevelObserver.Closed())
	})
}
