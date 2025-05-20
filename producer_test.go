package mediasoup

import (
	"context"
	"testing"
	"time"

	FbsCommon "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Common"
	FbsNotification "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Notification"
	FbsProducer "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Producer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func createAudioProducer(tranpsort *Transport) *Producer {
	producer, err := tranpsort.Produce(&ProducerOptions{
		Kind: MediaKindAudio,
		RtpParameters: &RtpParameters{
			Mid: "AUDIO",
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "audio/opus",
					PayloadType: 0,
					ClockRate:   48000,
					Channels:    2,
					Parameters: RtpCodecSpecificParameters{
						Useinbandfec: 1,
						Usedtx:       1,
					},
				},
			},
			HeaderExtensions: []*RtpHeaderExtensionParameters{
				{
					Uri: "urn:ietf:params:rtp-hdrext:sdes:mid",
					Id:  10,
				},
				{
					Uri: "urn:ietf:params:rtp-hdrext:ssrc-audio-level",
					Id:  12,
				},
			},
			Encodings: []*RtpEncodingParameters{{Ssrc: 11111111, Dtx: true}},
			Rtcp: &RtcpParameters{
				Cname: "audio-1",
			},
		},
		AppData: H{"foo": 1, "bar": "2"},
	})

	if err != nil {
		panic(err)
	}
	return producer
}

func createVideoProducer(tranpsort *Transport) *Producer {
	producer, err := tranpsort.Produce(&ProducerOptions{
		Kind: MediaKindVideo,
		RtpParameters: &RtpParameters{
			Mid: "VIDEO",
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "video/h264",
					PayloadType: 112,
					ClockRate:   90000,
					Parameters: RtpCodecSpecificParameters{
						PacketizationMode: 1,
						ProfileLevelId:    "4d0032",
					},
					RtcpFeedback: []*RtcpFeedback{
						{Type: "nack", Parameter: ""},
						{Type: "nack", Parameter: "pli"},
						{Type: "goog-remb", Parameter: ""},
					},
				},
				{
					MimeType:    "video/rtx",
					PayloadType: 113,
					ClockRate:   90000,
					Parameters:  RtpCodecSpecificParameters{Apt: 112},
				},
			},
			HeaderExtensions: []*RtpHeaderExtensionParameters{
				{
					Uri: "urn:ietf:params:rtp-hdrext:sdes:mid",
					Id:  10,
				},
				{
					Uri: "urn:3gpp:video-orientation",
					Id:  13,
				},
			},
			Encodings: []*RtpEncodingParameters{
				{Ssrc: 22222222, Rtx: &RtpEncodingRtx{Ssrc: 22222223}, ScalabilityMode: "L1T5"},
				{Ssrc: 22222224, Rtx: &RtpEncodingRtx{Ssrc: 22222225}, ScalabilityMode: "L1T5"},
				{Ssrc: 22222226, Rtx: &RtpEncodingRtx{Ssrc: 22222227}, ScalabilityMode: "L1T5"},
				{Ssrc: 22222228, Rtx: &RtpEncodingRtx{Ssrc: 22222229}, ScalabilityMode: "L1T5"},
			},
			Rtcp: &RtcpParameters{
				Cname: "video-1",
			},
		},
		AppData: H{"foo": 1, "bar": "2"},
	})

	if err != nil {
		panic(err)
	}

	return producer
}

