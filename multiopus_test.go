package mediasoup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiOpus(t *testing.T) {
	audioProducerOptions := ProducerOptions{
		Kind: MediaKind_Audio,
		RtpParameters: RtpParameters{
			Mid: "AUDIO",
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "audio/multiopus",
					PayloadType: 0,
					ClockRate:   48000,
					Channels:    6,
					Parameters: RtpCodecSpecificParameters{
						Useinbandfec:   1,
						ChannelMapping: "0,4,1,2,3,5",
						NumStreams:     4,
						CoupledStreams: 2,
					},
				},
			},
			HeaderExtensions: []RtpHeaderExtensionParameters{
				{
					Uri: "urn:ietf:params:rtp-hdrext:sdes:mid",
					Id:  10,
				},
				{
					Uri: "urn:ietf:params:rtp-hdrext:ssrc-audio-level",
					Id:  12,
				},
			},
		},
	}
	consumerDeviceCapabilities := RtpCapabilities{
		Codecs: []*RtpCodecCapability{
			{
				Kind:                 "audio",
				MimeType:             "audio/multiopus",
				PreferredPayloadType: 100,
				ClockRate:            48000,
				Channels:             6,
				Parameters: RtpCodecSpecificParameters{
					Useinbandfec:   1,
					ChannelMapping: "0,4,1,2,3,5",
					NumStreams:     4,
					CoupledStreams: 2,
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
				Kind:             "audio",
				Uri:              "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time",
				PreferredId:      4,
				PreferredEncrypt: false,
			},
			{
				Kind:             "audio",
				Uri:              "urn:ietf:params:rtp-hdrext:ssrc-audio-level",
				PreferredId:      10,
				PreferredEncrypt: false,
			},
		},
	}

	// before all
	worker := CreateTestWorker()
	router, _ := worker.CreateRouter(RouterOptions{
		MediaCodecs: []*RtpCodecCapability{
			{
				Kind:      "audio",
				MimeType:  "audio/multiopus",
				ClockRate: 48000,
				Channels:  6,
				Parameters: RtpCodecSpecificParameters{
					Useinbandfec:   1,
					ChannelMapping: "0,4,1,2,3,5",
					NumStreams:     4,
					CoupledStreams: 2,
				},
			},
		},
	})
	transport, _ := router.CreateWebRtcTransport(WebRtcTransportOptions{
		ListenIps: []TransportListenIp{{Ip: "127.0.0.1"}},
	})
	// after all
	defer worker.Close()

	t.Run("produce/consume succeeds", func(t *testing.T) {
		audioProducer, err := transport.Produce(audioProducerOptions)
		require.NoError(t, err)

		assert.Equal(t, &RtpCodecParameters{
			MimeType:    "audio/multiopus",
			PayloadType: 0,
			ClockRate:   48000,
			Channels:    6,
			Parameters: RtpCodecSpecificParameters{
				Useinbandfec:   1,
				ChannelMapping: "0,4,1,2,3,5",
				NumStreams:     4,
				CoupledStreams: 2,
			},
		}, audioProducer.RtpParameters().Codecs[0])

		assert.True(t, router.CanConsume(audioProducer.Id(), consumerDeviceCapabilities))

		audioConsumer, _ := transport.Consume(ConsumerOptions{
			ProducerId:      audioProducer.Id(),
			RtpCapabilities: consumerDeviceCapabilities,
		})
		assert.Equal(t, &RtpCodecParameters{
			MimeType:    "audio/multiopus",
			PayloadType: 100,
			ClockRate:   48000,
			Channels:    6,
			Parameters: RtpCodecSpecificParameters{
				Useinbandfec:   1,
				ChannelMapping: "0,4,1,2,3,5",
				NumStreams:     4,
				CoupledStreams: 2,
			},
		}, audioConsumer.RtpParameters().Codecs[0])

		audioProducer.Close()
		audioConsumer.Close()
	})

	t.Run("fails to produce wrong parameters", func(t *testing.T) {
		_, err := transport.Produce(ProducerOptions{
			Kind: MediaKind_Audio,
			RtpParameters: RtpParameters{
				Mid: "AUDIO",
				Codecs: []*RtpCodecParameters{
					{
						MimeType:    "audio/multiopus",
						PayloadType: 0,
						ClockRate:   48000,
						Channels:    6,
						Parameters: RtpCodecSpecificParameters{
							Useinbandfec:   1,
							ChannelMapping: "0,4,1,2,3,5",
							NumStreams:     2,
							CoupledStreams: 2,
						},
					},
				},
			},
		})
		assert.IsType(t, UnsupportedError{}, err)

		_, err = transport.Produce(ProducerOptions{
			Kind: MediaKind_Audio,
			RtpParameters: RtpParameters{
				Mid: "AUDIO",
				Codecs: []*RtpCodecParameters{
					{
						MimeType:    "audio/multiopus",
						PayloadType: 0,
						ClockRate:   48000,
						Channels:    6,
						Parameters: RtpCodecSpecificParameters{
							Useinbandfec:   1,
							ChannelMapping: "0,4,1,2,3,5",
							NumStreams:     4,
							CoupledStreams: 1,
						},
					},
				},
			},
		})
		assert.IsType(t, UnsupportedError{}, err)
	})

	t.Run("fails to consume wrong channels", func(t *testing.T) {
		audioProducer, _ := transport.Produce(audioProducerOptions)
		localConsumerDeviceCapabilities := RtpCapabilities{
			Codecs: []*RtpCodecCapability{
				{
					Kind:                 "audio",
					MimeType:             "audio/multiopus",
					PreferredPayloadType: 100,
					ClockRate:            48000,
					Channels:             8,
					Parameters: RtpCodecSpecificParameters{
						Useinbandfec:   1,
						ChannelMapping: "0,4,1,2,3,5",
						NumStreams:     4,
						CoupledStreams: 2,
					},
				},
			},
		}
		assert.False(t, router.CanConsume(audioProducer.Id(), localConsumerDeviceCapabilities))
		_, err := transport.Consume(ConsumerOptions{
			ProducerId:      audioProducer.Id(),
			RtpCapabilities: localConsumerDeviceCapabilities,
		})
		assert.Error(t, err)

		audioProducer.Close()
	})
}
