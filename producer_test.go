package mediasoup

import (
	"testing"

	"github.com/jiyeyuran/mediasoup-go/h264"

	"github.com/stretchr/testify/suite"
)

type ProducerTestingSuite struct {
	TestingSuite
	worker     *Worker
	router     *Router
	transport1 ITransport
	transport2 ITransport
}

func (suite *ProducerTestingSuite) SetupTest() {
	mediaCodecs := []*RtpCodecCapability{
		{
			Kind:      "audio",
			MimeType:  "audio/opus",
			ClockRate: 48000,
			Channels:  2,
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

	suite.worker = CreateTestWorker()
	suite.router, _ = suite.worker.CreateRouter(RouterOptions{
		MediaCodecs: mediaCodecs,
	})

	var err error

	suite.transport1, err = suite.router.CreateWebRtcTransport(WebRtcTransportOptions{
		ListenIps: []TransportListenIp{
			{Ip: "127.0.0.1"},
		},
	})
	suite.NoError(err)
	suite.transport2, err = suite.router.CreatePlainTransport(PlainTransportOptions{
		ListenIp: TransportListenIp{
			Ip: "127.0.0.1",
		},
	})
	suite.NoError(err)
}

func (suite *ProducerTestingSuite) TearDownTest() {
	suite.worker.Close()
}

func (suite *ProducerTestingSuite) TestWebRtcTransportProduce_Succeeds() {
	onObserverNewProducer := NewMockFunc(suite.T())
	suite.transport1.Observer().Once("newproducer", onObserverNewProducer.Fn())

	audioProducer := suite.audioProducer()
	onObserverNewProducer.ExpectCalledTimes(1)
	onObserverNewProducer.ExpectCalledWith(audioProducer)
	suite.NotEmpty(audioProducer.Id())
	suite.False(audioProducer.Closed())
	suite.EqualValues("audio", audioProducer.Kind())
	suite.NotEqual(RtpParameters{}, audioProducer.RtpParameters())
	suite.EqualValues("simple", audioProducer.Type())
	// Private API.
	suite.NotEmpty(audioProducer.ConsumableRtpParameters())
	suite.False(audioProducer.Paused())
	suite.Empty(audioProducer.Score())
	suite.Equal(H{"foo": 1, "bar": "2"}, audioProducer.AppData())

	routerDump, _ := suite.router.Dump()

	consumerIds, ok := routerDump.MapProducerIdConsumerIds[audioProducer.Id()]
	suite.True(ok)
	suite.Empty(consumerIds)
	suite.Empty(routerDump.MapConsumerIdProducerId)

	transportDump, _ := suite.transport1.Dump()

	suite.Equal(suite.transport1.Id(), transportDump.Id)
	suite.Equal([]string{audioProducer.Id()}, transportDump.ProducerIds)
	suite.Empty(transportDump.ConsumerIds)
}

func (suite *ProducerTestingSuite) TestPlainRtpTransportProduce_Succeeds() {
	onObserverNewProducer := NewMockFunc(suite.T())

	suite.transport2.Observer().Once("newproducer", onObserverNewProducer.Fn())

	videoProducer := suite.videoProducer()

	onObserverNewProducer.ExpectCalledTimes(1)
	onObserverNewProducer.ExpectCalledWith(videoProducer)
	suite.NotEmpty(videoProducer.Id())
	suite.False(videoProducer.Closed())
	suite.EqualValues("video", videoProducer.Kind())
	suite.NotEmpty(videoProducer.RtpParameters())
	suite.EqualValues("simulcast", videoProducer.Type())
	// Private API.
	suite.NotEmpty(videoProducer.ConsumableRtpParameters())
	suite.False(videoProducer.Paused())
	suite.Empty(videoProducer.Score())
	suite.Equal(H{"foo": 1, "bar": "2"}, videoProducer.AppData())

	routerDump, _ := suite.router.Dump()

	consumerIds, ok := routerDump.MapProducerIdConsumerIds[videoProducer.Id()]
	suite.True(ok)
	suite.Empty(consumerIds)
	suite.Empty(routerDump.MapConsumerIdProducerId)

	transportDump, _ := suite.transport2.Dump()

	suite.Equal(suite.transport2.Id(), transportDump.Id)
	suite.Equal([]string{videoProducer.Id()}, transportDump.ProducerIds)
	suite.Empty(transportDump.ConsumerIds)
}

func (suite *ProducerTestingSuite) TestWebRtcTransportProduce_TypeError() {
	transport1 := suite.transport1

	_, err := transport1.Produce(ProducerOptions{
		Kind: "chicken",
	})
	suite.IsType(NewTypeError(""), err)

	_, err = transport1.Produce(ProducerOptions{
		Kind: "audio",
	})
	suite.IsType(NewTypeError(""), err)

	// Missing or empty rtpParameters.codecs.
	_, err = transport1.Produce(ProducerOptions{
		Kind: "audio",
		RtpParameters: RtpParameters{
			Encodings: []RtpEncodingParameters{
				{Ssrc: 1111},
			},
			Rtcp: RtcpParameters{Cname: "qwerty"},
		},
	})
	suite.IsType(NewTypeError(""), err)

	// Missing or empty rtpParameters.encodings.
	_, err = transport1.Produce(ProducerOptions{
		Kind: "video",
		RtpParameters: RtpParameters{
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "video/h264",
					PayloadType: 112,
					ClockRate:   90000,
					Parameters: RtpCodecSpecificParameters{
						RtpParameter: h264.RtpParameter{
							PacketizationMode: 1,
							ProfileLevelId:    "4d0032",
						},
					},
				},
				{
					MimeType:    "video/rtx",
					PayloadType: 113,
					ClockRate:   90000,
					Parameters:  RtpCodecSpecificParameters{Apt: 112},
				},
			},
			Rtcp: RtcpParameters{
				Cname: "qwerty",
			},
		},
	})
	suite.IsType(NewTypeError(""), err)

	// Wrong apt in RTX codec.
	_, err = transport1.Produce(ProducerOptions{
		Kind: "video",
		RtpParameters: RtpParameters{
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "video/h264",
					PayloadType: 112,
					ClockRate:   90000,
					Parameters: RtpCodecSpecificParameters{
						RtpParameter: h264.RtpParameter{
							PacketizationMode: 1,
							ProfileLevelId:    "4d0032",
						},
					},
				},
				{
					MimeType:    "video/rtx",
					PayloadType: 113,
					ClockRate:   90000,
					Parameters:  RtpCodecSpecificParameters{Apt: 111},
				},
			},
			Encodings: []RtpEncodingParameters{
				{Ssrc: 6666, Rtx: &RtpEncodingRtx{Ssrc: 6667}},
			},
			Rtcp: RtcpParameters{
				Cname: "video-1",
			},
		},
	})
	suite.IsType(NewTypeError(""), err)
}