func TestProducerDump(t *testing.T) {
	transport := createWebRtcTransport(nil)
	audioProducer := createAudioProducer(transport)
	data, _ := audioProducer.Dump()

	assert.Equal(t, audioProducer.Id(), data.Id)
	assert.Equal(t, audioProducer.Kind(), data.Kind)
	assert.Len(t, data.RtpParameters.Codecs, 1)
	assert.Equal(t, "audio/opus", data.RtpParameters.Codecs[0].MimeType)
	assert.EqualValues(t, 0, data.RtpParameters.Codecs[0].PayloadType)
	assert.EqualValues(t, 48000, data.RtpParameters.Codecs[0].ClockRate)
	assert.EqualValues(t, 2, data.RtpParameters.Codecs[0].Channels)
	assert.Equal(t, RtpCodecSpecificParameters{
		Useinbandfec: 1,
		Usedtx:       1,
	}, data.RtpParameters.Codecs[0].Parameters)
	assert.Empty(t, data.RtpParameters.Codecs[0].RtcpFeedback)
	assert.Len(t, data.RtpParameters.HeaderExtensions, 2)
	assert.Equal(t, []*RtpHeaderExtensionParameters{
		{
			Uri:        "urn:ietf:params:rtp-hdrext:sdes:mid",
			Id:         10,
			Parameters: RtpCodecSpecificParameters{},
			Encrypt:    false,
		},
		{
			Uri:        "urn:ietf:params:rtp-hdrext:ssrc-audio-level",
			Id:         12,
			Parameters: RtpCodecSpecificParameters{},
			Encrypt:    false,
		},
	}, data.RtpParameters.HeaderExtensions)
	assert.Len(t, data.RtpParameters.Encodings, 1)
	assert.EqualValues(t, 0, *data.RtpParameters.Encodings[0].CodecPayloadType)
	assert.Equal(t, ProducerSimple, data.Type)

	videoProducer := createVideoProducer(transport)
	data, _ = videoProducer.Dump()

	assert.Equal(t, videoProducer.Id(), data.Id)
	assert.Equal(t, videoProducer.Kind(), data.Kind)
	assert.Len(t, data.RtpParameters.Codecs, 2)
	assert.Equal(t, "video/H264", data.RtpParameters.Codecs[0].MimeType)
	assert.EqualValues(t, 112, data.RtpParameters.Codecs[0].PayloadType)
	assert.EqualValues(t, 90000, data.RtpParameters.Codecs[0].ClockRate)
	assert.Empty(t, data.RtpParameters.Codecs[0].Channels)
	assert.Equal(t, RtpCodecSpecificParameters{
		PacketizationMode: 1,
		ProfileLevelId:    "4d0032",
	}, data.RtpParameters.Codecs[0].Parameters)
	assert.Equal(t, []*RtcpFeedback{
		{Type: "nack"},
		{Type: "nack", Parameter: "pli"},
		{Type: "goog-remb"},
	}, data.RtpParameters.Codecs[0].RtcpFeedback)

	assert.Equal(t, "video/rtx", data.RtpParameters.Codecs[1].MimeType)
	assert.EqualValues(t, 113, data.RtpParameters.Codecs[1].PayloadType)
	assert.EqualValues(t, 90000, data.RtpParameters.Codecs[1].ClockRate)
	assert.Empty(t, data.RtpParameters.Codecs[1].Channels)
	assert.Equal(t, RtpCodecSpecificParameters{
		Apt: 112,
	}, data.RtpParameters.Codecs[1].Parameters)
	assert.Empty(t, data.RtpParameters.Codecs[1].RtcpFeedback)
	assert.Len(t, data.RtpParameters.HeaderExtensions, 2)
	assert.Equal(t, []*RtpHeaderExtensionParameters{
		{
			Uri:        "urn:ietf:params:rtp-hdrext:sdes:mid",
			Id:         10,
			Parameters: RtpCodecSpecificParameters{},
			Encrypt:    false,
		},
		{
			Uri:        "urn:3gpp:video-orientation",
			Id:         13,
			Parameters: RtpCodecSpecificParameters{},
			Encrypt:    false,
		},
	}, data.RtpParameters.HeaderExtensions)
	assert.Len(t, data.RtpParameters.Encodings, 4)
	assert.Equal(t, []*RtpEncodingParameters{
		{CodecPayloadType: ref[uint8](112), Ssrc: 22222222, Rtx: &RtpEncodingRtx{Ssrc: 22222223}, ScalabilityMode: "L1T5"},
		{CodecPayloadType: ref[uint8](112), Ssrc: 22222224, Rtx: &RtpEncodingRtx{Ssrc: 22222225}, ScalabilityMode: "L1T5"},
		{CodecPayloadType: ref[uint8](112), Ssrc: 22222226, Rtx: &RtpEncodingRtx{Ssrc: 22222227}, ScalabilityMode: "L1T5"},
		{CodecPayloadType: ref[uint8](112), Ssrc: 22222228, Rtx: &RtpEncodingRtx{Ssrc: 22222229}, ScalabilityMode: "L1T5"},
	}, data.RtpParameters.Encodings)
	assert.Equal(t, ProducerSimulcast, data.Type)
}

