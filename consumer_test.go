package mediasoup

import (
	"context"
	"slices"
	"testing"
	"time"

	FbsConsumer "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Consumer"
	FbsNotification "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var consumerDeviceCapabilities = &RtpCapabilities{
	Codecs: []*RtpCodecCapability{
		{
			MimeType:             "audio/opus",
			Kind:                 "audio",
			PreferredPayloadType: 100,
			ClockRate:            48000,
			Channels:             2,
			RtcpFeedback: []*RtcpFeedback{
				{Type: "nack"},
			},
		},
		{
			MimeType:             "video/H264",
			Kind:                 "video",
			PreferredPayloadType: 101,
			ClockRate:            90000,
			Parameters: RtpCodecSpecificParameters{
				LevelAsymmetryAllowed: 1,
				PacketizationMode:     1,
				ProfileLevelId:        "4d0032",
			},
			RtcpFeedback: []*RtcpFeedback{
				{Type: "nack", Parameter: ""},
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "goog-remb", Parameter: ""},
			},
		},
		{
			MimeType:             "video/rtx",
			Kind:                 "video",
			PreferredPayloadType: 102,
			ClockRate:            90000,
			Parameters: RtpCodecSpecificParameters{
				Apt: 101,
			},
		},
	},
	HeaderExtensions: []*RtpHeaderExtension{
		{
			Kind:             "audio",
			Uri:              "urn:ietf:params:rtp-hdrext:sdes:mid",
			PreferredId:      1,
			PreferredEncrypt: false,
		},
		{
			Kind:             "video",
			Uri:              "urn:ietf:params:rtp-hdrext:sdes:mid",
			PreferredId:      1,
			PreferredEncrypt: false,
		},
		{
			Kind:             "video",
			Uri:              "urn:ietf:params:rtp-hdrext:sdes:rtp-stream-id",
			PreferredId:      2,
			PreferredEncrypt: false,
		},
		{
			Kind:             "audio",
			Uri:              "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time", // eslint-disable-line max-len
			PreferredId:      4,
			PreferredEncrypt: false,
		},
		{
			Kind:             "video",
			Uri:              "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time", // eslint-disable-line max-len
			PreferredId:      4,
			PreferredEncrypt: false,
		},
		{
			Kind:             "audio",
			Uri:              "urn:ietf:params:rtp-hdrext:ssrc-audio-level",
			PreferredId:      6,
			PreferredEncrypt: false,
		},
		{
			Kind:             "video",
			Uri:              "urn:3gpp:video-orientation",
			PreferredId:      8,
			PreferredEncrypt: false,
		},
		{
			Kind:             "video",
			Uri:              "urn:ietf:params:rtp-hdrext:toffset",
			PreferredId:      9,
			PreferredEncrypt: false,
		},
	},
}

func createConsumer(transport *Transport, producerId string, options ...func(*ConsumerOptions)) *Consumer {
	opts := &ConsumerOptions{
		ProducerId:      producerId,
		RtpCapabilities: consumerDeviceCapabilities,
		AppData:         H{"baz": "LOL"},
	}
	for _, o := range options {
		o(opts)
	}
	consumer, err := transport.Consume(opts)
	if err != nil {
		panic(err)
	}
	return consumer
}

