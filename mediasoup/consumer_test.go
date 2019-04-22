package mediasoup

import (
	"encoding/json"
	"testing"

	"github.com/jiyeyuran/mediasoup-go/mediasoup/h264profile"

	"github.com/stretchr/testify/suite"
)

type ConsumerTestSuite struct {
	suite.Suite
	consumerDeviceCapabilities RtpCapabilities
	worker                     *Worker
	router                     *Router
	transport1                 Transport
	transport2                 Transport
	audioProducer              *Producer
	videoProducer              *Producer
}

func (suite *ConsumerTestSuite) SetupTest() {
	const (
		mediaCodecsJSON = `
[
  {
    "kind" : "audio",
    "mimeType" : "audio/opus",
    "clockRate" : 48000,
    "channels" : 2,
    "parameters" : { "foo":"bar" }
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
    "parameters" : {
      "level-asymmetry-allowed" : 1,
      "packetization-mode" : 1,
      "profile-level-id" : "4d0032",
      "foo" : "bar"
    }
  }
]
`
		audioProducerParametersJSON = `
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
	  "encodings" : [ { "ssrc":11111111 } ],
	  "rtcp" : { "cname":"FOOBAR" }
	},
	"appData" : { "foo":1, "bar":"2" }
  }
`

		videoProducerParametersJSON = `
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
	  "rtcp" : { "cname":"FOOBAR" }
	},
	"appData" : { "foo":1, "bar":"2" }
  }
`

		consumerDeviceCapabilitiesJSON = `
{
	"codecs" : [
	  {
		"mimeType" : "audio/opus",
		"kind" : "audio",
		"clockRate" : 48000,
		"preferredPayloadType" : 100,
		"channels" : 2
	  },
	  {
		"mimeType" : "video/H264",
		"kind" : "video",
		"clockRate" : 90000,
		"preferredPayloadType" : 101,
		"rtcpFeedback" : [
		  { "type":"nack" },
		  {
			"type" : "nack",
			"parameter" : "pli"
		  },
		  {
			"type" : "ccm",
			"parameter" : "fir"
		  },
		  { "type":"goog-remb" }
		],
		"parameters" : {
		  "level-asymmetry-allowed" : 1,
		  "packetization-mode" : 1,
		  "profile-level-id" : "4d0032"
		}
	  },
	  {
		"mimeType" : "video/rtx",
		"kind" : "video",
		"clockRate" : 90000,
		"preferredPayloadType" : 102,
		"rtcpFeedback" : [],
		"parameters" : { "apt":101 }
	  }
	],
	"headerExtensions" : [
	  {
		"kind" : "audio",
		"uri" : "urn:ietf:params:rtp-hdrext:ssrc-audio-level",
		"preferredId" : 1,
		"preferredEncrypt" : false
	  },
	  {
		"kind" : "video",
		"uri" : "urn:ietf:params:rtp-hdrext:toffset",
		"preferredId" : 2,
		"preferredEncrypt" : false
	  },
	  {
		"kind" : "audio",
		"uri" : "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time",
		"preferredId" : 3,
		"preferredEncrypt" : false
	  },
	  {
		"kind" : "video",
		"uri" : "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time",
		"preferredId" : 3,
		"preferredEncrypt" : false
	  },
	  {
		"kind" : "video",
		"uri" : "urn:3gpp:video-orientation",
		"preferredId" : 4,
		"preferredEncrypt" : false
	  },
	  {
		"kind" : "audio",
		"uri" : "urn:ietf:params:rtp-hdrext:sdes:mid",
		"preferredId" : 5,
		"preferredEncrypt" : false
	  },
	  {
		"kind" : "video",
		"uri" : "urn:ietf:params:rtp-hdrext:sdes:mid",
		"preferredId" : 5,
		"preferredEncrypt" : false
	  },
	  {
		"kind" : "video",
		"uri" : "urn:ietf:params:rtp-hdrext:sdes:rtp-stream-id",
		"preferredId" : 6,
		"preferredEncrypt" : false
	  }
	],
	"fecMechanisms" : []
  }
`
	)

	var (
		mediaCodecs             []RtpCodecCapability
		audioProducerParameters transportProduceParams
		videoProducerParameters transportProduceParams
	)

	err := json.Unmarshal([]byte(mediaCodecsJSON), &mediaCodecs)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal([]byte(audioProducerParametersJSON), &audioProducerParameters)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal([]byte(videoProducerParametersJSON), &videoProducerParameters)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal([]byte(consumerDeviceCapabilitiesJSON), &suite.consumerDeviceCapabilities)
	if err != nil {
		panic(err)
	}

	suite.worker = CreateTestWorker()
	suite.router, _ = suite.worker.CreateRouter(mediaCodecs)

	suite.transport1, _ = suite.router.CreateWebRtcTransport(CreateWebRtcTransportParams{
		ListenIps: []ListenIp{
			{Ip: "127.0.0.1"},
		},
	})
	suite.transport2, _ = suite.router.CreateWebRtcTransport(CreateWebRtcTransportParams{
		ListenIps: []ListenIp{
			{Ip: "127.0.0.1"},
		},
	})

	suite.audioProducer, _ = suite.transport1.Produce(audioProducerParameters)
	suite.videoProducer, _ = suite.transport1.Produce(videoProducerParameters)

	// // Pause the videoProducer.
	suite.videoProducer.Pause()
}

