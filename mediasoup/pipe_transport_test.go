package mediasoup

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testPipeMediaCodecs = []RtpCodecCapability{
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
		MimeType:  "video/VP8",
		ClockRate: 90000,
	},
}

const audioProducerParametersJSON = `
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
			"foo" : "bar1"
		  }
		}
	  ],
	  "headerExtensions" : [
		{
		  "uri" : "urn:ietf:params:rtp-hdrext:sdes:mid",
		  "id" : 10
		}
	  ],
	  "encodings" : [ { "ssrc":11111111 } ],
	  "rtcp" : { "cname":"FOOBAR" }
	},
	"appData" : { "foo":"bar1" }
}`

const videoProducerParametersJSON = `
{
  "kind" : "video",
  "rtpParameters" : {
    "mid" : "VIDEO",
    "codecs" : [
      {
        "mimeType" : "video/VP8",
        "payloadType" : 112,
        "clockRate" : 90000,
        "rtcpFeedback" : [
          { "type":"nack" },
          {
            "type" : "nack",
            "parameter" : "pli"
          },
          { "type":"goog-remb" },
          { "type":"lalala" }
        ]
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
      { "ssrc":22222222 },
      { "ssrc":22222223 },
      { "ssrc":22222224 }
    ],
    "rtcp" : { "cname":"FOOBAR" }
  },
  "appData" : { "foo":"bar2" }
}`

const consumerDeviceCapabilitiesJSON = `
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
      "mimeType" : "video/VP8",
      "kind" : "video",
      "clockRate" : 90000,
      "preferredPayloadType" : 101,
      "rtcpFeedback" : [
        { "type":"nack" },
        {
          "type" : "ccm",
          "parameter" : "fir"
        },
        { "type":"goog-remb" }
      ]
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
    }
  ]
}`

var (
	audioProducerParameters    transportProduceParams
	videoProducerParameters    transportProduceParams
	consumerDeviceCapabilities RtpCapabilities
)

type PipeTransportNs struct {
	router1       *Router
	router2       *Router
	transport1    *WebRtcTransport
	transport2    *WebRtcTransport
	audioProducer *Producer
	videoProducer *Producer
}

func init() {
	err := json.Unmarshal([]byte(audioProducerParametersJSON), &audioProducerParameters)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal([]byte(videoProducerParametersJSON), &videoProducerParameters)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal([]byte(consumerDeviceCapabilitiesJSON), &consumerDeviceCapabilities)
	if err != nil {
		panic(err)
	}
}

func setupPipeTest(t *testing.T) PipeTransportNs {
	router1, err := worker.CreateRouter(testPipeMediaCodecs)
	assert.NoError(t, err)
	router2, err := worker.CreateRouter(testPipeMediaCodecs)
	assert.NoError(t, err)

	transport1, err := router1.CreateWebRtcTransport(CreateWebRtcTransportParams{
		ListenIps: []ListenIp{
			{Ip: "127.0.0.1"},
		},
	})
	assert.NoError(t, err)
	transport2, err := router2.CreateWebRtcTransport(CreateWebRtcTransportParams{
		ListenIps: []ListenIp{
			{Ip: "127.0.0.1"},
		},
	})
	assert.NoError(t, err)

	audioProducer, err := transport1.Produce(audioProducerParameters)
	assert.NoError(t, err)
	videoProducer, err := transport1.Produce(videoProducerParameters)
	assert.NoError(t, err)

	// // Pause the videoProducer.
	err = videoProducer.Pause()
	assert.NoError(t, err)

	return PipeTransportNs{
		router1:       router1,
		router2:       router2,
		transport1:    transport1,
		transport2:    transport2,
		audioProducer: audioProducer,
		videoProducer: videoProducer,
	}
}

