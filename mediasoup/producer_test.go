package mediasoup

import (
	"encoding/json"
	"testing"

	"github.com/jiyeyuran/mediasoup-go/mediasoup/h264profile"

	"github.com/stretchr/testify/suite"
)

type ProducerTestSuite struct {
	suite.Suite
	worker            *Worker
	router            *Router
	webRtcTransport   Transport
	plainRtpTransport Transport
}

func (suite *ProducerTestSuite) SetupTest() {
	const mediaCodecsJSON = `
	[
		{
		  "kind" : "audio",
		  "mimeType" : "audio/opus",
		  "clockRate" : 48000,
		  "channels" : 2,
		  "parameters" : { "foo":"111" }
		},
		{
		  "kind" : "video",
		  "mimeType" : "video/VP8",
		  "clockRate" : 90000
		},
		{
		  "kind" : "video",
		  "mimeType" : "video/H264",
		  "clockRate" : 90000,
		  "rtcpFeedback" : [],
		  "parameters" : {
			"level-asymmetry-allowed" : 1,
			"packetization-mode" : 1,
			"profile-level-id" : "4d0032",
			"foo" : "bar"
		  }
		}
	  ]
`

	var mediaCodecs []RtpCodecCapability
	err := json.Unmarshal([]byte(mediaCodecsJSON), &mediaCodecs)
	if err != nil {
		panic(err)
	}

	suite.worker = CreateTestWorker()
	suite.router, _ = suite.worker.CreateRouter(mediaCodecs)

	suite.webRtcTransport, _ = suite.router.CreateWebRtcTransport(
		CreateWebRtcTransportParams{
			ListenIps: []ListenIp{
				{Ip: "127.0.0.1"},
			},
		},
	)
	suite.plainRtpTransport, _ = suite.router.CreatePlainRtpTransport(
		CreatePlainRtpTransportParams{
			ListenIp: ListenIp{Ip: "127.0.0.1"},
		},
	)
}

func (suite *ProducerTestSuite) TearDownTest() {
	suite.worker.Close()
}

func (suite *ProducerTestSuite) TestWebRtcTransportProduce_Succeeds() {
	onObserverNewProducer := NewMockFunc(suite.T())
	suite.webRtcTransport.Observer().Once("newproducer", onObserverNewProducer.Fn())

	audioProducer := suite.audioProducer()
	onObserverNewProducer.ExpectCalledTimes(1)
	onObserverNewProducer.ExpectCalledWith(audioProducer)
	suite.NotEmpty(audioProducer.Id())
	suite.False(audioProducer.Closed())
	suite.Equal("audio", audioProducer.Kind())
	suite.NotEmpty(audioProducer.RtpParameters())
	suite.Equal("simple", audioProducer.Type())
	// Private API.
	suite.NotEmpty(audioProducer.ConsumableRtpParameters())
	suite.False(audioProducer.Paused())
	suite.Empty(audioProducer.Score())
	assertJSONEq(suite.T(), H{"foo": 1, "bar": "2"}, audioProducer.AppData())

	var routerDump struct {
		MapProducerIdConsumerIds map[string][]string
		MapConsumerIdProducerId  map[string]string
	}
	suite.router.Dump().Unmarshal(&routerDump)

	consumerIds, ok := routerDump.MapProducerIdConsumerIds[audioProducer.Id()]
	suite.True(ok)
	suite.Empty(consumerIds)
	suite.Empty(routerDump.MapConsumerIdProducerId)

	var transportDump struct {
		Id          string
		ProducerIds []string
		ConsumerIds []string
	}
	suite.webRtcTransport.Dump().Unmarshal(&transportDump)

	suite.Equal(suite.webRtcTransport.Id(), transportDump.Id)
	suite.Equal([]string{audioProducer.Id()}, transportDump.ProducerIds)
	suite.Empty(transportDump.ConsumerIds)
}