func (suite *ConsumerTestSuite) TestTransportConsume_Succeeds() {
	router, transport2 := suite.router, suite.transport2

	var called int
	var newConsumer *Consumer
	transport2.Observer().Once("newconsumer", func(c *Consumer) {
		called, newConsumer = called+1, c
	})

	suite.True(router.CanConsume(suite.audioProducer.Id(), suite.consumerDeviceCapabilities))

	audioConsumer, err := transport2.Consume(transportConsumeParams{
		ProducerId:      suite.audioProducer.Id(),
		RtpCapabilities: suite.consumerDeviceCapabilities,
		AppData:         H{"baz": "LOL"},
	})

	suite.NoError(err)
	suite.Equal(1, called)
	suite.Equal(audioConsumer, newConsumer)
	suite.NotEmpty(audioConsumer.Id())
	suite.Equal(suite.audioProducer.Id(), audioConsumer.ProducerId())
	suite.False(audioConsumer.Closed())
	suite.Equal("audio", audioConsumer.Kind())
	suite.NotEmpty(audioConsumer.RtpParameters())
	suite.Empty(audioConsumer.RtpParameters().Mid)
	suite.Len(audioConsumer.RtpParameters().Codecs, 1)

	data, _ := json.Marshal(audioConsumer.RtpParameters().Codecs[0])
	suite.JSONEq(`
	{
		"mimeType" : "audio/opus",
		"clockRate" : 48000,
		"payloadType" : 100,
		"channels" : 2,
		"parameters" : {
		  "useinbandfec" : 1,
		  "usedtx" : 1
		}
	  }
	`, string(data))

	suite.Equal("simple", audioConsumer.Type())
	suite.False(audioConsumer.Paused())
	suite.False(audioConsumer.ProducerPaused())
	data, _ = json.Marshal(audioConsumer.Score())
	suite.JSONEq(`{ "producer": 0, "consumer": 10 }`, string(data))
	suite.Equal(H{"baz": "LOL"}, audioConsumer.AppData())

	type RouterDump struct {
		MapProducerIdConsumerIds map[string][]string `json:"mapProducerIdConsumerIds"`
		MapConsumerIdProducerId  map[string]string   `json:"mapConsumerIdProducerId"`
	}
	var routerDump RouterDump
	router.Dump().Unmarshal(&routerDump)

	suite.Equal([]string{audioConsumer.Id()}, routerDump.MapProducerIdConsumerIds[suite.audioProducer.Id()])
	suite.Equal(suite.audioProducer.Id(), routerDump.MapConsumerIdProducerId[audioConsumer.Id()])

	type TransportDump struct {
		Id          string
		ProducerIds []string
		ConsumerIds []string
	}
	var transportDump TransportDump
	transport2.Dump().Unmarshal(&transportDump)

	suite.Equal(transport2.Id(), transportDump.Id)
	suite.Equal([]string{}, transportDump.ProducerIds)
	suite.Equal([]string{audioConsumer.Id()}, transportDump.ConsumerIds)

	called, newConsumer = 0, nil
	transport2.Observer().Once("newconsumer", func(c *Consumer) {
		called, newConsumer = called+1, c
	})

	suite.True(router.CanConsume(suite.videoProducer.Id(), suite.consumerDeviceCapabilities))

	videoConsumer, err := transport2.Consume(transportConsumeParams{
		ProducerId:      suite.videoProducer.Id(),
		RtpCapabilities: suite.consumerDeviceCapabilities,
		Paused:          true,
		AppData:         H{"baz": "LOL"},
	})

	suite.NoError(err)
	suite.Equal(transport2.ListenerCount("newconsumer"), 0)
	suite.Equal(1, called)
	suite.Equal(videoConsumer, newConsumer)
	suite.NotEmpty(videoConsumer.Id())
	suite.Equal(suite.videoProducer.Id(), videoConsumer.ProducerId())
	suite.False(videoConsumer.Closed())
	suite.Equal("video", videoConsumer.Kind())
	suite.NotEmpty(videoConsumer.RtpParameters())
	suite.Empty(videoConsumer.RtpParameters().Mid)
	suite.Len(videoConsumer.RtpParameters().Codecs, 2)
	data, _ = json.Marshal(videoConsumer.RtpParameters().Codecs[0])
	suite.JSONEq(`
	{
		"mimeType" : "video/H264",
		"clockRate" : 90000,
		"payloadType" : 103,
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
		  { "type":"ccm", "parameter":"fir" },
		  { "type":"goog-remb" }
		]
	  }
	`, string(data))
	data, _ = json.Marshal(videoConsumer.RtpParameters().Codecs[1])
	suite.JSONEq(`
	{
		"mimeType" : "video/rtx",
		"clockRate" : 90000,
		"payloadType" : 104,
		"parameters" : { "apt":103 }
	  }
	`, string(data))
	suite.Equal("simulcast", videoConsumer.Type())
	suite.True(videoConsumer.Paused())
	suite.True(videoConsumer.ProducerPaused())
	data, _ = json.Marshal(videoConsumer.Score())
	suite.JSONEq(`{ "producer": 0, "consumer": 10 }`, string(data))
	suite.Empty(videoConsumer.CurrentLayers())
	suite.Equal(H{"baz": "LOL"}, videoConsumer.AppData())

	routerDump = RouterDump{}
	router.Dump().Unmarshal(&routerDump)

	suite.Equal([]string{audioConsumer.Id()}, routerDump.MapProducerIdConsumerIds[suite.audioProducer.Id()])
	suite.Equal([]string{videoConsumer.Id()}, routerDump.MapProducerIdConsumerIds[suite.videoProducer.Id()])
	suite.Equal(suite.audioProducer.Id(), routerDump.MapConsumerIdProducerId[audioConsumer.Id()])
	suite.Equal(suite.videoProducer.Id(), routerDump.MapConsumerIdProducerId[videoConsumer.Id()])

	transportDump = TransportDump{}
	transport2.Dump().Unmarshal(&transportDump)

	suite.Equal(transport2.Id(), transportDump.Id)
	suite.Equal([]string{}, transportDump.ProducerIds)
	suite.ElementsMatch([]string{audioConsumer.Id(), videoConsumer.Id()}, transportDump.ConsumerIds)
}

