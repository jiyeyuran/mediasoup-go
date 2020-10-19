package mediasoup

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	audioLevelMediaCodecs = []*RtpCodecCapability{
		{
			Kind:      MediaKind_Audio,
			MimeType:  "audio/opus",
			ClockRate: 48000,
			Channels:  2,
			Parameters: RtpCodecSpecificParameters{
				Useinbandfec: 1,
			},
		},
	}
)

func TestCreateAudioLevelObserver_Succeeds(t *testing.T) {
	worker := CreateTestWorker()
	router, _ := worker.CreateRouter(RouterOptions{
		MediaCodecs: audioLevelMediaCodecs,
	})
	audioLevelObserver, err := router.CreateAudioLevelObserver()
	assert.NoError(t, err)
	assert.False(t, audioLevelObserver.Closed())
	assert.False(t, audioLevelObserver.Paused())

	result, _ := router.Dump()

	assert.Equal(t, []string{audioLevelObserver.Id()}, result.RtpObserverIds)
}

func TestCreateAudioLevelObserver_TypeError(t *testing.T) {
	worker := CreateTestWorker()
	router, _ := worker.CreateRouter(RouterOptions{
		MediaCodecs: audioLevelMediaCodecs,
	})
	_, err := router.CreateAudioLevelObserver(func(o *AudioLevelObserverOptions) {
		o.MaxEntries = 0
	})
	assert.IsType(t, err, NewTypeError(""))
}

func TestCreateAudioLevelObserver_Pause_Resume(t *testing.T) {
	worker := CreateTestWorker()
	router, _ := worker.CreateRouter(RouterOptions{
		MediaCodecs: audioLevelMediaCodecs,
	})
	audioLevelObserver, err := router.CreateAudioLevelObserver()
	assert.NoError(t, err)

	audioLevelObserver.Pause()
	assert.True(t, audioLevelObserver.Paused())

	audioLevelObserver.Resume()
	assert.False(t, audioLevelObserver.Paused())
}

func TestCreateAudioLevelObserver_Close(t *testing.T) {
	worker := CreateTestWorker()
	router, _ := worker.CreateRouter(RouterOptions{
		MediaCodecs: audioLevelMediaCodecs,
	})
	_, err := router.CreateAudioLevelObserver()
	assert.NoError(t, err)
	audioLevelObserver2, err := router.CreateAudioLevelObserver()
	assert.NoError(t, err)

	result, _ := router.Dump()
	assert.Equal(t, 2, len(result.RtpObserverIds))

	audioLevelObserver2.Close()
	assert.True(t, audioLevelObserver2.Closed())

	result, _ = router.Dump()
	assert.Equal(t, 1, len(result.RtpObserverIds))
}

func TestCreateAudioLevelObserver_Router_Close(t *testing.T) {
	worker := CreateTestWorker()
	router, _ := worker.CreateRouter(RouterOptions{
		MediaCodecs: audioLevelMediaCodecs,
	})
	audioLevelObserver, err := router.CreateAudioLevelObserver()
	assert.NoError(t, err)

	routerclose := false
	audioLevelObserver.On("routerclose", func() {
		routerclose = true
	})
	router.Close()

	Wait(time.Millisecond)

	assert.True(t, audioLevelObserver.Closed())
	assert.True(t, routerclose)
}

func TestCreateAudioLevelObserver_Worker_Close(t *testing.T) {
	worker := CreateTestWorker()
	router, _ := worker.CreateRouter(RouterOptions{
		MediaCodecs: audioLevelMediaCodecs,
	})

	audioLevelObserver, err := router.CreateAudioLevelObserver()
	assert.NoError(t, err)

	routerclose := false
	audioLevelObserver.On("routerclose", func() {
		routerclose = true
	})
	worker.Close()

	Wait(time.Millisecond)

	assert.True(t, audioLevelObserver.Closed())
	assert.True(t, routerclose)
}