func (suite *ProducerTestingSuite) TestWebRtcTransportProduce_UnsupportedError() {
	transport1 := suite.transport1

	_, err := transport1.Produce(ProducerOptions{
		Kind: "audio",
		RtpParameters: RtpParameters{
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "audio/ISAC",
					PayloadType: 108,
					ClockRate:   32000,
				},
			},
			Encodings: []RtpEncodingParameters{
				{Ssrc: 1111},
			},
			Rtcp: RtcpParameters{Cname: "audio"},
		},
	})
	suite.IsType(NewUnsupportedError(""), err)

	_, err = transport1.Produce(ProducerOptions{
		Kind: "video",
		RtpParameters: RtpParameters{
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "video/h264",
					PayloadType: 112,
					ClockRate:   90000,
					Parameters: RtpCodecSpecificParameters{
						RtpParameter: h264.RtpParameter{
							PacketizationMode: 1,
							ProfileLevelId:    "CHICKEN",
						},
					},
				},
				{
					MimeType:    "video/rtx",
					PayloadType: 113,
					ClockRate:   90000,
					Parameters:  RtpCodecSpecificParameters{Apt: 112},
				},
			},
			Encodings: []RtpEncodingParameters{
				{Ssrc: 6666, Rtx: &RtpEncodingRtx{Ssrc: 6667}},
			},
		},
	})
	suite.IsType(NewUnsupportedError(""), err)
}

func (suite *ProducerTestingSuite) TestWebRtcTransportProduce_WithAlreadyUsedMIDOrSSRCError() {
	suite.audioProducer()

	// Mid already used
	_, err := suite.transport1.Produce(ProducerOptions{
		Kind: "audio",
		RtpParameters: RtpParameters{
			Mid: "AUDIO",
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "audio/opus",
					PayloadType: 111,
					ClockRate:   48000,
					Channels:    2,
				},
			},
			Encodings: []RtpEncodingParameters{
				{Ssrc: 33333333},
			},
			Rtcp: RtcpParameters{Cname: "audio-2"},
		},
	})
	suite.Error(err)

	suite.videoProducer()

	// Ssrc already used
	_, err = suite.transport2.Produce(ProducerOptions{
		Kind: "video",
		RtpParameters: RtpParameters{
			Mid: "VIDEO2",
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "video/h264",
					PayloadType: 112,
					ClockRate:   90000,
					Parameters: RtpCodecSpecificParameters{
						RtpParameter: h264.RtpParameter{
							PacketizationMode: 1,
							ProfileLevelId:    "4d0032",
						},
					},
				},
			},
			HeaderExtensions: []RtpHeaderExtensionParameters{
				{
					Uri: "urn:ietf:params:rtp-hdrext:sdes:mid",
					Id:  10,
				},
			},
			Encodings: []RtpEncodingParameters{
				{Ssrc: 22222222},
			},
		},
	})
	suite.Error(err)
}