func (suite *ProducerTestSuite) TestPlainRtpTransportProduce_Succeeds() {
	onObserverNewProducer := NewMockFunc(suite.T())

	suite.plainRtpTransport.Observer().Once("newproducer", onObserverNewProducer.Fn())

	videoProducer := suite.videoProducer()

	onObserverNewProducer.ExpectCalledTimes(1)
	onObserverNewProducer.ExpectCalledWith(videoProducer)
	suite.NotEmpty(videoProducer.Id())
	suite.False(videoProducer.Closed())
	suite.Equal("video", videoProducer.Kind())
	suite.NotEmpty(videoProducer.RtpParameters())
	suite.Equal("simulcast", videoProducer.Type())
	// Private API.
	suite.NotEmpty(videoProducer.ConsumableRtpParameters())
	suite.False(videoProducer.Paused())
	suite.Empty(videoProducer.Score())
	assertJSONEq(suite.T(), H{"foo": 1, "bar": "2"}, videoProducer.AppData())

	var routerDump struct {
		MapProducerIdConsumerIds map[string][]string
		MapConsumerIdProducerId  map[string]string
	}
	suite.router.Dump().Unmarshal(&routerDump)

	consumerIds, ok := routerDump.MapProducerIdConsumerIds[videoProducer.Id()]
	suite.True(ok)
	suite.Empty(consumerIds)
	suite.Empty(routerDump.MapConsumerIdProducerId)

	var transportDump struct {
		Id          string
		ProducerIds []string
		ConsumerIds []string
	}
	suite.plainRtpTransport.Dump().Unmarshal(&transportDump)

	suite.Equal(suite.plainRtpTransport.Id(), transportDump.Id)
	suite.Equal([]string{videoProducer.Id()}, transportDump.ProducerIds)
	suite.Empty(transportDump.ConsumerIds)
}

func (suite *ProducerTestSuite) TestWebRtcTransportProduce_TypeError() {
	webRtcTransport := suite.webRtcTransport

	_, err := webRtcTransport.Produce(transportProduceParams{
		Kind: "chicken",
	})
	suite.IsType(NewTypeError(""), err)

	_, err = webRtcTransport.Produce(transportProduceParams{
		Kind: "audio",
	})
	suite.IsType(NewTypeError(""), err)

	// Missing or empty rtpParameters.codecs.
	_, err = webRtcTransport.Produce(transportProduceParams{
		Kind: "audio",
		RtpParameters: RtpParameters{
			Encodings: []RtpEncoding{
				{Ssrc: 1111},
			},
			Rtcp: RtcpConfiguation{Cname: "qwerty"},
		},
	})
	suite.IsType(NewTypeError(""), err)

	// Missing or empty rtpParameters.encodings.
	produceParamsJSON := `
	{
		"kind" : "video",
		"rtpParameters" : {
		  "codecs" : [
			{
			  "mimeType" : "video/h264",
			  "payloadType" : 112,
			  "clockRate" : 90000,
			  "parameters" : {
				"packetization-mode" : 1,
				"profile-level-id" : "4d0032"
			  }
			},
			{
			  "mimeType" : "video/rtx",
			  "payloadType" : 113,
			  "clockRate" : 90000,
			  "parameters" : { "apt":112 }
			}
		  ],
		  "headerExtensions" : [],
		  "encodings" : [],
		  "rtcp" : { "cname":"qwerty" }
		}
	  }
	`
	var produceParams transportProduceParams
	json.Unmarshal([]byte(produceParamsJSON), &produceParams)
	_, err = webRtcTransport.Produce(produceParams)
	suite.IsType(NewTypeError(""), err)

	// Wrong apt in RTX codec.
	produceParamsJSON = `
	  {
		"kind"          : "audio",
		"rtpParameters" : {
		  "codecs"           : [
			{
			  "mimeType"    : "video/h264",
			  "payloadType" : 112,
			  "clockRate"   : 90000,
			  "parameters"  : {
				"packetization-mode" : 1,
				"profile-level-id"   : "4d0032"
			  }
			},
			{
			  "mimeType"    : "video/rtx",
			  "payloadType" : 113,
			  "clockRate"   : 90000,
			  "parameters"  : { "apt":111 }
			}
		  ],
		  "headerExtensions" : [],
		  "encodings"        : [
			{
			  "ssrc" : 6666,
			  "rtx"  : { "ssrc":6667 }
			}
		  ],
		  "rtcp"             : { "cname":"video-1" }
		}
	  }
	  `
	json.Unmarshal([]byte(produceParamsJSON), &produceParams)
	_, err = webRtcTransport.Produce(produceParams)
	suite.IsType(NewTypeError(""), err)
}