func (suite *ConsumerTestSuite) TestTransportConsume_UnsupportedError() {
	router, transport2, audioProducer := suite.router, suite.transport2, suite.audioProducer
	invalidDeviceCapabilitiesJSON := `
	{
		"codecs" : [
		  {
			"kind" : "audio",
			"mimeType" : "audio/ISAC",
			"clockRate" : 32000,
			"preferredPayloadType" : 100,
			"channels" : 1
		  }
		],
		"headerExtensions" : []
	  }
	`
	var invalidDeviceCapabilities RtpCapabilities

	json.Unmarshal([]byte(invalidDeviceCapabilitiesJSON), &invalidDeviceCapabilities)

	suite.False(router.CanConsume(audioProducer.Id(), invalidDeviceCapabilities))

	_, err := transport2.Consume(transportConsumeParams{
		ProducerId:      audioProducer.Id(),
		RtpCapabilities: invalidDeviceCapabilities,
	})
	suite.IsType(NewUnsupportedError(""), err)

	invalidDeviceCapabilities = RtpCapabilities{}

	suite.False(router.CanConsume(audioProducer.Id(), invalidDeviceCapabilities))

	_, err = transport2.Consume(transportConsumeParams{
		ProducerId:      audioProducer.Id(),
		RtpCapabilities: invalidDeviceCapabilities,
	})
	suite.IsType(NewUnsupportedError(""), err)
}