func TestRouterPipeToRouter_SucceedsWithAudio(t *testing.T) {
	ns := setupPipeTest(t)

	pipeConsumer, pipeProducer, err := ns.router1.PipeToRouter(PipeToRouterParams{
		ProducerId: ns.audioProducer.Id(),
		Router:     ns.router2,
	})
	assert.NoError(t, err)

	var dump struct {
		TransportIds []string
	}
	ns.router1.Dump().Unmarshal(&dump)

	// There shoud should be two Transports in router1:
	// - WebRtcTransport for audioProducer and videoProducer.
	// - PipeTransport between router1 and router2.
	assert.Len(t, dump.TransportIds, 2)

	dump.TransportIds = nil
	ns.router2.Dump().Unmarshal(&dump)

	// There shoud should be two Transports in router2:
	// - WebRtcTransport for audioConsumer and videoConsumer.
	// - pipeTransport between router2 and router1.
	assert.Len(t, dump.TransportIds, 2)

	assert.False(t, pipeConsumer.Closed())
	assert.NotNil(t, pipeConsumer.RtpParameters())
	assert.Zero(t, pipeConsumer.RtpParameters().Mid)
	assertJSONEq(t, pipeConsumer.RtpParameters().Codecs, []RtpCodecCapability{
		{
			MimeType:    "audio/opus",
			ClockRate:   48000,
			Channels:    2,
			PayloadType: 100,
			Parameters:  &RtpCodecParameter{Useinbandfec: 1},
		},
	})
	assertJSONEq(t, pipeConsumer.RtpParameters().HeaderExtensions, []RtpHeaderExtension{
		{
			Uri: "urn:ietf:params:rtp-hdrext:ssrc-audio-level",
			Id:  1,
		},
	})
	assertJSONEq(t, pipeConsumer.RtpParameters().Encodings, []RtpEncoding{
		ns.audioProducer.ConsumableRtpParameters().Encodings[0],
	})
	assert.Equal(t, pipeConsumer.Type(), "pipe")
	assert.False(t, pipeConsumer.Paused())
	assert.False(t, pipeConsumer.ProducerPaused())
	assert.Zero(t, pipeConsumer.Score())
	assertJSONEq(t, pipeConsumer.AppData(), H{})

	assert.Equal(t, pipeProducer.Id(), ns.audioProducer.Id())
	assert.False(t, pipeProducer.Closed())
	assert.Equal(t, pipeProducer.Kind(), "audio")
	assert.NotNil(t, pipeProducer.RtpParameters())
	assert.Zero(t, pipeProducer.RtpParameters().Mid)
	assertJSONEq(t, pipeProducer.RtpParameters().Codecs, []RtpCodecCapability{
		{
			MimeType:     "audio/opus",
			ClockRate:    48000,
			Channels:     2,
			PayloadType:  100,
			Parameters:   &RtpCodecParameter{Useinbandfec: 1},
			RtcpFeedback: []RtcpFeedback{},
		},
	})
	assertJSONEq(t, pipeProducer.RtpParameters().HeaderExtensions, []RtpHeaderExtension{
		{
			Uri: "urn:ietf:params:rtp-hdrext:ssrc-audio-level",
			Id:  1,
		},
	})
	assertJSONEq(t, pipeProducer.RtpParameters().Encodings, []RtpEncoding{
		ns.audioProducer.ConsumableRtpParameters().Encodings[0],
	})
	assert.False(t, pipeProducer.Paused())
}