func TestConsumerDump(t *testing.T) {
	transport := createPlainTransport(nil)
	audioProducer := createAudioProducer(transport)
	audioConsumer := createConsumer(transport, audioProducer.Id())

	data, _ := audioConsumer.Dump()

	assert.Equal(t, audioConsumer.Id(), data.Id)
	assert.Equal(t, audioConsumer.Kind(), data.Kind)
	assert.NotEmpty(t, data.RtpParameters)
	assert.Len(t, data.RtpParameters.Codecs, 1)
	assert.Equal(t, "audio/opus", data.RtpParameters.Codecs[0].MimeType)
	assert.EqualValues(t, 100, data.RtpParameters.Codecs[0].PayloadType)
	assert.EqualValues(t, 48000, data.RtpParameters.Codecs[0].ClockRate)
	assert.EqualValues(t, 2, data.RtpParameters.Codecs[0].Channels)
	assert.Equal(t, RtpCodecSpecificParameters{Useinbandfec: 1, Usedtx: 1}, data.RtpParameters.Codecs[0].Parameters)
	assert.Equal(t, []*RtcpFeedback{{Type: "nack"}}, data.RtpParameters.Codecs[0].RtcpFeedback)
	assert.Len(t, data.RtpParameters.HeaderExtensions, 3)
	assert.Equal(t, []*RtpHeaderExtensionParameters{
		{
			Uri:        "urn:ietf:params:rtp-hdrext:sdes:mid",
			Id:         1,
			Encrypt:    false,
			Parameters: RtpCodecSpecificParameters{},
		},
		{
			Uri:        "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time",
			Id:         4,
			Encrypt:    false,
			Parameters: RtpCodecSpecificParameters{},
		},
		{
			Uri:        "urn:ietf:params:rtp-hdrext:ssrc-audio-level",
			Id:         6,
			Encrypt:    false,
			Parameters: RtpCodecSpecificParameters{},
		},
	}, data.RtpParameters.HeaderExtensions)
	assert.Len(t, data.RtpParameters.Encodings, 1)
	assert.Equal(t, []*RtpEncodingParameters{
		{CodecPayloadType: ref[uint8](100), Ssrc: audioConsumer.RtpParameters().Encodings[0].Ssrc, ScalabilityMode: "S1T1"},
	}, data.RtpParameters.Encodings)
	assert.Equal(t, ConsumerSimple, data.Type)
	assert.Len(t, data.ConsumableRtpEncodings, 1)
	assert.Equal(t, []*RtpEncodingParameters{
		{Ssrc: audioProducer.ConsumableRtpParameters().Encodings[0].Ssrc, Dtx: true, ScalabilityMode: "S1T1"},
	}, data.ConsumableRtpEncodings)
	assert.Equal(t, []int{100}, data.SupportedCodecPayloadTypes)
	assert.False(t, data.Paused)
	assert.False(t, data.ProducerPaused)
	assert.Equal(t, "1111-1111-1111-1111 2222-2222-2222-2222", data.RtpParameters.Msid)

	videoProducer := createVideoProducer(transport)
	videoProducer.Pause()
	videoConsumer := createConsumer(transport, videoProducer.Id(), func(co *ConsumerOptions) {
		co.Paused = true
	})
	data, _ = videoConsumer.Dump()

	assert.Equal(t, videoConsumer.Id(), data.Id)
	assert.Equal(t, videoConsumer.Kind(), data.Kind)
	assert.NotEmpty(t, data.RtpParameters)
	assert.Len(t, data.RtpParameters.Codecs, 2)
	assert.Equal(t, "video/H264", data.RtpParameters.Codecs[0].MimeType)
	assert.EqualValues(t, 103, data.RtpParameters.Codecs[0].PayloadType)
	assert.EqualValues(t, 90000, data.RtpParameters.Codecs[0].ClockRate)
	assert.Empty(t, data.RtpParameters.Codecs[0].Channels)
	assert.Equal(t, RtpCodecSpecificParameters{
		PacketizationMode: 1,
		ProfileLevelId:    "4d0032",
	}, data.RtpParameters.Codecs[0].Parameters)
	assert.Equal(t, []*RtcpFeedback{
		{Type: "nack"},
		{Type: "nack", Parameter: "pli"},
		{Type: "ccm", Parameter: "fir"},
		{Type: "goog-remb"},
	}, data.RtpParameters.Codecs[0].RtcpFeedback)
	assert.Len(t, data.RtpParameters.HeaderExtensions, 4)
	assert.Equal(t, []*RtpHeaderExtensionParameters{
		{
			Uri:        "urn:ietf:params:rtp-hdrext:sdes:mid",
			Id:         1,
			Encrypt:    false,
			Parameters: RtpCodecSpecificParameters{},
		},
		{
			Uri:        "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time",
			Id:         4,
			Encrypt:    false,
			Parameters: RtpCodecSpecificParameters{},
		},
		{
			Uri:        "urn:3gpp:video-orientation",
			Id:         8,
			Encrypt:    false,
			Parameters: RtpCodecSpecificParameters{},
		},
		{
			Uri:        "urn:ietf:params:rtp-hdrext:toffset",
			Id:         9,
			Encrypt:    false,
			Parameters: RtpCodecSpecificParameters{},
		},
	}, data.RtpParameters.HeaderExtensions)
	assert.Len(t, data.RtpParameters.Encodings, 1)
	assert.Equal(t, []*RtpEncodingParameters{
		{
			CodecPayloadType: ref[uint8](103),
			Ssrc:             videoConsumer.RtpParameters().Encodings[0].Ssrc,
			Rtx: &RtpEncodingRtx{
				Ssrc: videoConsumer.RtpParameters().Encodings[0].Rtx.Ssrc,
			},
			ScalabilityMode: "L4T5",
		},
	}, data.RtpParameters.Encodings)
	assert.Len(t, data.ConsumableRtpEncodings, 4)
	assert.Equal(t, []*RtpEncodingParameters{
		{Ssrc: videoProducer.ConsumableRtpParameters().Encodings[0].Ssrc, ScalabilityMode: "L1T5"},
		{Ssrc: videoProducer.ConsumableRtpParameters().Encodings[1].Ssrc, ScalabilityMode: "L1T5"},
		{Ssrc: videoProducer.ConsumableRtpParameters().Encodings[2].Ssrc, ScalabilityMode: "L1T5"},
		{Ssrc: videoProducer.ConsumableRtpParameters().Encodings[3].Ssrc, ScalabilityMode: "L1T5"},
	}, data.ConsumableRtpEncodings)
	assert.Equal(t, []int{103}, data.SupportedCodecPayloadTypes)
	assert.True(t, data.Paused)
	assert.True(t, data.ProducerPaused)
	assert.Empty(t, data.RtpParameters.Msid)
}

