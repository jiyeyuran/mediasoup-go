package mediasoup

import (
	"testing"

	"github.com/jiyeyuran/mediasoup-go/mediasoup/h264profile"
	"github.com/stretchr/testify/assert"
)

var testRouterMediaCodecs = []RtpCodecCapability{
	{
		Kind:      "audio",
		MimeType:  "audio/opus",
		ClockRate: 48000,
		Channels:  2,
		Parameters: &RtpParameter{
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
		Parameters: &RtpParameter{
			RtpH264Parameter: h264profile.RtpH264Parameter{
				LevelAsymmetryAllowed: 1,
				PacketizationMode:     1,
				ProfileLevelId:        "4d0032",
			},
		},
	},
}

func TestCreateRouter_Succeeds(t *testing.T) {
	worker := CreateTestWorker()
	called := 0
	var callRouter *Router
	worker.Observer().Once("newrouter", func(router *Router) {
		called++
		callRouter = router
	})

	router, err := worker.CreateRouter(testRouterMediaCodecs)
	assert.NoError(t, err)
	assert.Equal(t, router, callRouter)
	assert.Equal(t, 1, called)
	assert.False(t, router.Closed())

	dump := worker.Dump()

	type Result struct {
		Pid       int
		RouterIds []string
	}
	var result Result
	assert.NoError(t, dump.Result(&result))

	result1 := Result{
		Pid:       worker.Pid(),
		RouterIds: []string{router.Id()},
	}
	assert.Equal(t, result, result1)

	type RouterDumpResult struct {
		Id                       string
		TransportIds             []string
		RtpObserverIds           []string
		MapProducerIdConsumerIds map[string][]string
		MapConsumerIdProducerId  map[string]string
		MapProducerIdObserverIds map[string][]string
	}
	var routerDumpResult1 RouterDumpResult
	dump = router.Dump()
	assert.NoError(t, dump.Result(&routerDumpResult1))
	routerDumpResult2 := RouterDumpResult{
		Id:                       router.Id(),
		TransportIds:             []string{},
		RtpObserverIds:           []string{},
		MapProducerIdConsumerIds: map[string][]string{},
		MapConsumerIdProducerId:  map[string]string{},
		MapProducerIdObserverIds: map[string][]string{},
	}
	assert.Equal(t, routerDumpResult1, routerDumpResult2)
	assert.Equal(t, 1, len(worker.routers))

	worker.Close()

	assert.True(t, router.Closed())
	assert.Equal(t, 0, len(worker.routers))
}

func TestCreateRouter_TypeError(t *testing.T) {
	worker := CreateTestWorker()
	_, err := worker.CreateRouter(nil)

	assert.IsType(t, err, NewTypeError(""))
}

func TestCreateRouter_InvalidStateError(t *testing.T) {
	worker := CreateTestWorker()
	worker.Close()

	_, err := worker.CreateRouter(testRouterMediaCodecs)
	assert.Error(t, err, NewInvalidStateError(""))
}

func TestRouterClose_Succeeds(t *testing.T) {
	worker := CreateTestWorker()
	router, _ := worker.CreateRouter(testRouterMediaCodecs)
	called := 0
	router.Observer().Once("close", func() {
		called++
	})
	router.Close()

	assert.Equal(t, 1, called)
	assert.True(t, router.Closed())
}

func TestRouterEmitsWorkCloseIfWorkerIsClosed(t *testing.T) {
	worker := CreateTestWorker()
	router, _ := worker.CreateRouter(testRouterMediaCodecs)
	called := 0
	router.Observer().Once("close", func() {
		called++
	})
	worker.Close()
	assert.Equal(t, 1, called)
	assert.True(t, router.Closed())
}
