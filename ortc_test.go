package mediasoup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateRouterRtpCapabilities(t *testing.T) {
	t.Run("succeed", func(t *testing.T) {
		mediaCodecs := []*RtpCodecCapability{
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
				Kind:                 "video",
				MimeType:             "video/VP8",
				PreferredPayloadType: 125,
				ClockRate:            90000,
			},
			{
				Kind:      "video",
				MimeType:  "video/H264",
				ClockRate: 90000,
				Parameters: RtpCodecSpecificParameters{
					LevelAsymmetryAllowed: 1,
					ProfileLevelId:        "42e01f",
				},
			},
		}

		rtpCapabilities, err := generateRouterRtpCapabilities(mediaCodecs)
		assert.NoError(t, err)

		assert.Len(t, rtpCapabilities.Codecs, 5)
		assert.Equal(t, &RtpCodecCapability{
			Kind:                 MediaKindAudio,
			MimeType:             "audio/opus",
			PreferredPayloadType: 100, // 100 is the first available dynamic PT.
			Channels:             2,
			ClockRate:            48000,
			Parameters:           RtpCodecSpecificParameters{Useinbandfec: 1},
			RtcpFeedback: []*RtcpFeedback{
				{Type: "nack"},
				{Type: "transport-cc"},
			},
		}, rtpCapabilities.Codecs[0])

		assert.Equal(t, &RtpCodecCapability{
			Kind:                 MediaKindVideo,
			MimeType:             "video/VP8",
			PreferredPayloadType: 125,
			ClockRate:            90000,
			RtcpFeedback: []*RtcpFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "goog-remb"},
				{Type: "transport-cc"},
			},
		}, rtpCapabilities.Codecs[1])

		assert.Equal(t, &RtpCodecCapability{
			Kind:                 MediaKindVideo,
			MimeType:             "video/rtx",
			PreferredPayloadType: 101, // 101 is the second available dynamic PT.
			ClockRate:            90000,
			Parameters:           RtpCodecSpecificParameters{Apt: 125},
			RtcpFeedback:         []*RtcpFeedback{},
		}, rtpCapabilities.Codecs[2])

		assert.Equal(t, &RtpCodecCapability{
			Kind:                 MediaKindVideo,
			MimeType:             "video/H264",
			PreferredPayloadType: 102, // 102 is the third available dynamic PT.
			ClockRate:            90000,
			Parameters: RtpCodecSpecificParameters{
				LevelAsymmetryAllowed: 1,
				ProfileLevelId:        "42e01f",
			},
			RtcpFeedback: []*RtcpFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "goog-remb"},
				{Type: "transport-cc"},
			},
		}, rtpCapabilities.Codecs[3])

		assert.Equal(t, &RtpCodecCapability{
			Kind:                 MediaKindVideo,
			MimeType:             "video/rtx",
			PreferredPayloadType: 103,
			ClockRate:            90000,
			Parameters:           RtpCodecSpecificParameters{Apt: 102},
			RtcpFeedback:         []*RtcpFeedback{},
		}, rtpCapabilities.Codecs[4])
	})

	t.Run("unsupported codecs", func(t *testing.T) {
		mediaCodecs := []*RtpCodecCapability{
			{
				Kind:      "audio",
				MimeType:  "audio/chicken",
				ClockRate: 48000,
				Channels:  4,
			},
		}
		_, err := generateRouterRtpCapabilities(mediaCodecs)
		assert.Error(t, err)

		mediaCodecs = []*RtpCodecCapability{
			{
				Kind:      "audio",
				MimeType:  "audio/opus",
				ClockRate: 48000,
				Channels:  1,
			},
		}
		_, err = generateRouterRtpCapabilities(mediaCodecs)
		assert.Error(t, err)
	})

	t.Run("too many codecs", func(t *testing.T) {
		mediaCodecs := []*RtpCodecCapability{}
		for i := 0; i < 100; i++ {
			mediaCodecs = append(mediaCodecs, &RtpCodecCapability{
				Kind:      "audio",
				MimeType:  "audio/opus",
				ClockRate: 48000,
				Channels:  2,
			})
		}
		_, err := generateRouterRtpCapabilities(mediaCodecs)
		assert.Error(t, err)
	})
}