func TestRouterPipeToRouter_SucceedsWithVideo(t *testing.T) {
	ns := setupPipeTest(t)

	ns.router1.PipeToRouter(PipeToRouterParams{
		ProducerId: ns.audioProducer.Id(),
		Router:     ns.router2,
	})

	pipeConsumer, pipeProducer, err := ns.router1.PipeToRouter(PipeToRouterParams{
		ProducerId: ns.videoProducer.Id(),
		Router:     ns.router2,
	})
	assert.NoError(t, err)

	var dump struct {
		TransportIds []string
	}
	ns.router1.Dump().Unmarshal(&dump)

	// No new PipeTransport should has been created. The existing one is used.
	assert.Len(t, dump.TransportIds, 2)

	dump.TransportIds = nil
	ns.router2.Dump().Unmarshal(&dump)

	// No new PipeTransport should has been created. The existing one is used.
	assert.Len(t, dump.TransportIds, 2)

	assert.NotEmpty(t, pipeConsumer.Id())
	assert.False(t, pipeConsumer.Closed())
	assert.Equal(t, pipeConsumer.Kind(), "video")
	assert.NotNil(t, pipeConsumer.RtpParameters())
	assert.Zero(t, pipeConsumer.RtpParameters().Mid)
	assertJSONEq(t, pipeConsumer.RtpParameters().Codecs, []RtpCodecCapability{
		{
			MimeType:    "video/VP8",
			ClockRate:   90000,
			PayloadType: 101,
			RtcpFeedback: []RtcpFeedback{
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
			},
		},
	})
	assertJSONEq(t, pipeConsumer.RtpParameters().HeaderExtensions, []RtpHeaderExtension{
		{
			Uri: "urn:ietf:params:rtp-hdrext:toffset",
			Id:  2,
		},
		{
			Uri: "urn:3gpp:video-orientation",
			Id:  4,
		},
	})
	assertJSONEq(t, pipeConsumer.RtpParameters().Encodings, []RtpEncoding{
		ns.videoProducer.ConsumableRtpParameters().Encodings[0],
		ns.videoProducer.ConsumableRtpParameters().Encodings[1],
		ns.videoProducer.ConsumableRtpParameters().Encodings[2],
	})
	assert.Equal(t, pipeConsumer.Type(), "pipe")
	assert.False(t, pipeConsumer.Paused())
	assert.True(t, pipeConsumer.ProducerPaused())
	assert.Zero(t, pipeConsumer.Score())
	assertJSONEq(t, pipeConsumer.AppData(), H{})

	assert.Equal(t, pipeProducer.Id(), ns.videoProducer.Id())
	assert.False(t, pipeProducer.Closed())
	assert.Equal(t, pipeProducer.Kind(), "video")
	assert.NotNil(t, pipeProducer.RtpParameters())
	assert.Zero(t, pipeProducer.RtpParameters().Mid)
	assertJSONEq(t, pipeProducer.RtpParameters().Codecs, []RtpCodecCapability{
		{
			MimeType:    "video/VP8",
			ClockRate:   90000,
			PayloadType: 101,
			RtcpFeedback: []RtcpFeedback{
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
			},
		},
	})
	assertJSONEq(t, pipeProducer.RtpParameters().HeaderExtensions, []RtpHeaderExtension{
		{
			Uri: "urn:ietf:params:rtp-hdrext:toffset",
			Id:  2,
		},
		{
			Uri: "urn:3gpp:video-orientation",
			Id:  4,
		},
	})
	assert.True(t, pipeProducer.Paused())
}