func TestConsumerGetStats(t *testing.T) {
	transport := createPlainTransport(nil)
	audioProducer := createAudioProducer(transport)
	audioConsumer := createConsumer(transport, audioProducer.Id())

	consumerStats, err := audioConsumer.GetStats()
	assert.NoError(t, err)

	expectedConsumerStats := []*ConsumerStat{}
	for _, stats := range consumerStats {
		expectedConsumerStats = append(expectedConsumerStats, &ConsumerStat{
			Type:     stats.Type,
			Kind:     stats.Kind,
			MimeType: stats.MimeType,
			Ssrc:     stats.Ssrc,
		})
	}

	assert.Contains(t, expectedConsumerStats, &ConsumerStat{
		Type:     "outbound-rtp",
		Kind:     "audio",
		MimeType: "audio/opus",
		Ssrc:     audioConsumer.RtpParameters().Encodings[0].Ssrc,
	})

	videoProducer := createVideoProducer(transport)
	videoConsumer := createConsumer(transport, videoProducer.Id())
	consumerStats, err = videoConsumer.GetStats()
	assert.NoError(t, err)

	expectedConsumerStats = []*ConsumerStat{}
	for _, stats := range consumerStats {
		expectedConsumerStats = append(expectedConsumerStats, &ConsumerStat{
			Type:     stats.Type,
			Kind:     stats.Kind,
			MimeType: stats.MimeType,
			Ssrc:     stats.Ssrc,
		})
	}

	assert.Contains(t, expectedConsumerStats, &ConsumerStat{
		Type:     "outbound-rtp",
		Kind:     "video",
		MimeType: "video/H264",
		Ssrc:     videoConsumer.RtpParameters().Encodings[0].Ssrc,
	})
}

