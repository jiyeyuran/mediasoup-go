package mediasoup

import (
	"encoding/json"
	"testing"

	"github.com/jiyeyuran/mediasoup-go/mediasoup/h264profile"
	"github.com/stretchr/testify/assert"
)

func TestGenerateRouterRtpCapabilities_Succeeds(t *testing.T) {
	mediaCodecs := []RtpCodecCapability{
		{
			Kind:         "audio",
			MimeType:     "audio/opus",
			ClockRate:    48000,
			Channels:     2,
			RtcpFeedback: []RtcpFeedback{},
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
			Kind:         "video",
			MimeType:     "video/H264",
			ClockRate:    90000,
			RtcpFeedback: []RtcpFeedback{},
			Parameters: &RtpParameter{
				RtpH264Parameter: h264profile.RtpH264Parameter{
					LevelAsymmetryAllowed: 1,
					ProfileLevelId:        "42e01f",
				},
			},
		},
	}

	rtpCapabilities, err := GenerateRouterRtpCapabilities(mediaCodecs)
	assert.NoError(t, err)
	assert.Len(t, rtpCapabilities.Codecs, 5)

	// opus.
	assert.Equal(t, RtpCodecCapability{
		Kind:                 "audio",
		MimeType:             "audio/opus",
		PreferredPayloadType: 100, // 100 is the first PT chosen.
		ClockRate:            48000,
		Channels:             2,
		RtcpFeedback:         []RtcpFeedback{},
		Parameters: &RtpParameter{
			Useinbandfec: 1,
		},
	}, rtpCapabilities.Codecs[0])

	// VP8.
	assert.Equal(t, RtpCodecCapability{
		Kind:                 "video",
		MimeType:             "video/VP8",
		PreferredPayloadType: 101,
		ClockRate:            90000,
		RtcpFeedback: []RtcpFeedback{
			{Type: "nack"},
			{Type: "nack", Parameter: "pli"},
			{Type: "ccm", Parameter: "fir"},
			{Type: "goog-remb"},
		},
		Parameters: &RtpParameter{},
	}, rtpCapabilities.Codecs[1])

	// VP8 RTX.
	assert.Equal(t, RtpCodecCapability{
		Kind:                 "video",
		MimeType:             "video/rtx",
		PreferredPayloadType: 102,
		ClockRate:            90000,
		RtcpFeedback:         []RtcpFeedback{},
		Parameters: &RtpParameter{
			Apt: 101,
		},
	}, rtpCapabilities.Codecs[2])

	// H264.
	assert.Equal(t, RtpCodecCapability{
		Kind:                 "video",
		MimeType:             "video/H264",
		PreferredPayloadType: 103,
		ClockRate:            90000,
		RtcpFeedback: []RtcpFeedback{
			{Type: "nack"},
			{Type: "nack", Parameter: "pli"},
			{Type: "ccm", Parameter: "fir"},
			{Type: "goog-remb"},
		},
		Parameters: &RtpParameter{
			RtpH264Parameter: h264profile.RtpH264Parameter{
				PacketizationMode:     0,
				LevelAsymmetryAllowed: 1,
				ProfileLevelId:        "42e01f",
			},
		},
	}, rtpCapabilities.Codecs[3])

	// H264 RTX.
	assert.Equal(t, RtpCodecCapability{
		Kind:                 "video",
		MimeType:             "video/rtx",
		PreferredPayloadType: 104,
		ClockRate:            90000,
		RtcpFeedback:         []RtcpFeedback{},
		Parameters: &RtpParameter{
			Apt: 103,
		},
	}, rtpCapabilities.Codecs[4])
}

func TestGenerateRouterRtpCapabilities_UnsupportedError(t *testing.T) {
	mediaCodecss := [][]RtpCodecCapability{
		[]RtpCodecCapability{
			{
				Kind:      "audio",
				MimeType:  "audio/opus",
				ClockRate: 48000,
				Channels:  4,
			},
		},
		[]RtpCodecCapability{
			{
				Kind:      "audio",
				MimeType:  "audio/opus",
				ClockRate: 48000,
				Channels:  1,
			},
		},
		[]RtpCodecCapability{
			{
				Kind:      "video",
				MimeType:  "video/H264",
				ClockRate: 90000,
				Parameters: &RtpParameter{
					RtpH264Parameter: h264profile.RtpH264Parameter{
						PacketizationMode: 5,
					},
				},
			},
		},
	}
	for _, mediaCodecs := range mediaCodecss {
		_, err := GenerateRouterRtpCapabilities(mediaCodecs)
		assert.IsType(t, err, NewUnsupportedError(""))
	}

}