func (suite *ProducerTestingSuite) TestProduerDump_Succeeds() {
	audioProducer := suite.audioProducer()

	data, _ := audioProducer.Dump()

	suite.Equal(audioProducer.Id(), data.Id)
	suite.EqualValues(audioProducer.Kind(), data.Kind)
	suite.Len(data.RtpParameters.Codecs, 1)
	suite.Equal("audio/opus", data.RtpParameters.Codecs[0].MimeType)
	suite.EqualValues(111, data.RtpParameters.Codecs[0].PayloadType)
	suite.EqualValues(48000, data.RtpParameters.Codecs[0].ClockRate)
	suite.EqualValues(2, data.RtpParameters.Codecs[0].Channels)
	suite.Equal(RtpCodecSpecificParameters{
		Useinbandfec: 1,
		Usedtx:       1,
	}, data.RtpParameters.Codecs[0].Parameters)
	suite.Empty(data.RtpParameters.Codecs[0].RtcpFeedback)
	suite.Len(data.RtpParameters.HeaderExtensions, 2)
	suite.EqualValues([]RtpHeaderExtensionParameters{
		{
			Uri:        "urn:ietf:params:rtp-hdrext:sdes:mid",
			Id:         10,
			Parameters: &RtpCodecSpecificParameters{},
			Encrypt:    false,
		},
		{
			Uri:        "urn:ietf:params:rtp-hdrext:ssrc-audio-level",
			Id:         12,
			Parameters: &RtpCodecSpecificParameters{},
			Encrypt:    false,
		},
	}, data.RtpParameters.HeaderExtensions)
	suite.Len(data.RtpParameters.Encodings, 1)
	suite.EqualValues([]RtpEncodingParameters{
		{CodecPayloadType: 111, Ssrc: 11111111, Dtx: true},
	}, data.RtpParameters.Encodings)
	suite.EqualValues("simple", data.Type)

	videoProducer := suite.videoProducer()
	data, _ = videoProducer.Dump()

	suite.Equal(videoProducer.Id(), data.Id)
	suite.EqualValues(videoProducer.Kind(), data.Kind)
	suite.Len(data.RtpParameters.Codecs, 2)
	suite.Equal("video/H264", data.RtpParameters.Codecs[0].MimeType)
	suite.EqualValues(112, data.RtpParameters.Codecs[0].PayloadType)
	suite.EqualValues(90000, data.RtpParameters.Codecs[0].ClockRate)
	suite.Empty(data.RtpParameters.Codecs[0].Channels)
	suite.Equal(RtpCodecSpecificParameters{
		RtpParameter: h264.RtpParameter{
			PacketizationMode: 1,
			ProfileLevelId:    "4d0032",
		},
	}, data.RtpParameters.Codecs[0].Parameters)
	suite.EqualValues([]RtcpFeedback{
		{Type: "nack"},
		{Type: "nack", Parameter: "pli"},
		{Type: "goog-remb"},
	}, data.RtpParameters.Codecs[0].RtcpFeedback)

	suite.Equal("video/rtx", data.RtpParameters.Codecs[1].MimeType)
	suite.EqualValues(113, data.RtpParameters.Codecs[1].PayloadType)
	suite.EqualValues(90000, data.RtpParameters.Codecs[1].ClockRate)
	suite.Empty(data.RtpParameters.Codecs[1].Channels)
	suite.Equal(RtpCodecSpecificParameters{
		Apt: 112,
	}, data.RtpParameters.Codecs[1].Parameters)
	suite.Empty(data.RtpParameters.Codecs[1].RtcpFeedback)
	suite.Len(data.RtpParameters.HeaderExtensions, 2)
	suite.EqualValues([]RtpHeaderExtensionParameters{
		{
			Uri:        "urn:ietf:params:rtp-hdrext:sdes:mid",
			Id:         10,
			Parameters: &RtpCodecSpecificParameters{},
			Encrypt:    false,
		},
		{
			Uri:        "urn:3gpp:video-orientation",
			Id:         13,
			Parameters: &RtpCodecSpecificParameters{},
			Encrypt:    false,
		},
	}, data.RtpParameters.HeaderExtensions)
	suite.Len(data.RtpParameters.Encodings, 4)
	suite.EqualValues([]RtpEncodingParameters{
		{CodecPayloadType: 112, Ssrc: 22222222, Rtx: &RtpEncodingRtx{Ssrc: 22222223}},
		{CodecPayloadType: 112, Ssrc: 22222224, Rtx: &RtpEncodingRtx{Ssrc: 22222225}},
		{CodecPayloadType: 112, Ssrc: 22222226, Rtx: &RtpEncodingRtx{Ssrc: 22222227}},
		{CodecPayloadType: 112, Ssrc: 22222228, Rtx: &RtpEncodingRtx{Ssrc: 22222229}},
	}, data.RtpParameters.Encodings)
	suite.EqualValues("simulcast", data.Type)
}

