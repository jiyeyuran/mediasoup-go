package mediasoup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	audioLevelMediaCodecs = []RtpCodecCapability{
		{
			Kind:      "audio",
			MimeType:  "audio/opus",
			ClockRate: 48000,
			Channels:  2,
			Parameters: &RtpParameter{
				Useinbandfec: 1,
			},
		},
	}
)

func TestCreateAudioLevelObserver_Succeeds(t *testing.T) {
	worker := CreateTestWorker()
	router, _ := worker.CreateRouter(audioLevelMediaCodecs)
	audioLevelObserver, err := router.CreateAudioLevelObserver(nil)

	assert.NoError(t, err)
	assert.False(t, audioLevelObserver.Closed())
	assert.False(t, audioLevelObserver.Paused())

	dump := router.Dump()

	var result struct {
		RtpObserverIds []string
	}
	assert.NoError(t, dump.Result(&result))
	assert.Equal(t, []string{audioLevelObserver.Id()}, result.RtpObserverIds)
}

func TestCreateAudioLevelObserver_TypeError(t *testing.T) {
	worker := CreateTestWorker()
	router, _ := worker.CreateRouter(audioLevelMediaCodecs)
	_, err := router.CreateAudioLevelObserver(&CreateAudioLevelObserverParams{
		MaxEntries: 0,
	})
	assert.IsType(t, err, NewTypeError(""))
}

func TestCreateAudioLevelObserver_Pause_Resume(t *testing.T) {
	worker := CreateTestWorker()
	router, _ := worker.CreateRouter(audioLevelMediaCodecs)
	audioLevelObserver, err := router.CreateAudioLevelObserver(nil)

	assert.NoError(t, err)

	audioLevelObserver.Pause()

	assert.True(t, audioLevelObserver.Paused())

	audioLevelObserver.Resume()

	assert.False(t, audioLevelObserver.Paused())
}

func TestCreateAudioLevelObserver_Close(t *testing.T) {
	worker := CreateTestWorker()
	router, _ := worker.CreateRouter(audioLevelMediaCodecs)
	_, err := router.CreateAudioLevelObserver(nil)
	assert.NoError(t, err)
	audioLevelObserver2, err := router.CreateAudioLevelObserver(nil)
	assert.NoError(t, err)

	dump := router.Dump()
	var result struct {
		RtpObserverIds []string
	}
	assert.NoError(t, dump.Result(&result))

	assert.Equal(t, 2, len(result.RtpObserverIds))

	audioLevelObserver2.Close()

	assert.True(t, audioLevelObserver2.Closed())

	dump = router.Dump()
	assert.NoError(t, dump.Result(&result))

	assert.Equal(t, 1, len(result.RtpObserverIds))
}

func TestCreateAudioLevelObserver_Router_Close(t *testing.T) {
	worker := CreateTestWorker()
	router, _ := worker.CreateRouter(audioLevelMediaCodecs)
	audioLevelObserver, err := router.CreateAudioLevelObserver(nil)
	assert.NoError(t, err)

	routerclose := false
	audioLevelObserver.On("routerclose", func() {
		routerclose = true
	})
	router.Close()

	assert.True(t, audioLevelObserver.Closed())
	assert.True(t, routerclose)
}

func TestCreateAudioLevelObserver_Worker_Close(t *testing.T) {
	worker := CreateTestWorker()
	router, err := worker.CreateRouter(audioLevelMediaCodecs)
	assert.NoError(t, err)

	audioLevelObserver, err := router.CreateAudioLevelObserver(nil)
	assert.NoError(t, err)

	routerclose := false
	audioLevelObserver.On("routerclose", func() {
		routerclose = true
	})
	worker.Close()

	assert.True(t, audioLevelObserver.Closed())
	assert.True(t, routerclose)
}