func (suite *ConsumerTestSuite) TestConsumerDump() {
	type Dump struct {
		RtpParameters              *RtpParameters
		Id                         string
		Kind                       string
		Type                       string
		ConsumableRtpEncodings     []RtpMappingEncoding
		SupportedCodecPayloadTypes []uint32
		Paused                     bool
		ProducerPaused             bool
	}
	var data Dump
	audioConsumer := suite.audioConsumer()
	audioConsumer.Dump().Unmarshal(&data)

	suite.Equal(audioConsumer.Id(), data.Id)
	suite.Equal(audioConsumer.Kind(), data.Kind)
	suite.NotEmpty(data.RtpParameters)
	suite.Len(data.RtpParameters.Codecs, 1)
	suite.Equal("audio/opus", data.RtpParameters.Codecs[0].MimeType)
	suite.Equal(100, data.RtpParameters.Codecs[0].PayloadType)
	suite.Equal(48000, data.RtpParameters.Codecs[0].ClockRate)
	suite.Equal(2, data.RtpParameters.Codecs[0].Channels)
	suite.Equal(&RtpCodecParameter{Useinbandfec: 1, Usedtx: 1}, data.RtpParameters.Codecs[0].Parameters)
	suite.Equal([]RtcpFeedback{}, data.RtpParameters.Codecs[0].RtcpFeedback)
	suite.Len(data.RtpParameters.HeaderExtensions, 2)
	headerExtensionsJSON, _ := json.Marshal(data.RtpParameters.HeaderExtensions)
	suite.JSONEq(`
	[
		{
		  "uri" : "urn:ietf:params:rtp-hdrext:ssrc-audio-level",
		  "id" : 1,
		  "parameters" : {},
		  "encrypt"    : false
		},
		{
		  "uri" : "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time",
		  "id" : 3,
		  "parameters" : {},
		  "encrypt"    : false
		}
	  ]
	`, string(headerExtensionsJSON))
	suite.Len(data.RtpParameters.Encodings, 1)
	suite.Equal([]RtpEncoding{
		{CodecPayloadType: 100, Ssrc: audioConsumer.RtpParameters().Encodings[0].Ssrc},
	}, data.RtpParameters.Encodings)
	suite.Equal("simple", data.Type)
	suite.Len(data.ConsumableRtpEncodings, 1)
	suite.Equal([]RtpMappingEncoding{
		{Ssrc: suite.audioProducer.ConsumableRtpParameters().Encodings[0].Ssrc},
	}, data.ConsumableRtpEncodings)
	suite.Equal([]uint32{100}, data.SupportedCodecPayloadTypes)
	suite.False(data.Paused)
	suite.False(data.ProducerPaused)

	videoConsumer := suite.videoConsumer(true)
	data = Dump{}
	videoConsumer.Dump().Unmarshal(&data)

	suite.Equal(videoConsumer.Id(), data.Id)
	suite.Equal(videoConsumer.Kind(), data.Kind)
	suite.NotEmpty(data.RtpParameters)
	suite.Len(data.RtpParameters.Codecs, 2)
	suite.Equal("video/H264", data.RtpParameters.Codecs[0].MimeType)
	suite.Equal(103, data.RtpParameters.Codecs[0].PayloadType)
	suite.Equal(90000, data.RtpParameters.Codecs[0].ClockRate)
	suite.Empty(data.RtpParameters.Codecs[0].Channels)
	suite.Equal(h264profile.RtpH264Parameter{
		PacketizationMode: 1,
		ProfileLevelId:    "4d0032",
	}, data.RtpParameters.Codecs[0].Parameters.RtpH264Parameter)
	suite.EqualValues([]RtcpFeedback{
		{Type: "nack"},
		{Type: "nack", Parameter: "pli"},
		{Type: "ccm", Parameter: "fir"},
		{Type: "goog-remb"},
	}, data.RtpParameters.Codecs[0].RtcpFeedback)
	suite.Len(data.RtpParameters.HeaderExtensions, 3)
	headerExtensionsJSON, _ = json.Marshal(data.RtpParameters.HeaderExtensions)
	suite.JSONEq(`
	[
		{
		  "uri" : "urn:ietf:params:rtp-hdrext:toffset",
		  "id" : 2,
		  "parameters" : {},
		  "encrypt" : false
		},
		{
		  "uri" : "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time",
		  "id" : 3,
		  "parameters" : {},
		  "encrypt" : false
		},
		{
		  "uri" : "urn:3gpp:video-orientation",
		  "id" : 4,
		  "parameters" : {},
		  "encrypt" : false
		}
	  ]
	`, string(headerExtensionsJSON))
	suite.Len(data.RtpParameters.Encodings, 1)
	suite.EqualValues([]RtpEncoding{
		{
			CodecPayloadType: 103,
			Ssrc:             videoConsumer.RtpParameters().Encodings[0].Ssrc,
			Rtx: &RtpEncoding{
				Ssrc: videoConsumer.RtpParameters().Encodings[0].Rtx.Ssrc,
			},
		},
	}, data.RtpParameters.Encodings)
	suite.Len(data.ConsumableRtpEncodings, 4)
	suite.EqualValues([]RtpMappingEncoding{
		{Ssrc: suite.videoProducer.ConsumableRtpParameters().Encodings[0].Ssrc},
		{Ssrc: suite.videoProducer.ConsumableRtpParameters().Encodings[1].Ssrc},
		{Ssrc: suite.videoProducer.ConsumableRtpParameters().Encodings[2].Ssrc},
		{Ssrc: suite.videoProducer.ConsumableRtpParameters().Encodings[3].Ssrc},
	}, data.ConsumableRtpEncodings)
	suite.Equal([]uint32{103}, data.SupportedCodecPayloadTypes)
	suite.True(data.Paused)
	suite.True(data.ProducerPaused)
}