func TestRtpParameters(t *testing.T) {
	mediaCodecs := []*RtpCodecCapability{
		{
			Kind:      "audio",
			MimeType:  "audio/opus",
			ClockRate: 48000,
			Channels:  2,
		},
		{
			Kind:      "video",
			MimeType:  "video/H264",
			ClockRate: 90000,
			Parameters: RtpCodecSpecificParameters{
				LevelAsymmetryAllowed: 1,
				PacketizationMode:     1,
				ProfileLevelId:        "4d0032",
			},
		},
	}

	routerRtpCapabilities, _ := generateRouterRtpCapabilities(mediaCodecs)

	rtpParameters := &RtpParameters{
		Codecs: []*RtpCodecParameters{
			{
				MimeType:    "video/H264",
				PayloadType: 111,
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
				PayloadType: 112,
				ClockRate:   90000,
				Parameters: RtpCodecSpecificParameters{
					Apt: 111,
				},
				RtcpFeedback: []*RtcpFeedback{},
			},
		},
		HeaderExtensions: []*RtpHeaderExtensionParameters{
			{
				Id:  1,
				Uri: "urn:ietf:params:rtp-hdrext:sdes:mid",
			},
			{
				Id:  2,
				Uri: "urn:3gpp:video-orientation",
			},
		},
		Encodings: []*RtpEncodingParameters{
			{
				Ssrc:            11111111,
				Rtx:             &RtpEncodingRtx{Ssrc: 11111112},
				MaxBitrate:      111111,
				ScalabilityMode: "L1T3",
			},
			{
				Ssrc:            21111111,
				Rtx:             &RtpEncodingRtx{Ssrc: 21111112},
				MaxBitrate:      222222,
				ScalabilityMode: "L1T3",
			},
			{
				Rid:             "high",
				MaxBitrate:      333333,
				ScalabilityMode: "L1T3",
			},
		},
		Rtcp: &RtcpParameters{Cname: "qwerty1234"},
	}

	rtpMapping, err := getProducerRtpParametersMapping(rtpParameters, routerRtpCapabilities)
	assert.NoError(t, err)

	assert.ElementsMatch(t, []*RtpMappingCodec{
		{PayloadType: 111, MappedPayloadType: 101},
		{PayloadType: 112, MappedPayloadType: 102},
	}, rtpMapping.Codecs)
	assert.EqualValues(t, 11111111, *rtpMapping.Encodings[0].Ssrc)
	assert.Empty(t, rtpMapping.Encodings[0].Rid)
	assert.NotZero(t, rtpMapping.Encodings[0].MappedSsrc)
	assert.EqualValues(t, 21111111, *rtpMapping.Encodings[1].Ssrc)
	assert.Empty(t, rtpMapping.Encodings[1].Rid)
	assert.NotZero(t, rtpMapping.Encodings[1].MappedSsrc)
	assert.Empty(t, rtpMapping.Encodings[2].Ssrc)
	assert.Equal(t, "high", rtpMapping.Encodings[2].Rid)
	assert.NotZero(t, rtpMapping.Encodings[2].MappedSsrc)

	consumableRtpParameters := getConsumableRtpParameters(MediaKindVideo, rtpParameters, routerRtpCapabilities, rtpMapping)

	assert.Equal(t, "video/H264", consumableRtpParameters.Codecs[0].MimeType)
	assert.EqualValues(t, 101, consumableRtpParameters.Codecs[0].PayloadType)
	assert.EqualValues(t, 90000, consumableRtpParameters.Codecs[0].ClockRate)
	assert.Equal(t, RtpCodecSpecificParameters{
		PacketizationMode: 1,
		ProfileLevelId:    "4d0032",
	}, consumableRtpParameters.Codecs[0].Parameters)

	assert.Equal(t, "video/rtx", consumableRtpParameters.Codecs[1].MimeType)
	assert.EqualValues(t, 102, consumableRtpParameters.Codecs[1].PayloadType)
	assert.EqualValues(t, 90000, consumableRtpParameters.Codecs[1].ClockRate)
	assert.Equal(t, RtpCodecSpecificParameters{
		Apt: 101,
	}, consumableRtpParameters.Codecs[1].Parameters)

	assert.Equal(t, &RtpEncodingParameters{
		Ssrc:            rtpMapping.Encodings[0].MappedSsrc,
		MaxBitrate:      111111,
		ScalabilityMode: "L1T3",
	}, consumableRtpParameters.Encodings[0])
	assert.Equal(t, &RtpEncodingParameters{
		Ssrc:            rtpMapping.Encodings[1].MappedSsrc,
		MaxBitrate:      222222,
		ScalabilityMode: "L1T3",
	}, consumableRtpParameters.Encodings[1])
	assert.Equal(t, &RtpEncodingParameters{
		Ssrc:            rtpMapping.Encodings[2].MappedSsrc,
		MaxBitrate:      333333,
		ScalabilityMode: "L1T3",
	}, consumableRtpParameters.Encodings[2])

	assert.Equal(t, &RtcpParameters{
		Cname:       rtpParameters.Rtcp.Cname,
		ReducedSize: ref(true),
	}, consumableRtpParameters.Rtcp)

	remoteRtpCapabilities := &RtpCapabilities{
		Codecs: []*RtpCodecCapability{
			{
				MimeType:             "audio/opus",
				Kind:                 "audio",
				PreferredPayloadType: 100,
				ClockRate:            48000,
				Channels:             2,
			},
			{
				MimeType:             "video/H264",
				Kind:                 "video",
				PreferredPayloadType: 101,
				ClockRate:            90000,
				Parameters: RtpCodecSpecificParameters{
					PacketizationMode: 1,
					ProfileLevelId:    "4d0032",
				},
				RtcpFeedback: []*RtcpFeedback{
					{Type: "nack", Parameter: ""},
					{Type: "nack", Parameter: "pli"},
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
				Direction:        MediaDirectionSendrecv,
			},
			{
				Kind:             "video",
				Uri:              "urn:ietf:params:rtp-hdrext:sdes:mid",
				PreferredId:      1,
				PreferredEncrypt: false,
				Direction:        MediaDirectionSendrecv,
			},
			{
				Kind:             "video",
				Uri:              "urn:ietf:params:rtp-hdrext:sdes:rtp-stream-id",
				PreferredId:      2,
				PreferredEncrypt: false,
				Direction:        MediaDirectionSendrecv,
			},
			{
				Kind:             "audio",
				Uri:              "urn:ietf:params:rtp-hdrext:ssrc-audio-level",
				PreferredId:      6,
				PreferredEncrypt: false,
				Direction:        MediaDirectionSendrecv,
			},
			{
				Kind:             "video",
				Uri:              "urn:3gpp:video-orientation",
				PreferredId:      8,
				PreferredEncrypt: false,
				Direction:        MediaDirectionSendrecv,
			},
			{
				Kind:             "video",
				Uri:              "urn:ietf:params:rtp-hdrext:toffset",
				PreferredId:      9,
				PreferredEncrypt: false,
				Direction:        MediaDirectionSendrecv,
			},
		},
	}

	consumerRtpParameters, err := getConsumerRtpParameters(
		consumableRtpParameters,
		remoteRtpCapabilities,
		false,
		true,
	)
	assert.NoError(t, err)

	assert.Len(t, consumerRtpParameters.Codecs, 2)
	assert.Equal(t, &RtpCodecParameters{
		MimeType:    "video/H264",
		PayloadType: 101,
		ClockRate:   90000,
		Parameters: RtpCodecSpecificParameters{
			PacketizationMode: 1,
			ProfileLevelId:    "4d0032",
		},
		RtcpFeedback: []*RtcpFeedback{
			{Type: "nack"},
			{Type: "nack", Parameter: "pli"},
		},
	}, consumerRtpParameters.Codecs[0])
	assert.Equal(t, &RtpCodecParameters{
		MimeType:    "video/rtx",
		PayloadType: 102,
		ClockRate:   90000,
		Parameters:  RtpCodecSpecificParameters{Apt: 101},
	}, consumerRtpParameters.Codecs[1])

	assert.Len(t, consumerRtpParameters.Encodings, 1)
	assert.NotZero(t, consumerRtpParameters.Encodings[0].Ssrc)
	assert.NotZero(t, consumerRtpParameters.Encodings[0].Rtx.Ssrc)
	assert.Equal(t, "L3T3", consumerRtpParameters.Encodings[0].ScalabilityMode)
	assert.EqualValues(t, 333333, consumerRtpParameters.Encodings[0].MaxBitrate)

	assert.Equal(t, []*RtpHeaderExtensionParameters{
		{
			Uri:     "urn:ietf:params:rtp-hdrext:sdes:mid",
			Id:      1,
			Encrypt: false,
		},
		{
			Uri:     "urn:3gpp:video-orientation",
			Id:      8,
			Encrypt: false,
		},
		{
			Uri:     "urn:ietf:params:rtp-hdrext:toffset",
			Id:      9,
			Encrypt: false,
		},
	}, consumerRtpParameters.HeaderExtensions)

	assert.Equal(t, &RtcpParameters{
		Cname:       rtpParameters.Rtcp.Cname,
		ReducedSize: ref(true),
	}, consumerRtpParameters.Rtcp)

	pipeConsumerRtpParameters := getPipeConsumerRtpParameters(consumableRtpParameters, false)

	assert.Len(t, pipeConsumerRtpParameters.Codecs, 1)
	assert.Equal(t, &RtpCodecParameters{
		MimeType:    "video/H264",
		PayloadType: 101,
		ClockRate:   90000,
		Parameters: RtpCodecSpecificParameters{
			PacketizationMode: 1,
			ProfileLevelId:    "4d0032",
		},
		RtcpFeedback: []*RtcpFeedback{
			{Type: "nack", Parameter: "pli"},
			{Type: "ccm", Parameter: "fir"},
		},
	}, pipeConsumerRtpParameters.Codecs[0])

	assert.Len(t, pipeConsumerRtpParameters.Encodings, 3)
	assert.NotZero(t, pipeConsumerRtpParameters.Encodings[0].Ssrc)
	assert.Empty(t, pipeConsumerRtpParameters.Encodings[0].Rtx)
	assert.NotZero(t, pipeConsumerRtpParameters.Encodings[0].MaxBitrate)
	assert.Equal(t, "L1T3", pipeConsumerRtpParameters.Encodings[0].ScalabilityMode)
	assert.NotZero(t, pipeConsumerRtpParameters.Encodings[1].Ssrc)
	assert.Empty(t, pipeConsumerRtpParameters.Encodings[1].Rtx)
	assert.NotZero(t, pipeConsumerRtpParameters.Encodings[1].MaxBitrate)
	assert.Equal(t, "L1T3", pipeConsumerRtpParameters.Encodings[1].ScalabilityMode)
	assert.NotZero(t, pipeConsumerRtpParameters.Encodings[2].Ssrc)
	assert.Empty(t, pipeConsumerRtpParameters.Encodings[2].Rtx)
	assert.NotZero(t, pipeConsumerRtpParameters.Encodings[2].MaxBitrate)
	assert.Equal(t, "L1T3", pipeConsumerRtpParameters.Encodings[2].ScalabilityMode)

	assert.Equal(t, &RtcpParameters{
		Cname:       rtpParameters.Rtcp.Cname,
		ReducedSize: ref(true),
	}, pipeConsumerRtpParameters.Rtcp)
}

func TestGetProducerRtpParametersMappingWithIncompatibleParams(t *testing.T) {
	mediaCodecs := []*RtpCodecCapability{
		{
			Kind:      "audio",
			MimeType:  "audio/opus",
			ClockRate: 48000,
			Channels:  2,
		},
		{
			Kind:      "video",
			MimeType:  "video/H264",
			ClockRate: 90000,
			Parameters: RtpCodecSpecificParameters{
				PacketizationMode: 1,
				ProfileLevelId:    "4d0032",
			},
		},
	}

	routerRtpCapabilities, _ := generateRouterRtpCapabilities(mediaCodecs)

	rtpParameters := &RtpParameters{
		Codecs: []*RtpCodecParameters{
			{
				MimeType:    "video/VP8",
				PayloadType: 120,
				ClockRate:   90000,
				RtcpFeedback: []*RtcpFeedback{
					{Type: "nack", Parameter: ""},
					{Type: "nack", Parameter: "fir"},
				},
			},
		},

		Encodings: []*RtpEncodingParameters{
			{
				Ssrc: 11111111,
			},
		},
		Rtcp: &RtcpParameters{Cname: "qwerty1234"},
	}

	_, err := getProducerRtpParametersMapping(rtpParameters, routerRtpCapabilities)
	assert.Error(t, err)
}

// TestGetConsumerRtpParametersOverride exercises the "override" variant of
// getConsumerRtpParameters (where the caller passes *RtpParameters instead of
// *RtpCapabilities) together with getConsumerRtpMapping. The shape mirrors the
// per-Consumer egress remap use case.
func TestGetConsumerRtpParametersOverride(t *testing.T) {
	makeConsumable := func() *RtpParameters {
		return &RtpParameters{
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "video/H264",
					PayloadType: 101,
					ClockRate:   90000,
					Parameters: RtpCodecSpecificParameters{
						PacketizationMode: 1,
						ProfileLevelId:    "4d0032",
					},
					RtcpFeedback: []*RtcpFeedback{
						{Type: "nack"},
						{Type: "nack", Parameter: "pli"},
					},
				},
				{
					MimeType:    "video/rtx",
					PayloadType: 102,
					ClockRate:   90000,
					Parameters:  RtpCodecSpecificParameters{Apt: 101},
				},
			},
			HeaderExtensions: []*RtpHeaderExtensionParameters{
				{Uri: "urn:ietf:params:rtp-hdrext:sdes:mid", Id: 1},
				{
					Uri: "http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01",
					Id:  5,
				},
			},
			Encodings: []*RtpEncodingParameters{
				{
					Ssrc:            10000001,
					MaxBitrate:      500000,
					ScalabilityMode: "L1T3",
				},
			},
			Rtcp: &RtcpParameters{Cname: "cname1234"},
		}
	}

	t.Run("succeeds with happy path and produces a mapping", func(t *testing.T) {
		consumable := makeConsumable()
		override := &RtpParameters{
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "video/H264",
					PayloadType: 97,
					ClockRate:   90000,
					Parameters: RtpCodecSpecificParameters{
						PacketizationMode: 1,
						ProfileLevelId:    "4d0032",
					},
				},
				{
					MimeType:    "video/rtx",
					PayloadType: 98,
					ClockRate:   90000,
					Parameters:  RtpCodecSpecificParameters{Apt: 97},
				},
			},
			HeaderExtensions: []*RtpHeaderExtensionParameters{
				{Uri: "urn:ietf:params:rtp-hdrext:sdes:mid", Id: 3},
				{
					Uri: "http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01",
					Id:  7,
				},
			},
		}

		rtpParameters, err := getConsumerRtpParameters(consumable, override, false, true)
		assert.NoError(t, err)
		assert.Len(t, rtpParameters.Codecs, 2)
		assert.EqualValues(t, 97, rtpParameters.Codecs[0].PayloadType)
		assert.EqualValues(t, 98, rtpParameters.Codecs[1].PayloadType)

		// rtcp should fall back to consumable when not provided by caller.
		if assert.NotNil(t, rtpParameters.Rtcp) {
			assert.Equal(t, "cname1234", rtpParameters.Rtcp.Cname)
		}

		mapping := getConsumerRtpMapping(consumable, rtpParameters)
		assert.ElementsMatch(t, []ConsumerCodecMapping{
			{ProducerPayloadType: 101, ConsumerPayloadType: 97},
			{ProducerPayloadType: 102, ConsumerPayloadType: 98},
		}, mapping.Codecs)

		assert.ElementsMatch(t, []ConsumerHeaderExtensionMapping{
			{ProducerExtId: 1, ConsumerExtId: 3},
			{ProducerExtId: 5, ConsumerExtId: 7},
		}, mapping.HeaderExtensions)
	})

	t.Run("auto-generates SSRCs regardless of caller-provided encodings", func(t *testing.T) {
		consumable := makeConsumable()
		override := &RtpParameters{
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "video/H264",
					PayloadType: 97,
					ClockRate:   90000,
					Parameters: RtpCodecSpecificParameters{
						PacketizationMode: 1,
						ProfileLevelId:    "4d0032",
					},
				},
				{
					MimeType:    "video/rtx",
					PayloadType: 98,
					ClockRate:   90000,
					Parameters:  RtpCodecSpecificParameters{Apt: 97},
				},
			},
			HeaderExtensions: []*RtpHeaderExtensionParameters{
				{Uri: "urn:ietf:params:rtp-hdrext:sdes:mid", Id: 3},
			},
		}

		rtpParameters, err := getConsumerRtpParameters(consumable, override, false, true)
		assert.NoError(t, err)
		assert.Len(t, rtpParameters.Encodings, 1)
		assert.NotZero(t, rtpParameters.Encodings[0].Ssrc)
		if assert.NotNil(t, rtpParameters.Encodings[0].Rtx) {
			assert.NotZero(t, rtpParameters.Encodings[0].Rtx.Ssrc)
		}
	})

	t.Run("rtcp.cname from caller is preserved when provided", func(t *testing.T) {
		consumable := makeConsumable()
		override := &RtpParameters{
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "video/H264",
					PayloadType: 97,
					ClockRate:   90000,
					Parameters: RtpCodecSpecificParameters{
						PacketizationMode: 1,
						ProfileLevelId:    "4d0032",
					},
				},
			},
			Rtcp: &RtcpParameters{Cname: "custom-cname"},
		}

		rtpParameters, err := getConsumerRtpParameters(consumable, override, false, false)
		assert.NoError(t, err)
		if assert.NotNil(t, rtpParameters.Rtcp) {
			assert.Equal(t, "custom-cname", rtpParameters.Rtcp.Cname)
		}
	})

	t.Run("errors when no codec has a consumable counterpart", func(t *testing.T) {
		consumable := makeConsumable()
		override := &RtpParameters{
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "video/VP8",
					PayloadType: 97,
					ClockRate:   90000,
				},
			},
		}

		_, err := getConsumerRtpParameters(consumable, override, false, true)
		assert.Error(t, err)
	})

	t.Run("drops RTX codec when its apt points to no consumer-side codec", func(t *testing.T) {
		consumable := makeConsumable()
		override := &RtpParameters{
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "video/H264",
					PayloadType: 97,
					ClockRate:   90000,
					Parameters: RtpCodecSpecificParameters{
						PacketizationMode: 1,
						ProfileLevelId:    "4d0032",
					},
				},
				{
					MimeType:    "video/rtx",
					PayloadType: 98,
					ClockRate:   90000,
					Parameters:  RtpCodecSpecificParameters{Apt: 123},
				},
			},
		}

		rtpParameters, err := getConsumerRtpParameters(consumable, override, false, true)
		assert.NoError(t, err)
		// RTX was sanitised out because its apt does not match any media codec.
		assert.Len(t, rtpParameters.Codecs, 1)
		assert.EqualValues(t, 97, rtpParameters.Codecs[0].PayloadType)
	})

	t.Run("drops unknown header extension URIs from the final rtpParameters", func(t *testing.T) {
		consumable := makeConsumable()
		override := &RtpParameters{
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "video/H264",
					PayloadType: 97,
					ClockRate:   90000,
					Parameters: RtpCodecSpecificParameters{
						PacketizationMode: 1,
						ProfileLevelId:    "4d0032",
					},
				},
			},
			HeaderExtensions: []*RtpHeaderExtensionParameters{
				{Uri: "urn:ietf:params:rtp-hdrext:sdes:mid", Id: 3},
				{Uri: "urn:3gpp:video-orientation", Id: 2},
			},
		}

		rtpParameters, err := getConsumerRtpParameters(consumable, override, false, false)
		assert.NoError(t, err)
		assert.Len(t, rtpParameters.HeaderExtensions, 1)
		assert.Equal(t, "urn:ietf:params:rtp-hdrext:sdes:mid", rtpParameters.HeaderExtensions[0].Uri)
	})

	t.Run("keeps all matching header extensions (no early break)", func(t *testing.T) {
		consumable := makeConsumable()
		override := &RtpParameters{
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "video/H264",
					PayloadType: 97,
					ClockRate:   90000,
					Parameters: RtpCodecSpecificParameters{
						PacketizationMode: 1,
						ProfileLevelId:    "4d0032",
					},
				},
			},
			HeaderExtensions: []*RtpHeaderExtensionParameters{
				{Uri: "urn:ietf:params:rtp-hdrext:sdes:mid", Id: 3},
				{
					Uri: "http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01",
					Id:  7,
				},
			},
		}

		rtpParameters, err := getConsumerRtpParameters(consumable, override, false, false)
		assert.NoError(t, err)
		assert.Len(t, rtpParameters.HeaderExtensions, 2)
	})

	t.Run("rejects header extension with zero id (validateRtpParameters)", func(t *testing.T) {
		consumable := makeConsumable()
		override := &RtpParameters{
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "video/H264",
					PayloadType: 97,
					ClockRate:   90000,
					Parameters: RtpCodecSpecificParameters{
						PacketizationMode: 1,
						ProfileLevelId:    "4d0032",
					},
				},
			},
			HeaderExtensions: []*RtpHeaderExtensionParameters{
				{Uri: "urn:ietf:params:rtp-hdrext:sdes:mid", Id: 0},
			},
		}

		_, err := getConsumerRtpParameters(consumable, override, false, false)
		assert.Error(t, err)
	})

	t.Run("preserves caller-declared rtcpFeedback (rfc4585 subset-of-offer)", func(t *testing.T) {
		// Regression test: if the Producer-side consumable rtcpFeedback is
		// richer than the caller's override, the final Consumer rtpParameters
		// must still carry the caller's list (not the consumable's). This is
		// the WHEP case: the offer dictates which feedback types are legal in
		// the answer; leaking extras (e.g. `nack` on opus) would violate
		// RFC 4585 §4.2.2.
		consumable := &RtpParameters{
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "audio/opus",
					PayloadType: 100,
					ClockRate:   48000,
					Channels:    2,
					// Producer-side feedback is `nack` (not what we want the
					// WHEP answer to advertise).
					RtcpFeedback: []*RtcpFeedback{{Type: "nack"}},
				},
			},
			HeaderExtensions: []*RtpHeaderExtensionParameters{
				{Uri: "urn:ietf:params:rtp-hdrext:sdes:mid", Id: 1},
				{
					Uri: "http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01",
					Id:  4,
				},
			},
			Encodings: []*RtpEncodingParameters{{Ssrc: 20000001}},
			Rtcp:      &RtcpParameters{Cname: "cname1234"},
		}
		override := &RtpParameters{
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "audio/opus",
					PayloadType: 111,
					ClockRate:   48000,
					Channels:    2,
					// Caller offered transport-cc only (this is what the
					// browser actually advertised in the WHEP offer).
					RtcpFeedback: []*RtcpFeedback{{Type: "transport-cc"}},
				},
			},
			HeaderExtensions: []*RtpHeaderExtensionParameters{
				{Uri: "urn:ietf:params:rtp-hdrext:sdes:mid", Id: 9},
				{
					Uri: "http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01",
					Id:  4,
				},
			},
		}

		rtpParameters, err := getConsumerRtpParameters(consumable, override, false, false)
		assert.NoError(t, err)
		require.Len(t, rtpParameters.Codecs, 1)
		// Must be exactly what the caller declared: no `nack` leaked from
		// consumable, no feedback types never offered.
		assert.Equal(t, []*RtcpFeedback{{Type: "transport-cc"}}, rtpParameters.Codecs[0].RtcpFeedback)
	})

	t.Run("mid is preserved from caller-provided rtpParameters", func(t *testing.T) {
		consumable := makeConsumable()
		override := &RtpParameters{
			Mid: "video0",
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "video/H264",
					PayloadType: 97,
					ClockRate:   90000,
					Parameters: RtpCodecSpecificParameters{
						PacketizationMode: 1,
						ProfileLevelId:    "4d0032",
					},
				},
			},
			HeaderExtensions: []*RtpHeaderExtensionParameters{
				{Uri: "urn:ietf:params:rtp-hdrext:sdes:mid", Id: 3},
			},
		}

		rtpParameters, err := getConsumerRtpParameters(consumable, override, false, false)
		assert.NoError(t, err)
		// Caller's mid flows through to the final Consumer rtpParameters.
		// Transport.ConsumeContext() only falls back to options.Mid / auto
		// when rtpParameters.Mid is empty.
		assert.Equal(t, "video0", rtpParameters.Mid)
	})

	t.Run("mid left empty when caller did not provide one", func(t *testing.T) {
		consumable := makeConsumable()
		override := &RtpParameters{
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "video/H264",
					PayloadType: 97,
					ClockRate:   90000,
					Parameters: RtpCodecSpecificParameters{
						PacketizationMode: 1,
						ProfileLevelId:    "4d0032",
					},
				},
			},
		}

		rtpParameters, err := getConsumerRtpParameters(consumable, override, false, false)
		assert.NoError(t, err)
		// Transport.ConsumeContext() is what assigns the MID in this case;
		// getConsumerRtpParameters must not fabricate one here.
		assert.Empty(t, rtpParameters.Mid)
	})

	t.Run("enableRtx=false strips RTX from the caller-provided codec list", func(t *testing.T) {
		consumable := makeConsumable()
		override := &RtpParameters{
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "video/H264",
					PayloadType: 97,
					ClockRate:   90000,
					Parameters: RtpCodecSpecificParameters{
						PacketizationMode: 1,
						ProfileLevelId:    "4d0032",
					},
				},
				{
					MimeType:    "video/rtx",
					PayloadType: 98,
					ClockRate:   90000,
					Parameters:  RtpCodecSpecificParameters{Apt: 97},
				},
			},
		}

		rtpParameters, err := getConsumerRtpParameters(consumable, override, false, false)
		assert.NoError(t, err)
		assert.Len(t, rtpParameters.Codecs, 1)
		assert.EqualValues(t, 97, rtpParameters.Codecs[0].PayloadType)
		if assert.Len(t, rtpParameters.Encodings, 1) {
			assert.Nil(t, rtpParameters.Encodings[0].Rtx)
		}
	})
}