func (suite *ProducerTestSuite) TestWebRtcTransportProduce_UnsupportedError() {
	webRtcTransport := suite.webRtcTransport

	produceParamsJSON := `
{
	"kind"          : "audio",
	"rtpParameters" : {
	  "codecs"           : [
		{
		  "mimeType"    : "audio/ISAC",
		  "payloadType" : 108,
		  "clockRate"   : 32000
		}
	  ],
	  "headerExtensions" : [],
	  "encodings"        : [ { "ssrc":1111 } ],
	  "rtcp"             : { "cname":"audio" }
	}
  }
`
	var produceParams transportProduceParams
	json.Unmarshal([]byte(produceParamsJSON), &produceParams)
	_, err := webRtcTransport.Produce(produceParams)
	suite.IsType(NewUnsupportedError(""), err)

	produceParamsJSON = `
	{
		"kind"          : "video",
		"rtpParameters" : {
		  "codecs"           : [
			{
			  "mimeType"    : "video/h264",
			  "payloadType" : 112,
			  "clockRate"   : 90000,
			  "parameters"  : {
				"packetization-mode" : 1,
				"profile-level-id"   : "CHICKEN"
			  }
			},
			{
			  "mimeType"    : "video/rtx",
			  "payloadType" : 113,
			  "clockRate"   : 90000,
			  "parameters"  : { "apt":112 }
			}
		  ],
		  "headerExtensions" : [],
		  "encodings"        : [
			{
			  "ssrc" : 6666,
			  "rtx"  : { "ssrc":6667 }
			}
		  ]
		}
	  }
`
	json.Unmarshal([]byte(produceParamsJSON), &produceParams)
	_, err = webRtcTransport.Produce(produceParams)
	suite.IsType(NewUnsupportedError(""), err)
}

func (suite *ProducerTestSuite) TestWebRtcTransportProduce_WithAlreadyUsedMIDOrSSRCError() {
	webRtcTransport := suite.webRtcTransport

	produceParamsJSON := `
	{
		"kind"          : "audio",
		"rtpParameters" : {
		  "mid"              : "AUDIO",
		  "codecs"           : [
			{
			  "mimeType"    : "audio/opus",
			  "payloadType" : 111,
			  "clockRate"   : 48000,
			  "channels"    : 2
			}
		  ],
		  "headerExtensions" : [],
		  "encodings"        : [ { "ssrc":33333333 } ],
		  "rtcp"             : { "cname":"audio-2" }
		}
	  }
`
	var produceParams transportProduceParams
	json.Unmarshal([]byte(produceParamsJSON), &produceParams)
	_, err := webRtcTransport.Produce(produceParams)
	suite.Error(err)

	produceParamsJSON = `
	{
		"kind"          : "audio",
		"rtpParameters" : {
		  "mid"              : "AUDIO-2",
		  "codecs"           : [
			{
			  "mimeType"    : "audio/opus",
			  "payloadType" : 111,
			  "clockRate"   : 48000,
			  "channels"    : 2
			}
		  ],
		  "headerExtensions" : [],
		  "encodings"        : [ { "ssrc":11111111 } ],
		  "rtcp"             : { "cname":"audio-2" }
		}
	  }
`
	json.Unmarshal([]byte(produceParamsJSON), &produceParams)
	_, err = webRtcTransport.Produce(produceParams)
	suite.Error(err)
}