func TestConsumerPauseAndResume(t *testing.T) {
	transport := createPlainTransport(nil)
	audioProducer := createAudioProducer(transport)
	audioConsumer := createConsumer(transport, audioProducer.Id(), func(co *ConsumerOptions) {
		co.Paused = true
	})

	assert.True(t, audioConsumer.Paused())

	data, _ := audioConsumer.Dump()
	assert.True(t, data.Paused)

	audioConsumer.Resume()

	assert.False(t, audioConsumer.Paused())

	data, _ = audioConsumer.Dump()
	assert.False(t, data.Paused)
}

func TestConsumerSetPreferredLayers(t *testing.T) {
	transport := createPlainTransport(nil)
	audioProducer := createAudioProducer(transport)
	audioConsumer := createConsumer(transport, audioProducer.Id())
	videoProducer := createVideoProducer(transport)
	videoConsumer := createConsumer(transport, videoProducer.Id())

	err := audioConsumer.SetPreferredLayers(ConsumerLayers{SpatialLayer: 1, TemporalLayer: ref[uint8](1)})
	assert.NoError(t, err)
	assert.Nil(t, audioConsumer.PreferredLayers())

	err = videoConsumer.SetPreferredLayers(ConsumerLayers{SpatialLayer: 2, TemporalLayer: ref[uint8](3)})
	assert.NoError(t, err)
	assert.Equal(t, &ConsumerLayers{SpatialLayer: 2, TemporalLayer: ref[uint8](3)}, videoConsumer.PreferredLayers())

	err = videoConsumer.SetPreferredLayers(ConsumerLayers{SpatialLayer: 3})
	assert.NoError(t, err)
	assert.Equal(t, &ConsumerLayers{SpatialLayer: 3, TemporalLayer: ref[uint8](4)}, videoConsumer.PreferredLayers())

	err = videoConsumer.SetPreferredLayers(ConsumerLayers{SpatialLayer: 3, TemporalLayer: ref[uint8](0)})
	assert.NoError(t, err)
	assert.Equal(t, &ConsumerLayers{SpatialLayer: 3, TemporalLayer: ref[uint8](0)}, videoConsumer.PreferredLayers())

	err = videoConsumer.SetPreferredLayers(ConsumerLayers{SpatialLayer: 66, TemporalLayer: ref[uint8](66)})
	assert.NoError(t, err)
	assert.Equal(t, &ConsumerLayers{SpatialLayer: 3, TemporalLayer: ref[uint8](4)}, videoConsumer.PreferredLayers())
}

func TestConsumerSetPriority(t *testing.T) {
	transport := createPlainTransport(nil)
	videoProducer := createVideoProducer(transport)
	videoConsumer := createConsumer(transport, videoProducer.Id())

	err := videoConsumer.SetPriority(2)
	assert.NoError(t, err)

	assert.EqualValues(t, 2, videoConsumer.Priority())
	err = videoConsumer.SetPriority(0)
	assert.Error(t, err)
}

func TestConsumerUnsetPriority(t *testing.T) {
	transport := createPlainTransport(nil)
	videoProducer := createVideoProducer(transport)
	videoConsumer := createConsumer(transport, videoProducer.Id())

	err := videoConsumer.UnsetPriority()
	assert.NoError(t, err)
	assert.EqualValues(t, 1, videoConsumer.Priority())
}