func TestGenerateRouterRtpCapabilities_TooManyCodecs(t *testing.T) {
	mediaCodecs := []RtpCodecCapability{}

	for i := 0; i < 100; i++ {
		mediaCodecs = append(mediaCodecs, RtpCodecCapability{
			Kind:      "audio",
			MimeType:  "audio/opus",
			ClockRate: 48000,
			Channels:  2,
		})
	}
	_, err := GenerateRouterRtpCapabilities(mediaCodecs)
	assert.Error(t, err)
}

func TestProducerComsumerPipeRtpParameters_Succeed(t *testing.T) {
	mediaCodecs := []RtpCodecCapability{
		{
			Kind:         "audio",
			MimeType:     "audio/opus",
			ClockRate:    48000,
			Channels:     2,
			RtcpFeedback: []RtcpFeedback{},
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

	routerRtpCapabilities, err := GenerateRouterRtpCapabilities(mediaCodecs)
	assert.NoError(t, err)

	rtpParameters := RtpRemoteCapabilities{
		Codecs: []RtpMappingCodec{
			{
				RtpCodecCapability: &RtpCodecCapability{
					Kind:      "video",
					MimeType:  "video/H264",
					ClockRate: 90000,
					RtcpFeedback: []RtcpFeedback{
						{Type: "nack"},
						{Type: "nack", Parameter: "pli"},
						{Type: "goog-remb"},
					},
					Parameters: &RtpParameter{
						RtpH264Parameter: h264profile.RtpH264Parameter{
							PacketizationMode: 1,
							ProfileLevelId:    "4d0032",
						},
					},
				},
				PayloadType: 111,
			},
			{
				RtpCodecCapability: &RtpCodecCapability{
					MimeType:  "video/rtx",
					ClockRate: 90000,
					Parameters: &RtpParameter{
						Apt: 111,
					},
				},
				PayloadType: 112,
			},
		},
		HeaderExtensions: []RtpMappingHeaderExt{
			{
				RtpHeaderExtension: &RtpHeaderExtension{
					Uri: "urn:ietf:params:rtp-hdrext:sdes:mid",
				},
				Id: 1,
			},
			{
				RtpHeaderExtension: &RtpHeaderExtension{
					Uri: "urn:3gpp:video-orientation",
				},
				Id: 2,
			},
		},
		Encodings: []RtpMappingEncoding{
			{
				Ssrc:       11111111,
				MaxBitrate: 111111,
				Rtx: &RtpMappingEncoding{
					Ssrc: 11111112,
				},
			},
			{
				Ssrc:       21111111,
				MaxBitrate: 222222,
				Rtx: &RtpMappingEncoding{
					Ssrc: 21111112,
				},
			},
			{
				Rid:        "high",
				MaxBitrate: 333333,
			},
		},
		Rtcp: RtcpConfiguation{
			Cname: "qwerty1234",
		},
	}

	rtpMapping, err := GetProducerRtpParametersMapping(rtpParameters, routerRtpCapabilities)
	assert.NoError(t, err)

	assert.ElementsMatch(t, []RtpMappingCodec{
		{PayloadType: 111, MappedPayloadType: 101},
		{PayloadType: 112, MappedPayloadType: 102},
	}, rtpMapping.Codecs)

	assert.ElementsMatch(t, []RtpMappingHeaderExt{
		{Id: 1, MappedId: 5},
		{Id: 2, MappedId: 4},
	}, rtpMapping.HeaderExtensions)

	assert.EqualValues(t, 11111111, rtpMapping.Encodings[0].Ssrc)
	assert.Empty(t, rtpMapping.Encodings[0].Rid)
	assert.NotEmpty(t, rtpMapping.Encodings[0].MappedSsrc)
	assert.EqualValues(t, 21111111, rtpMapping.Encodings[1].Ssrc)
	assert.Empty(t, rtpMapping.Encodings[1].Rid)
	assert.NotEmpty(t, rtpMapping.Encodings[1].MappedSsrc)
	assert.Empty(t, rtpMapping.Encodings[2].Ssrc)
	assert.Equal(t, "high", rtpMapping.Encodings[2].Rid)
	assert.NotEmpty(t, rtpMapping.Encodings[2].MappedSsrc)

	consumableRtpParameters, err := GetConsumableRtpParameters("video",
		rtpParameters, routerRtpCapabilities, rtpMapping)
	assert.NoError(t, err)

	assert.Equal(t, "video/H264", consumableRtpParameters.Codecs[0].MimeType)
	assert.EqualValues(t, 101, consumableRtpParameters.Codecs[0].PayloadType)
	assert.EqualValues(t, 90000, consumableRtpParameters.Codecs[0].ClockRate)
	assert.Equal(t, &RtpParameter{
		RtpH264Parameter: h264profile.RtpH264Parameter{
			PacketizationMode: 1,
			ProfileLevelId:    "4d0032",
		},
	}, consumableRtpParameters.Codecs[0].Parameters)

	assert.Equal(t, "video/rtx", consumableRtpParameters.Codecs[1].MimeType)
	assert.EqualValues(t, 102, consumableRtpParameters.Codecs[1].PayloadType)
	assert.EqualValues(t, 90000, consumableRtpParameters.Codecs[1].ClockRate)
	assert.Equal(t, &RtpParameter{Apt: 101}, consumableRtpParameters.Codecs[1].Parameters)

	assert.Equal(t, RtpMappingEncoding{
		Ssrc:       rtpMapping.Encodings[0].MappedSsrc,
		MaxBitrate: 111111,
	}, consumableRtpParameters.Encodings[0])
	assert.Equal(t, RtpMappingEncoding{
		Ssrc:       rtpMapping.Encodings[1].MappedSsrc,
		MaxBitrate: 222222,
	}, consumableRtpParameters.Encodings[1])
	assert.Equal(t, RtpMappingEncoding{
		Ssrc:       rtpMapping.Encodings[2].MappedSsrc,
		MaxBitrate: 333333,
	}, consumableRtpParameters.Encodings[2])

	assert.Equal(t, RtcpConfiguation{
		Cname:       rtpParameters.Rtcp.Cname,
		ReducedSize: true,
		Mux:         true,
	}, consumableRtpParameters.Rtcp)

	remoteRtpCapabilities := RtpCapabilities{
		Codecs: []RtpCodecCapability{
			{
				Kind:                 "audio",
				MimeType:             "audio/opus",
				ClockRate:            48000,
				Channels:             2,
				PreferredPayloadType: 100,
			},
			{
				Kind:                 "video",
				MimeType:             "video/H264",
				ClockRate:            90000,
				PreferredPayloadType: 101,
				RtcpFeedback: []RtcpFeedback{
					{Type: "nack"},
					{Type: "nack", Parameter: "pli"},
					{Type: "foo", Parameter: "FOO"},
				},
				Parameters: &RtpParameter{
					RtpH264Parameter: h264profile.RtpH264Parameter{
						PacketizationMode: 1,
						ProfileLevelId:    "4d0032",
					},
				},
			},
			{
				Kind:                 "video",
				MimeType:             "video/rtx",
				ClockRate:            90000,
				PreferredPayloadType: 102,
				Parameters: &RtpParameter{
					Apt: 101,
				},
			},
		},
		HeaderExtensions: []RtpHeaderExtension{
			{
				Kind:             "audio",
				Uri:              "urn:ietf:params:rtp-hdrext:ssrc-audio-level",
				PreferredId:      1,
				PreferredEncrypt: false,
			},
			{
				Kind:             "video",
				Uri:              "urn:ietf:params:rtp-hdrext:toffset",
				PreferredId:      2,
				PreferredEncrypt: false,
			},
			{
				Kind:             "video",
				Uri:              "urn:3gpp:video-orientation",
				PreferredId:      4,
				PreferredEncrypt: false,
			},
			{
				Kind:             "audio",
				Uri:              "urn:ietf:params:rtp-hdrext:sdes:mid",
				PreferredId:      5,
				PreferredEncrypt: false,
			},
			{
				Kind:             "video",
				Uri:              "urn:ietf:params:rtp-hdrext:sdes:mid",
				PreferredId:      5,
				PreferredEncrypt: false,
			},
			{
				Kind:             "video",
				Uri:              "urn:ietf:params:rtp-hdrext:sdes:rtp-stream-id",
				PreferredId:      6,
				PreferredEncrypt: false,
			},
		},
	}

	consumerRtpParameters, err := GetConsumerRtpParameters(
		consumableRtpParameters, remoteRtpCapabilities)
	assert.NoError(t, err)

	assert.Len(t, consumerRtpParameters.Codecs, 2)
	assertJSONEq(t, RtpMappingCodec{
		RtpCodecCapability: &RtpCodecCapability{
			MimeType:  "video/H264",
			ClockRate: 90000,
			RtcpFeedback: []RtcpFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "foo", Parameter: "FOO"},
			},
			Parameters: &RtpParameter{
				RtpH264Parameter: h264profile.RtpH264Parameter{
					PacketizationMode: 1,
					ProfileLevelId:    "4d0032",
				},
			},
		},
		PayloadType: 101,
	}, consumerRtpParameters.Codecs[0])
	assertJSONEq(t, RtpMappingCodec{
		RtpCodecCapability: &RtpCodecCapability{
			MimeType:     "video/rtx",
			ClockRate:    90000,
			RtcpFeedback: []RtcpFeedback{},
			Parameters: &RtpParameter{
				Apt: 101,
			},
		},
		PayloadType: 102,
	}, consumerRtpParameters.Codecs[1])
	assert.Len(t, consumerRtpParameters.Encodings, 1)
	assert.NotEmpty(t, consumerRtpParameters.Encodings[0].Ssrc)
	assert.NotEmpty(t, consumerRtpParameters.Encodings[0].Rtx)
	assert.NotEmpty(t, consumerRtpParameters.Encodings[0].Rtx.Ssrc)

	assert.ElementsMatch(t, []RtpMappingHeaderExt{
		{
			RtpHeaderExtension: &RtpHeaderExtension{
				Uri: "urn:ietf:params:rtp-hdrext:toffset",
			},
			Id: 2,
		},
		{
			RtpHeaderExtension: &RtpHeaderExtension{
				Uri: "urn:3gpp:video-orientation",
			},
			Id: 4,
		},
	}, consumerRtpParameters.HeaderExtensions)

	assert.Equal(t, RtcpConfiguation{
		Cname:       rtpParameters.Rtcp.Cname,
		ReducedSize: true,
		Mux:         true,
	}, consumerRtpParameters.Rtcp)

	pipeConsumerRtpParameters := GetPipeConsumerRtpParameters(consumableRtpParameters)

	assert.Len(t, pipeConsumerRtpParameters.Codecs, 1)
	assertJSONEq(t, RtpMappingCodec{
		RtpCodecCapability: &RtpCodecCapability{
			MimeType:  "video/H264",
			ClockRate: 90000,
			RtcpFeedback: []RtcpFeedback{
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
			},
			Parameters: &RtpParameter{
				RtpH264Parameter: h264profile.RtpH264Parameter{
					PacketizationMode: 1,
					ProfileLevelId:    "4d0032",
				},
			},
		},
		PayloadType: 101,
	}, pipeConsumerRtpParameters.Codecs[0])

	assert.Len(t, pipeConsumerRtpParameters.Encodings, 3)
	assert.NotZero(t, pipeConsumerRtpParameters.Encodings[0].Ssrc)
	assert.Nil(t, pipeConsumerRtpParameters.Encodings[0].Rtx)
	assert.NotZero(t, pipeConsumerRtpParameters.Encodings[0].MaxBitrate)
	assert.NotZero(t, pipeConsumerRtpParameters.Encodings[1].Ssrc)
	assert.Nil(t, pipeConsumerRtpParameters.Encodings[1].Rtx)
	assert.NotZero(t, pipeConsumerRtpParameters.Encodings[1].MaxBitrate)
	assert.NotZero(t, pipeConsumerRtpParameters.Encodings[2].Ssrc)
	assert.Nil(t, pipeConsumerRtpParameters.Encodings[2].Rtx)
	assert.NotZero(t, pipeConsumerRtpParameters.Encodings[2].MaxBitrate)

	assert.Equal(t, RtcpConfiguation{
		Cname:       rtpParameters.Rtcp.Cname,
		ReducedSize: true,
		Mux:         true,
	}, pipeConsumerRtpParameters.Rtcp)
}