func (suite *ProducerTestingSuite) TestGetStats_Succeeds() {
	audioProducer := suite.audioProducer()
	stats, _ := audioProducer.GetStats()
	suite.Empty(stats)

	videoProducer := suite.videoProducer()
	stats, _ = videoProducer.GetStats()
	suite.Empty(stats)
}

func (suite *ProducerTestingSuite) TestProducerPauseAndResume_Succeeds() {
	audioProducer := suite.audioProducer()
	audioProducer.Pause()
	suite.True(audioProducer.Paused())

	data, _ := audioProducer.Dump()
	suite.True(data.Paused)

	audioProducer.Resume()
	suite.False(audioProducer.Paused())

	data, _ = audioProducer.Dump()
	suite.False(data.Paused)
}

func (suite *ProducerTestingSuite) TestProducerEnableTraceEventSucceed() {
	audioProducer := suite.audioProducer()
	audioProducer.EnableTraceEvent("rtp", "pli")
	data, _ := audioProducer.Dump()
	suite.EqualValues("rtp,pli", data.TraceEventTypes)

	audioProducer.EnableTraceEvent()
	data, _ = audioProducer.Dump()
	suite.Zero(data.TraceEventTypes)

	audioProducer.EnableTraceEvent("nack", "FOO", "fir")
	data, _ = audioProducer.Dump()
	suite.EqualValues("nack,fir", data.TraceEventTypes)

	audioProducer.EnableTraceEvent()
	data, _ = audioProducer.Dump()
	suite.Zero(data.TraceEventTypes)
}

func (suite *ProducerTestingSuite) TestProducerEmitsScore() {
	videoProducer := suite.videoProducer()
	channel := videoProducer.channel

	onScore := NewMockFunc(suite.T())

	videoProducer.On("score", onScore.Fn())

	channel.emit(videoProducer.Id(), "score",
		[]byte(`[ { "ssrc": 11, "score": 10 } ]`))
	channel.emit(videoProducer.Id(), "score",
		[]byte(`[ { "ssrc": 11, "score": 9 }, { "ssrc": 22, "score": 8 } ]`))
	channel.emit(videoProducer.Id(), "score",
		[]byte(`[ { "ssrc": 11, "score": 9 }, { "ssrc": 22, "score": 9 } ]`))

	suite.Equal(3, onScore.CalledTimes())
	suite.Equal([]ProducerScore{
		{Ssrc: 11, Score: 9},
		{Ssrc: 22, Score: 9},
	}, videoProducer.Score())
}

func (suite *ProducerTestingSuite) TestProduceClose_Succeeds() {
	onObserverClose := NewMockFunc(suite.T())

	audioProducer := suite.audioProducer()
	audioProducer.Observer().Once("close", onObserverClose.Fn())
	audioProducer.Close()

	suite.Equal(1, onObserverClose.CalledTimes())
	suite.True(audioProducer.Closed())

	routerDump, _ := suite.router.Dump()

	suite.Empty(routerDump.MapProducerIdConsumerIds)
	suite.Empty(routerDump.MapConsumerIdProducerId)

	transportDump, _ := suite.transport1.Dump()

	suite.Equal(suite.transport1.Id(), transportDump.Id)
	suite.Empty(transportDump.ProducerIds)
	suite.Empty(transportDump.ConsumerIds)
}

func (suite *ProducerTestingSuite) TestProduceMethodsRejectIfClosed() {
	audioProducer := suite.audioProducer()
	audioProducer.Close()

	_, err := audioProducer.Dump()
	suite.Error(err)

	_, err = audioProducer.GetStats()
	suite.Error(err)

	suite.Error(audioProducer.Pause())
	suite.Error(audioProducer.Resume())
}

func (suite *ProducerTestingSuite) TestProducerEmitsTransportclose() {
	onObserverClose := NewMockFunc(suite.T())

	videoProducer := suite.videoProducer()
	videoProducer.Observer().Once("close", onObserverClose.Fn())

	wf := NewMockFunc(suite.T())

	videoProducer.On("transportclose", wf.Fn())
	suite.transport2.Close()

	wf.ExpectCalled()
	onObserverClose.ExpectCalled()
	suite.True(videoProducer.Closed())
}

func (suite *ProducerTestingSuite) audioProducer() *Producer {
	return CreateAudioProducer(suite.transport1)
}

func (suite *ProducerTestingSuite) videoProducer() *Producer {
	return CreateH264Producer(suite.transport2)
}

func TestProducerTestingSuite(t *testing.T) {
	suite.Run(t, new(ProducerTestingSuite))
}