func TestConsumerEnableTraceEvent(t *testing.T) {
	transport := createPlainTransport(nil)
	audioProducer := createAudioProducer(transport)
	audioConsumer := createConsumer(transport, audioProducer.Id())

	audioConsumer.EnableTraceEvent([]ConsumerTraceEventType{"rtp", "pli"})

	dump, _ := audioConsumer.Dump()
	assert.Equal(t, []ConsumerTraceEventType{"rtp", "pli"}, dump.TraceEventTypes)

	audioConsumer.EnableTraceEvent(nil)

	dump, _ = audioConsumer.Dump()
	assert.Empty(t, dump.TraceEventTypes)

	audioConsumer.EnableTraceEvent([]ConsumerTraceEventType{"nack", "FOO", "fir"})

	dump, _ = audioConsumer.Dump()
	assert.Equal(t, []ConsumerTraceEventType{"nack", "fir"}, dump.TraceEventTypes)

	audioConsumer.EnableTraceEvent(nil)

	dump, _ = audioConsumer.Dump()
	assert.Empty(t, dump.TraceEventTypes)
}

func TestConsumerEmitsProducerPauseAndProducerResume(t *testing.T) {
	mymock := new(MockedHandler)
	defer mymock.AssertExpectations(t)

	mymock.On("OnProducerPause", mock.IsType(context.Background())).Once()
	mymock.On("OnProducerResume", mock.IsType(context.Background())).Once()

	transport := createPlainTransport(nil)
	audioProducer := createAudioProducer(transport)
	audioConsumer := createConsumer(transport, audioProducer.Id())
	audioConsumer.OnProducerPause(mymock.OnProducerPause)
	audioConsumer.OnProducerResume(mymock.OnProducerResume)

	audioProducer.Pause()

	time.Sleep(time.Millisecond)

	assert.False(t, audioConsumer.Paused())
	assert.True(t, audioConsumer.ProducerPaused())

	audioProducer.Resume()

	time.Sleep(time.Millisecond)

	assert.False(t, audioConsumer.Paused())
	assert.False(t, audioConsumer.ProducerPaused())
}

func TestConsumerEmitsScore(t *testing.T) {
	mymock := new(MockedHandler)
	defer mymock.AssertExpectations(t)

	transport := createPlainTransport(nil)
	audioProducer := createAudioProducer(transport)
	audioConsumer := createConsumer(transport, audioProducer.Id())
	audioConsumer.OnScore(mymock.OnConsumeScore)
	channel := audioConsumer.channel

	mymock.On("OnConsumeScore", ConsumerScore{
		ProducerScore: 10,
		Score:         9,
	})
	mymock.On("OnConsumeScore", ConsumerScore{
		ProducerScore: 9,
		Score:         9,
	})
	mymock.On("OnConsumeScore", ConsumerScore{
		ProducerScore: 8,
		Score:         8,
	})

	channel.ProcessNotificationForTesting(&FbsNotification.NotificationT{
		HandlerId: audioConsumer.Id(),
		Event:     FbsNotification.EventCONSUMER_SCORE,
		Body: &FbsNotification.BodyT{
			Type: FbsNotification.BodyConsumer_ScoreNotification,
			Value: &FbsConsumer.ScoreNotificationT{
				Score: &FbsConsumer.ConsumerScoreT{
					ProducerScore: 10,
					Score:         9,
				},
			},
		},
	})
	channel.ProcessNotificationForTesting(&FbsNotification.NotificationT{
		HandlerId: audioConsumer.Id(),
		Event:     FbsNotification.EventCONSUMER_SCORE,
		Body: &FbsNotification.BodyT{
			Type: FbsNotification.BodyConsumer_ScoreNotification,
			Value: &FbsConsumer.ScoreNotificationT{
				Score: &FbsConsumer.ConsumerScoreT{
					ProducerScore: 9,
					Score:         9,
				},
			},
		},
	})
	channel.ProcessNotificationForTesting(&FbsNotification.NotificationT{
		HandlerId: audioConsumer.Id(),
		Event:     FbsNotification.EventCONSUMER_SCORE,
		Body: &FbsNotification.BodyT{
			Type: FbsNotification.BodyConsumer_ScoreNotification,
			Value: &FbsConsumer.ScoreNotificationT{
				Score: &FbsConsumer.ConsumerScoreT{
					ProducerScore: 8,
					Score:         8,
				},
			},
		},
	})

	time.Sleep(time.Millisecond)
}

