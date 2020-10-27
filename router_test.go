package mediasoup

import (
	"testing"

	"github.com/jiyeyuran/mediasoup-go/h264"
	"github.com/stretchr/testify/assert"
)

var testRouterMediaCodecs = []*RtpCodecCapability{
	{
		Kind:      "audio",
		MimeType:  "audio/opus",
		ClockRate: 48000,
		Channels:  2,
		Parameters: RtpCodecSpecificParameters{
			Useinbandfec: 1,
		},
	},
	{
		Kind:      "video",
		MimeType:  "video/VP8",
		ClockRate: 90000,
	},
	{
		Kind:      "video",
		MimeType:  "video/H264",
		ClockRate: 90000,
		Parameters: RtpCodecSpecificParameters{
			RtpParameter: h264.RtpParameter{
				LevelAsymmetryAllowed: 1,
				PacketizationMode:     1,
				ProfileLevelId:        "4d0032",
			},
		},
	},
}

func TestCreateRouter_Succeeds(t *testing.T) {
	worker := CreateTestWorker()

	onObserverNewRouter := NewMockFunc(t)
	worker.Observer().Once("newrouter", onObserverNewRouter.Fn())

	router, err := worker.CreateRouter(RouterOptions{
		MediaCodecs: testRouterMediaCodecs,
	})
	assert.NoError(t, err)

	onObserverNewRouter.ExpectCalledTimes(1)
	onObserverNewRouter.ExpectCalledWith(router)
	assert.False(t, router.Closed())

	dump, _ := worker.Dump()
	expectDump := WorkerDump{
		Pid:       worker.Pid(),
		RouterIds: []string{router.Id()},
	}
	assert.Equal(t, expectDump, dump)

	routerDump, _ := router.Dump()

	assert.Equal(t, router.Id(), routerDump.Id)
	assert.Equal(t, 1, syncMapLen(&worker.routers))

	worker.Close()

	assert.True(t, router.Closed())
	assert.Equal(t, 0, syncMapLen(&worker.routers))
}

func TestCreateRouter_TypeError(t *testing.T) {
	worker := CreateTestWorker()
	_, err := worker.CreateRouter(RouterOptions{})
	assert.IsType(t, err, NewTypeError(""))
}

func TestCreateRouter_InvalidStateError(t *testing.T) {
	worker := CreateTestWorker()
	worker.Close()

	_, err := worker.CreateRouter(RouterOptions{
		MediaCodecs: testRouterMediaCodecs,
	})
	assert.Error(t, err, NewInvalidStateError(""))
}

func TestRouterClose_Succeeds(t *testing.T) {
	worker := CreateTestWorker()

	onObserverClose := NewMockFunc(t)
	router, _ := worker.CreateRouter(RouterOptions{
		MediaCodecs: testRouterMediaCodecs,
	})
	router.Observer().Once("close", onObserverClose.Fn())
	router.Close()

	onObserverClose.ExpectCalled()
	assert.True(t, router.Closed())
}

func TestRouterEmitsWorkCloseIfWorkerIsClosed(t *testing.T) {
	worker := CreateTestWorker()
	onObserverClose := NewMockFunc(t)
	router, _ := worker.CreateRouter(RouterOptions{
		MediaCodecs: testRouterMediaCodecs,
	})
	router.Observer().Once("close", onObserverClose.Fn())
	worker.Close()
	onObserverClose.ExpectCalled()
	assert.True(t, router.Closed())
}