func (suite *ConsumerTestSuite) TestConsumerGetStats() {
	type Stats struct {
		Type     string
		Kind     string
		MimeType string
		Ssrc     uint32
	}

	audioConsumer := suite.audioConsumer()
	var data []Stats
	err := audioConsumer.GetStats().Unmarshal(&data)
	suite.NoError(err)

	suite.Contains(data, Stats{
		Type:     "outbound-rtp",
		Kind:     "audio",
		MimeType: "audio/opus",
		Ssrc:     audioConsumer.RtpParameters().Encodings[0].Ssrc,
	})

	videoConsumer := suite.videoConsumer(false)
	data = []Stats{}
	videoConsumer.GetStats().Unmarshal(&data)

	suite.Contains(data, Stats{
		Type:     "outbound-rtp",
		Kind:     "video",
		MimeType: "video/H264",
		Ssrc:     videoConsumer.RtpParameters().Encodings[0].Ssrc,
	})
}

func (suite *ConsumerTestSuite) TestConsumerPauseAndResume() {
	audioConsumer := suite.audioConsumer()

	audioConsumer.Pause()

	suite.True(audioConsumer.Paused())
	var data struct{ Paused bool }
	audioConsumer.Dump().Unmarshal(&data)
	suite.True(data.Paused)

	audioConsumer.Resume()

	suite.False(audioConsumer.Paused())
	audioConsumer.Dump().Unmarshal(&data)
	suite.False(data.Paused)
}

func (suite *ConsumerTestSuite) TestConsumerEmitsProducerpauseAndProducerresume() {
	audioConsumer := suite.audioConsumer()

	wf := NewWaitFunc(suite.T())

	audioConsumer.On("producerpause", wf.Fn())
	suite.audioProducer.Pause()

	wf.Wait()

	suite.False(audioConsumer.Paused())
	suite.True(audioConsumer.ProducerPaused())

	audioConsumer.On("producerresume", wf.Fn())
	suite.audioProducer.Resume()

	wf.Wait()

	suite.False(audioConsumer.Paused())
	suite.False(audioConsumer.ProducerPaused())
}

//Consumer emits "score"
func (suite *ConsumerTestSuite) TestConsumerEmitsScore() {
	audioConsumer := suite.audioConsumer()

	onScore := NewMockFunc()
	audioConsumer.On("score", onScore.Fn())

	channel := audioConsumer.channel

	channel.Emit(audioConsumer.Id(), "score", json.RawMessage(`{"producer": 10, "consumer": 9}`))
	channel.Emit(audioConsumer.Id(), "score", json.RawMessage(`{"producer": 9, "consumer": 9}`))
	channel.Emit(audioConsumer.Id(), "score", json.RawMessage(`{"producer": 8, "consumer": 8}`))

	suite.Equal(3, onScore.CalledTimes())
	suite.Equal(&ConsumerScore{Producer: 8, Consumer: 8}, audioConsumer.Score())
}