func TestConsumerClose(t *testing.T) {
	mymock := new(MockedHandler)
	defer mymock.AssertExpectations(t)

	mymock.On("OnClose", mock.IsType(context.Background())).Once()

	router := createRouter(nil)
	transport := createWebRtcTransport(router)
	audioProducer := createAudioProducer(transport)
	audioConsumer := createConsumer(transport, audioProducer.Id())
	audioConsumer.OnClose(mymock.OnClose)
	videoProducer := createVideoProducer(transport)

	transport2 := createWebRtcTransport(router)
	videoConsumer := createConsumer(transport2, videoProducer.Id(), func(co *ConsumerOptions) {
		co.Paused = true
	})

	audioConsumer.Close()

	consumer := router.GetConsumerById(audioConsumer.Id())
	assert.Nil(t, consumer)

	routerDump, err := router.Dump()
	assert.NoError(t, err)
	assert.True(t, slices.ContainsFunc(routerDump.MapProducerIdConsumerIds, func(kvs KeyValues[string, string]) bool {
		return kvs.Key == audioProducer.Id()
	}))
	index := slices.IndexFunc(routerDump.MapConsumerIdProducerId, func(kvs KeyValue[string, string]) bool {
		return kvs.Key == videoConsumer.Id()
	})
	require.Greater(t, index, -1)
	require.Equal(t, videoProducer.Id(), routerDump.MapConsumerIdProducerId[index].Value)
}

func TestConsumerCloseByOthers(t *testing.T) {
	t.Run("producer closed", func(t *testing.T) {
		mymock := new(MockedHandler)
		defer mymock.AssertExpectations(t)

		ctx := context.TODO()

		mymock.On("OnProducerClose", ctx).Twice()
		mymock.On("OnClose", ctx).Twice()

		router := createRouter(nil)
		transport := createWebRtcTransport(router)
		audioProducer := createAudioProducer(transport)
		audioConsumer1 := createConsumer(transport, audioProducer.Id())
		audioConsumer1.OnProducerClose(mymock.OnProducerClose)
		audioConsumer1.OnClose(mymock.OnClose)

		audioConsumer2 := createConsumer(transport, audioProducer.Id())
		audioConsumer2.OnProducerClose(mymock.OnProducerClose)
		audioConsumer2.OnClose(mymock.OnClose)

		audioProducer.CloseContext(ctx)

		time.Sleep(time.Millisecond)

		assert.True(t, audioConsumer1.Closed())
		assert.True(t, audioConsumer2.Closed())
	})

	t.Run("transport closed", func(t *testing.T) {
		mymock := new(MockedHandler)
		defer mymock.AssertExpectations(t)

		mymock.On("OnClose", mock.IsType(context.Background())).Once()

		router := createRouter(nil)
		transport := createWebRtcTransport(router)
		audioProducer := createAudioProducer(transport)
		audioConsumer := createConsumer(transport, audioProducer.Id())
		audioConsumer.OnClose(mymock.OnClose)
		transport.Close()
		assert.True(t, audioConsumer.Closed())
		time.Sleep(time.Millisecond)
	})

	t.Run("router closed", func(t *testing.T) {
		mymock := new(MockedHandler)
		defer mymock.AssertExpectations(t)

		mymock.On("OnClose", mock.IsType(context.Background())).Once()

		router := createRouter(nil)
		transport := createWebRtcTransport(router)
		audioProducer := createAudioProducer(transport)
		audioConsumer := createConsumer(transport, audioProducer.Id())
		audioConsumer.OnClose(mymock.OnClose)
		router.Close()
		assert.True(t, audioConsumer.Closed())
	})
}