func TestProducerGetStats(t *testing.T) {
	transport := createWebRtcTransport(nil)
	audioProducer := createAudioProducer(transport)
	stats, _ := audioProducer.GetStats()
	assert.Empty(t, stats)

	videoProducer := createVideoProducer(transport)
	stats, _ = videoProducer.GetStats()
	assert.Empty(t, stats)
}

func TestProducerPauseAndResume(t *testing.T) {
	transport := createWebRtcTransport(nil)
	audioProducer := createAudioProducer(transport)
	audioProducer.Pause()
	assert.True(t, audioProducer.Paused())

	data, _ := audioProducer.Dump()
	assert.True(t, data.Paused)

	audioProducer.Resume()
	assert.False(t, audioProducer.Paused())

	data, _ = audioProducer.Dump()
	assert.False(t, data.Paused)
}

func TestProducerEnableTraceEvent(t *testing.T) {
	transport := createWebRtcTransport(nil)
	audioProducer := createAudioProducer(transport)
	audioProducer.EnableTraceEvent([]ProducerTraceEventType{"rtp", "pli"})
	data, _ := audioProducer.Dump()
	assert.Equal(t, []ProducerTraceEventType{"rtp", "pli"}, data.TraceEventTypes)

	audioProducer.EnableTraceEvent(nil)
	data, _ = audioProducer.Dump()
	assert.Empty(t, data.TraceEventTypes)

	audioProducer.EnableTraceEvent([]ProducerTraceEventType{"nack", "FOO", "fir"})
	data, _ = audioProducer.Dump()
	assert.Equal(t, []ProducerTraceEventType{"nack", "fir"}, data.TraceEventTypes)

	audioProducer.EnableTraceEvent(nil)
	data, _ = audioProducer.Dump()
	assert.Empty(t, data.TraceEventTypes)
}

func TestProducerSend(t *testing.T) {
	transport := createDirectTransport(nil)
	audioProducer := createAudioProducer(transport)
	err := audioProducer.Send([]byte{})
	assert.NoError(t, err)
}

func TestProducerHandlers(t *testing.T) {
	transport := createWebRtcTransport(nil)
	videoProducer := createVideoProducer(transport)

	mymock := new(MockedHandler)
	defer mymock.AssertExpectations(t)

	mymock.On("OnProducerScore", []ProducerScore{
		{Ssrc: 11, Score: 10},
	})
	mymock.On("OnProducerScore", []ProducerScore{
		{EncodingIdx: 0, Ssrc: 11, Score: 9},
		{EncodingIdx: 1, Ssrc: 22, Score: 8},
	})
	mymock.On("OnProducerScore", []ProducerScore{
		{EncodingIdx: 0, Ssrc: 11, Score: 9},
		{EncodingIdx: 1, Ssrc: 22, Score: 9},
	})

	channel := videoProducer.channel
	videoProducer.OnScore(mymock.OnProducerScore)
	videoProducer.OnVideoOrientationChange(mymock.OnProducerVideoOrientation)
	videoProducer.OnTrace(mymock.OnProducerEventTrace)

	channel.ProcessNotificationForTesting(&FbsNotification.NotificationT{
		HandlerId: videoProducer.Id(),
		Event:     FbsNotification.EventPRODUCER_SCORE,
		Body: &FbsNotification.BodyT{
			Type: FbsNotification.BodyProducer_ScoreNotification,
			Value: &FbsProducer.ScoreNotificationT{
				Scores: []*FbsProducer.ScoreT{
					{EncodingIdx: 0, Ssrc: 11, Score: 10},
				},
			},
		},
	})
	channel.ProcessNotificationForTesting(&FbsNotification.NotificationT{
		HandlerId: videoProducer.Id(),
		Event:     FbsNotification.EventPRODUCER_SCORE,
		Body: &FbsNotification.BodyT{
			Type: FbsNotification.BodyProducer_ScoreNotification,
			Value: &FbsProducer.ScoreNotificationT{
				Scores: []*FbsProducer.ScoreT{
					{EncodingIdx: 0, Ssrc: 11, Score: 9},
					{EncodingIdx: 1, Ssrc: 22, Score: 8},
				},
			},
		},
	})
	channel.ProcessNotificationForTesting(&FbsNotification.NotificationT{
		HandlerId: videoProducer.Id(),
		Event:     FbsNotification.EventPRODUCER_SCORE,
		Body: &FbsNotification.BodyT{
			Type: FbsNotification.BodyProducer_ScoreNotification,
			Value: &FbsProducer.ScoreNotificationT{
				Scores: []*FbsProducer.ScoreT{
					{EncodingIdx: 0, Ssrc: 11, Score: 9},
					{EncodingIdx: 1, Ssrc: 22, Score: 9},
				},
			},
		},
	})

	mymock.On("OnProducerVideoOrientation", ProducerVideoOrientation{})

	channel.ProcessNotificationForTesting(&FbsNotification.NotificationT{
		HandlerId: videoProducer.Id(),
		Event:     FbsNotification.EventPRODUCER_VIDEO_ORIENTATION_CHANGE,
		Body: &FbsNotification.BodyT{
			Type:  FbsNotification.BodyProducer_VideoOrientationChangeNotification,
			Value: &FbsProducer.VideoOrientationChangeNotificationT{},
		},
	})

	mymock.On("OnProducerEventTrace", ProducerTraceEventData{
		Type:      ProducerTraceEventRtp,
		Direction: "out",
		Timestamp: 123456789,
	})

	channel.ProcessNotificationForTesting(&FbsNotification.NotificationT{
		HandlerId: videoProducer.Id(),
		Event:     FbsNotification.EventPRODUCER_TRACE,
		Body: &FbsNotification.BodyT{
			Type: FbsNotification.BodyProducer_TraceNotification,
			Value: &FbsProducer.TraceNotificationT{
				Type:      FbsProducer.TraceEventTypeRTP,
				Direction: FbsCommon.TraceDirectionDIRECTION_OUT,
				Timestamp: 123456789,
				Info:      &FbsProducer.TraceInfoT{},
			},
		},
	})

	time.Sleep(time.Millisecond)
}

