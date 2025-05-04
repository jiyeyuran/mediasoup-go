package mediasoup

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
				PreferredId:      10,
				PreferredEncrypt: false,
				Direction:        MediaDirectionSendrecv,
			},
			{
				Kind:             "video",
				Uri:              "urn:3gpp:video-orientation",
				PreferredId:      11,
				PreferredEncrypt: false,
				Direction:        MediaDirectionSendrecv,
			},
			{
				Kind:             "video",
				Uri:              "urn:ietf:params:rtp-hdrext:toffset",
				PreferredId:      12,
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
			Id:      11,
			Encrypt: false,
		},
		{
			Uri:     "urn:ietf:params:rtp-hdrext:toffset",
			Id:      12,
			Encrypt: false,
		},
	}, consumerRtpParameters.HeaderExtensions)

	assert.Nil(t, consumerRtpParameters.Rtcp)

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