func (suite *ProducerTestSuite) TestProduerDump_Succeeds() {
	audioProducer := suite.audioProducer()

	type Dump struct {
		Id            string
		Kind          string
		Type          string
		RtpParameters RtpParameters
	}
	var data Dump
	audioProducer.Dump().Unmarshal(&data)

	suite.Equal(audioProducer.Id(), data.Id)
	suite.Equal(audioProducer.Kind(), data.Kind)
	suite.Len(data.RtpParameters.Codecs, 1)
	suite.Equal("audio/opus", data.RtpParameters.Codecs[0].MimeType)
	suite.EqualValues(111, data.RtpParameters.Codecs[0].PayloadType)
	suite.EqualValues(48000, data.RtpParameters.Codecs[0].ClockRate)
	suite.EqualValues(2, data.RtpParameters.Codecs[0].Channels)
	suite.Equal(&RtpCodecParameter{
		Useinbandfec: 1,
		Usedtx:       1,
	}, data.RtpParameters.Codecs[0].Parameters)
	suite.Empty(data.RtpParameters.Codecs[0].RtcpFeedback)
	suite.Len(data.RtpParameters.HeaderExtensions, 2)
	suite.EqualValues([]RtpHeaderExtension{
		{
			Uri:        "urn:ietf:params:rtp-hdrext:sdes:mid",
			Id:         10,
			Parameters: &H{},
			Encrypt:    newBool(false),
		},
		{
			Uri:        "urn:ietf:params:rtp-hdrext:ssrc-audio-level",
			Id:         12,
			Parameters: &H{},
			Encrypt:    newBool(false),
		},
	}, data.RtpParameters.HeaderExtensions)
	suite.Len(data.RtpParameters.Encodings, 1)
	suite.EqualValues([]RtpEncoding{
		{CodecPayloadType: 111, Ssrc: 11111111, Dtx: true},
	}, data.RtpParameters.Encodings)
	suite.Equal("simple", data.Type)

	data = Dump{}
	videoProducer := suite.videoProducer()
	videoProducer.Dump().Unmarshal(&data)

	suite.Equal(videoProducer.Id(), data.Id)
	suite.Equal(videoProducer.Kind(), data.Kind)
	suite.Len(data.RtpParameters.Codecs, 2)
	suite.Equal("video/H264", data.RtpParameters.Codecs[0].MimeType)
	suite.EqualValues(112, data.RtpParameters.Codecs[0].PayloadType)
	suite.EqualValues(90000, data.RtpParameters.Codecs[0].ClockRate)
	suite.Empty(data.RtpParameters.Codecs[0].Channels)
	suite.Equal(&RtpCodecParameter{
		RtpH264Parameter: h264profile.RtpH264Parameter{
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
	suite.Equal(&RtpCodecParameter{
		Apt: 112,
	}, data.RtpParameters.Codecs[1].Parameters)
	suite.Empty(data.RtpParameters.Codecs[1].RtcpFeedback)
	suite.Len(data.RtpParameters.HeaderExtensions, 2)
	suite.EqualValues([]RtpHeaderExtension{
		{
			Uri:        "urn:ietf:params:rtp-hdrext:sdes:mid",
			Id:         10,
			Parameters: &H{},
			Encrypt:    newBool(false),
		},
		{
			Uri:        "urn:3gpp:video-orientation",
			Id:         13,
			Parameters: &H{},
			Encrypt:    newBool(false),
		},
	}, data.RtpParameters.HeaderExtensions)
	suite.Len(data.RtpParameters.Encodings, 4)
	suite.EqualValues([]RtpEncoding{
		{CodecPayloadType: 112, Ssrc: 22222222, Rtx: &RtpEncoding{Ssrc: 22222223}},
		{CodecPayloadType: 112, Ssrc: 22222224, Rtx: &RtpEncoding{Ssrc: 22222225}},
		{CodecPayloadType: 112, Ssrc: 22222226, Rtx: &RtpEncoding{Ssrc: 22222227}},
		{CodecPayloadType: 112, Ssrc: 22222228, Rtx: &RtpEncoding{Ssrc: 22222229}},
	}, data.RtpParameters.Encodings)
	suite.Equal("simulcast", data.Type)
}

func (suite *ProducerTestSuite) TestGetStats_Succeeds() {
	audioProducer := suite.audioProducer()

	suite.Equal([]byte("[]"), audioProducer.GetStats().Data())

	videoProducer := suite.videoProducer()

	suite.Equal([]byte("[]"), videoProducer.GetStats().Data())
}

func (suite *ProducerTestSuite) TestProducerPauseAndResume_Succeeds() {
	audioProducer := suite.audioProducer()
	audioProducer.Pause()

	suite.True(audioProducer.Paused())

	type Dump struct {
		Paused bool
	}
	data := Dump{}
	audioProducer.Dump().Unmarshal(&data)

	suite.True(data.Paused)

	audioProducer.Resume()

	suite.False(audioProducer.Paused())

	data = Dump{}
	audioProducer.Dump().Unmarshal(&data)

	suite.False(data.Paused)
}

func (suite *ProducerTestSuite) TestProducerEmitsScore() {
	videoProducer := suite.videoProducer()
	channel := videoProducer.channel

	onScore := NewMockFunc(suite.T())

	videoProducer.On("score", onScore.Fn())

	channel.Emit(videoProducer.Id(), "score",
		json.RawMessage(`[ { "ssrc": 11, "score": 10 } ]`))
	channel.Emit(videoProducer.Id(), "score",
		json.RawMessage(`[ { "ssrc": 11, "score": 9 }, { "ssrc": 22, "score": 8 } ]`))
	channel.Emit(videoProducer.Id(), "score",
		json.RawMessage(`[ { "ssrc": 11, "score": 9 }, { "ssrc": 22, "score": 9 } ]`))

	suite.Equal(3, onScore.CalledTimes())
	suite.Equal([]ProducerScore{
		{Ssrc: 11, Score: 9},
		{Ssrc: 22, Score: 9},
	}, videoProducer.Score())
}

func (suite *ProducerTestSuite) TestProduceClose_Succeeds() {
	onObserverClose := NewMockFunc(suite.T())

	audioProducer := suite.audioProducer()
	audioProducer.Observer().Once("close", onObserverClose.Fn())
	audioProducer.Close()

	suite.Equal(1, onObserverClose.CalledTimes())
	suite.True(audioProducer.Closed())

	var routerDump struct {
		MapProducerIdConsumerIds map[string][]string
		MapConsumerIdProducerId  map[string]string
	}
	suite.router.Dump().Unmarshal(&routerDump)

	suite.Empty(routerDump.MapProducerIdConsumerIds)
	suite.Empty(routerDump.MapConsumerIdProducerId)

	var transportDump struct {
		Id          string
		ProducerIds []string
		ConsumerIds []string
	}
	suite.webRtcTransport.Dump().Unmarshal(&transportDump)

	suite.Equal(suite.webRtcTransport.Id(), transportDump.Id)
	suite.Empty(transportDump.ProducerIds)
	suite.Empty(transportDump.ConsumerIds)
}

func (suite *ProducerTestSuite) TestProduceMethodsRejectIfClosed() {
	audioProducer := suite.audioProducer()
	audioProducer.Close()

	suite.Error(audioProducer.Dump().Err())
	suite.Error(audioProducer.GetStats().Err())
	suite.Error(audioProducer.Pause())
	suite.Error(audioProducer.Resume())
}

func (suite *ProducerTestSuite) TestProducerEmitsTransportclose() {
	onObserverClose := NewMockFunc(suite.T())

	videoProducer := suite.videoProducer()
	videoProducer.Observer().Once("close", onObserverClose.Fn())

	wf := NewMockFunc(suite.T())

	videoProducer.On("transportclose", wf.Fn())
	suite.plainRtpTransport.Close()

	wf.Wait()

	suite.Equal(1, onObserverClose.CalledTimes())
	suite.True(videoProducer.Closed())
}

func (suite *ProducerTestSuite) audioProducer() *Producer {
	transportProduceParamsJSON := `
	{
		"kind" : "audio",
		"rtpParameters" : {
		  "mid" : "AUDIO",
		  "codecs" : [
			{
			  "mimeType" : "audio/opus",
			  "payloadType" : 111,
			  "clockRate" : 48000,
			  "channels" : 2,
			  "parameters" : {
				"useinbandfec" : 1,
				"usedtx" : 1,
				"foo" : 222.222,
				"bar" : "333"
			  }
			}
		  ],
		  "headerExtensions" : [
			{
			  "uri" : "urn:ietf:params:rtp-hdrext:sdes:mid",
			  "id" : 10
			},
			{
			  "uri" : "urn:ietf:params:rtp-hdrext:ssrc-audio-level",
			  "id" : 12
			}
		  ],
		  "encodings" : [ { "ssrc":11111111, "dtx":true } ],
		  "rtcp" : { "cname":"audio-1" }
		},
		"appData" : { "foo":1, "bar":"2" }
	  }
	`
	var transportProduceParams transportProduceParams
	json.Unmarshal([]byte(transportProduceParamsJSON), &transportProduceParams)

	audioProducer, err := suite.webRtcTransport.Produce(transportProduceParams)
	suite.NoError(err)

	return audioProducer
}

func (suite *ProducerTestSuite) videoProducer() *Producer {
	transportProduceParamsJSON := `
	{
		"kind" : "video",
		"rtpParameters" : {
		  "mid" : "VIDEO",
		  "codecs" : [
			{
			  "mimeType" : "video/h264",
			  "payloadType" : 112,
			  "clockRate" : 90000,
			  "parameters" : {
				"packetization-mode" : 1,
				"profile-level-id" : "4d0032"
			  },
			  "rtcpFeedback" : [
				{ "type":"nack" },
				{
				  "type" : "nack",
				  "parameter" : "pli"
				},
				{ "type":"goog-remb" }
			  ]
			},
			{
			  "mimeType" : "video/rtx",
			  "payloadType" : 113,
			  "clockRate" : 90000,
			  "parameters" : { "apt":112 }
			}
		  ],
		  "headerExtensions" : [
			{
			  "uri" : "urn:ietf:params:rtp-hdrext:sdes:mid",
			  "id" : 10
			},
			{
			  "uri" : "urn:3gpp:video-orientation",
			  "id" : 13
			}
		  ],
		  "encodings" : [
			{
			  "ssrc" : 22222222,
			  "rtx" : { "ssrc":22222223 }
			},
			{
			  "ssrc" : 22222224,
			  "rtx" : { "ssrc":22222225 }
			},
			{
			  "ssrc" : 22222226,
			  "rtx" : { "ssrc":22222227 }
			},
			{
			  "ssrc" : 22222228,
			  "rtx" : { "ssrc":22222229 }
			}
		  ],
		  "rtcp" : { "cname":"video-1" }
		},
		"appData" : { "foo":1, "bar":"2" }
	  }
	`
	var transportProduceParams transportProduceParams
	json.Unmarshal([]byte(transportProduceParamsJSON), &transportProduceParams)

	producer, err := suite.plainRtpTransport.Produce(transportProduceParams)
	suite.NoError(err)

	return producer
}

func TestProducerTestSuite(t *testing.T) {
	suite.Run(t, new(ProducerTestSuite))
}