func TestProducerClose(t *testing.T) {
	t.Run("close normally", func(t *testing.T) {
		mymock := new(MockedHandler)
		defer mymock.AssertExpectations(t)

		mymock.On("OnClose", mock.IsType(context.Background())).Once()

		router := createRouter(nil)
		transport := createWebRtcTransport(router)
		audioProducer := createAudioProducer(transport)
		audioProducer.OnClose(mymock.OnClose)

		err := audioProducer.Close()
		assert.NoError(t, err)
		assert.True(t, audioProducer.Closed())

		routerDump, _ := router.Dump()

		assert.Empty(t, routerDump.MapProducerIdConsumerIds)
		assert.Empty(t, routerDump.MapConsumerIdProducerId)

		transportDump, _ := transport.Dump()

		assert.Equal(t, transport.Id(), transportDump.Id)
		assert.Empty(t, transportDump.ProducerIds)
		assert.Empty(t, transportDump.ConsumerIds)

		producer := router.GetDataProducerById(audioProducer.Id())
		assert.Nil(t, producer)

		// methods reject if closed
		_, err = audioProducer.Dump()
		assert.Error(t, err)
		_, err = audioProducer.GetStats()
		assert.Error(t, err)
		assert.Error(t, audioProducer.Pause())
		assert.Error(t, audioProducer.Resume())
	})

	t.Run("transport closed", func(t *testing.T) {
		mymock := new(MockedHandler)
		defer mymock.AssertExpectations(t)

		mymock.On("OnClose", mock.IsType(context.Background())).Once()

		router := createRouter(nil)
		transport := createWebRtcTransport(router)
		audioProducer := createAudioProducer(transport)
		audioProducer.OnClose(mymock.OnClose)

		transport.Close()
		assert.True(t, audioProducer.Closed())
	})

	t.Run("router closed", func(t *testing.T) {
		mymock := new(MockedHandler)
		defer mymock.AssertExpectations(t)

		mymock.On("OnClose", mock.IsType(context.Background())).Once()

		router := createRouter(nil)
		transport := createWebRtcTransport(router)
		audioProducer := createAudioProducer(transport)
		audioProducer.OnClose(mymock.OnClose)

		router.Close()
		assert.True(t, audioProducer.Closed())
	})
}