func TestGetProducerRtpParametersMapping_UnsupportedError(t *testing.T) {
	mediaCodecs := []RtpCodecCapability{
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
			Parameters: &RtpParameter{
				RtpH264Parameter: h264profile.RtpH264Parameter{
					PacketizationMode: 1,
					ProfileLevelId:    "640032",
				},
			},
		},
	}

	routerRtpCapabilities, err := GenerateRouterRtpCapabilities(mediaCodecs)
	assert.NoError(t, err)

	rtpParameters := RtpRemoteCapabilities{
		Codecs: []RtpMappingCodec{
			{
				RtpCodecCapability: &RtpCodecCapability{
					Kind:      "video",
					MimeType:  "video/VP8",
					ClockRate: 90000,
					RtcpFeedback: []RtcpFeedback{
						{Type: "nack"},
						{Type: "nack", Parameter: "pli"},
					},
				},
				PayloadType: 120,
			},
		},
		HeaderExtensions: []RtpMappingHeaderExt{},
		Encodings: []RtpMappingEncoding{
			{
				Ssrc: 11111111,
			},
		},
		Rtcp: RtcpConfiguation{
			Cname: "qwerty1234",
		},
	}

	_, err = GetProducerRtpParametersMapping(rtpParameters, routerRtpCapabilities)
	assert.IsType(t, err, NewUnsupportedError(""))
}

func assertJSONEq(t *testing.T, expected, actual interface{}) {
	expectedData, err := json.Marshal(expected)
	assert.NoError(t, err)

	actualData, err := json.Marshal(actual)
	assert.NoError(t, err)

	assert.JSONEq(t, string(expectedData), string(actualData))
}