func (suite *ConsumerTestSuite) TestConsumerClose() {
	audioConsumer := suite.audioConsumer()
	videoConsumer := suite.videoConsumer(true)

	onObserverClose := NewMockFunc()

	audioConsumer.Observer().Once("close", onObserverClose.Fn())
	audioConsumer.Close()

	suite.Equal(1, onObserverClose.CalledTimes())
	suite.True(audioConsumer.Closed())

	var routerDump struct {
		MapProducerIdConsumerIds map[string][]string
		MapConsumerIdProducerId  map[string]string
	}
	suite.router.Dump().Unmarshal(&routerDump)

	suite.Empty(routerDump.MapProducerIdConsumerIds[suite.audioProducer.Id()])
	suite.Equal(suite.videoProducer.Id(), routerDump.MapConsumerIdProducerId[videoConsumer.Id()])

	var transportDump struct {
		Id          string
		ProducerIds []string
		ConsumerIds []string
	}
	suite.transport2.Dump().Unmarshal(&transportDump)

	suite.Equal(suite.transport2.Id(), transportDump.Id)
	suite.Empty(transportDump.ProducerIds)
	suite.EqualValues([]string{videoConsumer.Id()}, transportDump.ConsumerIds)
}

func (suite *ConsumerTestSuite) TestConsumerRejectIfClosed() {
	audioConsumer := suite.audioConsumer()
	audioConsumer.Close()

	suite.Error(audioConsumer.Dump().Err())
	suite.Error(audioConsumer.GetStats().Err())
	suite.Error(audioConsumer.Pause())
	suite.Error(audioConsumer.Resume())
	suite.Error(audioConsumer.SetPreferredLayers(0, 0))
	suite.Error(audioConsumer.RequestKeyFrame())
}

func (suite *ConsumerTestSuite) TestConsumerEmitsProducerClosed() {
	audioConsumer := suite.audioConsumer()

	onObserverClose := NewMockFunc()
	audioConsumer.Observer().Once("close", onObserverClose.Fn())

	wf := NewWaitFunc(suite.T())

	audioConsumer.On("producerclose", wf.Fn())

	suite.audioProducer.Close()

	wf.Wait()

	suite.Equal(1, onObserverClose.CalledTimes())
	suite.True(audioConsumer.Closed())
}

func (suite *ConsumerTestSuite) TestConsumerEmitsTransportClosed() {
	videoConsumer := suite.videoConsumer(false)

	onObserverClose := NewMockFunc()
	videoConsumer.Observer().Once("close", onObserverClose.Fn())

	wf := NewWaitFunc(suite.T())

	videoConsumer.Observer().On("transportclose", wf.Fn())

	suite.transport2.Close()

	suite.Equal(1, onObserverClose.CalledTimes())
	suite.True(videoConsumer.Closed())

	var routerDump struct {
		MapProducerIdConsumerIds map[string][]string
		MapConsumerIdProducerId  map[string]string
	}
	suite.router.Dump().Unmarshal(&routerDump)

	suite.Empty(routerDump.MapProducerIdConsumerIds[suite.audioProducer.Id()])
	suite.Empty(routerDump.MapConsumerIdProducerId)
}

func (suite *ConsumerTestSuite) audioConsumer() *Consumer {
	audioConsumer, _ := suite.transport2.Consume(transportConsumeParams{
		ProducerId:      suite.audioProducer.Id(),
		RtpCapabilities: suite.consumerDeviceCapabilities,
		AppData:         H{"baz": "LOL"},
	})

	return audioConsumer
}

func (suite *ConsumerTestSuite) videoConsumer(paused bool) *Consumer {
	videoConsumer, _ := suite.transport2.Consume(transportConsumeParams{
		ProducerId:      suite.videoProducer.Id(),
		RtpCapabilities: suite.consumerDeviceCapabilities,
		Paused:          paused,
		AppData:         H{"baz": "LOL"},
	})

	return videoConsumer
}

func TestConsumerTestSuite(t *testing.T) {
	suite.Run(t, new(ConsumerTestSuite))
}