func TestTransportConsume_ForAPipeProducerSucceeds(t *testing.T) {
	ns := setupPipeTest(t)

	ns.router1.PipeToRouter(PipeToRouterParams{
		ProducerId: ns.audioProducer.Id(),
		Router:     ns.router2,
	})

	ns.router1.PipeToRouter(PipeToRouterParams{
		ProducerId: ns.videoProducer.Id(),
		Router:     ns.router2,
	})

	videoConsumer, err := ns.transport2.Consume(transportConsumeParams{
		ProducerId:      ns.videoProducer.Id(),
		RtpCapabilities: consumerDeviceCapabilities,
	})
	assert.NoError(t, err)

	assert.NotEmpty(t, videoConsumer.Id())
	assert.False(t, videoConsumer.Closed())
	assert.Equal(t, videoConsumer.Kind(), "video")
	assert.NotNil(t, videoConsumer.RtpParameters())
	assert.Zero(t, videoConsumer.RtpParameters().Mid)
	assertJSONEq(t, videoConsumer.RtpParameters().Codecs, []RtpCodecCapability{
		{
			MimeType:    "video/VP8",
			ClockRate:   90000,
			PayloadType: 101,
			RtcpFeedback: []RtcpFeedback{
				{Type: "nack"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "goog-remb"},
			},
		},
		{
			MimeType:    "video/rtx",
			ClockRate:   90000,
			PayloadType: 102,
			Parameters: &RtpCodecParameter{
				Apt: 101,
			},
		},
	})
	assertJSONEq(t, videoConsumer.RtpParameters().HeaderExtensions, []RtpHeaderExtension{})
	assert.Len(t, videoConsumer.RtpParameters().Encodings, 1)
	assert.NotZero(t, videoConsumer.RtpParameters().Encodings[0].Ssrc)
	assert.NotEmpty(t, videoConsumer.RtpParameters().Encodings[0].Rtx)
	assert.NotZero(t, videoConsumer.RtpParameters().Encodings[0].Rtx.Ssrc)
	assert.Equal(t, videoConsumer.Type(), "simulcast")
	assert.False(t, videoConsumer.Paused())
	assert.True(t, videoConsumer.ProducerPaused())
	assertJSONEq(t, videoConsumer.Score(), ConsumerScore{Producer: 0, Consumer: 10})
	assertJSONEq(t, videoConsumer.AppData(), H{})
}

func TestTransportPauseAndResumeAreTransmittedToPipeConsumer(t *testing.T) {
	ns := setupPipeTest(t)

	ns.router1.PipeToRouter(PipeToRouterParams{
		ProducerId: ns.audioProducer.Id(),
		Router:     ns.router2,
	})

	ns.router1.PipeToRouter(PipeToRouterParams{
		ProducerId: ns.videoProducer.Id(),
		Router:     ns.router2,
	})

	videoConsumer, err := ns.transport2.Consume(transportConsumeParams{
		ProducerId:      ns.videoProducer.Id(),
		RtpCapabilities: consumerDeviceCapabilities,
	})
	assert.NoError(t, err)

	assert.True(t, ns.videoProducer.Paused())
	assert.True(t, videoConsumer.ProducerPaused())
	assert.False(t, videoConsumer.Paused())

	wg := sync.WaitGroup{}
	wg.Add(1)
	if videoConsumer.ProducerPaused() {
		videoConsumer.Once("producerresume", func() { wg.Done() })
	}

	assert.NoError(t, ns.videoProducer.Resume())

	wg.Wait()

	assert.False(t, videoConsumer.ProducerPaused())
	assert.False(t, videoConsumer.Paused())

	wg.Add(1)
	if !videoConsumer.ProducerPaused() {
		videoConsumer.Once("producerpause", func() { wg.Done() })
	}

	assert.NoError(t, ns.videoProducer.Pause())

	wg.Wait()

	assert.True(t, videoConsumer.ProducerPaused())
	assert.False(t, videoConsumer.Paused())
}

func TestProducerCloseIsTransmittedToPipeConsumer(t *testing.T) {
	ns := setupPipeTest(t)

	ns.router1.PipeToRouter(PipeToRouterParams{
		ProducerId: ns.audioProducer.Id(),
		Router:     ns.router2,
	})

	ns.router1.PipeToRouter(PipeToRouterParams{
		ProducerId: ns.videoProducer.Id(),
		Router:     ns.router2,
	})

	videoConsumer, _ := ns.transport2.Consume(transportConsumeParams{
		ProducerId:      ns.videoProducer.Id(),
		RtpCapabilities: consumerDeviceCapabilities,
	})

	err := ns.videoProducer.Close()
	assert.NoError(t, err)

	assert.True(t, ns.videoProducer.Closed())

	wg := sync.WaitGroup{}
	wg.Add(1)
	if !videoConsumer.Closed() {
		videoConsumer.Once("producerclose", func() { wg.Done() })
	}

	wg.Wait()

	assert.True(t, videoConsumer.Closed())
}
