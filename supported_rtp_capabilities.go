package mediasoup

import (
	"github.com/jiyeyuran/mediasoup-go/h264"
)

var supportedRtpCapabilities = RtpCapabilities{
	Codecs: []*RtpCodecCapability{
		{
			Kind:      "audio",
			MimeType:  "audio/opus",
			ClockRate: 48000,
			Channels:  2,
			RtcpFeedback: []RtcpFeedback{
				{Type: "transport-cc"},
			},
		},
		{
			Kind:                 "audio",
			MimeType:             "audio/PCMU",
			PreferredPayloadType: 0,
			ClockRate:            8000,
			RtcpFeedback: []RtcpFeedback{
				{Type: "transport-cc"},
			},
		},
		{
			Kind:                 "audio",
			MimeType:             "audio/PCMA",
			PreferredPayloadType: 8,
			ClockRate:            8000,
			RtcpFeedback: []RtcpFeedback{
				{Type: "transport-cc"},
			},
		},
		{
			Kind:      "audio",
			MimeType:  "audio/ISAC",
			ClockRate: 32000,
			RtcpFeedback: []RtcpFeedback{
				{Type: "transport-cc"},
			},
		},
		{
			Kind:      "audio",
			MimeType:  "audio/ISAC",
			ClockRate: 16000,
			RtcpFeedback: []RtcpFeedback{
				{Type: "transport-cc"},
			},
		},
		{
			Kind:                 "audio",
			MimeType:             "audio/G722",
			PreferredPayloadType: 9,
			ClockRate:            8000,
			RtcpFeedback: []RtcpFeedback{
				{Type: "transport-cc"},
			},
		},
		{
			Kind:      "audio",
			MimeType:  "audio/iLBC",
			ClockRate: 8000,
			RtcpFeedback: []RtcpFeedback{
				{Type: "transport-cc"},
			},
		},
		{
			Kind:      "audio",
			MimeType:  "audio/SILK",
			ClockRate: 24000,
			RtcpFeedback: []RtcpFeedback{
				{Type: "transport-cc"},
			},
		},
		{
			Kind:      "audio",
			MimeType:  "audio/SILK",
			ClockRate: 16000,
			RtcpFeedback: []RtcpFeedback{
				{Type: "transport-cc"},
			},
		},
		{
			Kind:      "audio",
			MimeType:  "audio/SILK",
			ClockRate: 12000,
			RtcpFeedback: []RtcpFeedback{
				{Type: "transport-cc"},
			},
		},
		{
			Kind:      "audio",
			MimeType:  "audio/SILK",
			ClockRate: 8000,
			RtcpFeedback: []RtcpFeedback{
				{Type: "transport-cc"},
			},
		},
		{
			Kind:                 "audio",
			MimeType:             "audio/CN",
			PreferredPayloadType: 13,
			ClockRate:            32000,
		},
		{
			Kind:                 "audio",
			MimeType:             "audio/CN",
			PreferredPayloadType: 13,
			ClockRate:            16000,
		},
		{
			Kind:                 "audio",
			MimeType:             "audio/CN",
			PreferredPayloadType: 13,
			ClockRate:            8000,
		},
		{
			Kind:      "audio",
			MimeType:  "audio/telephone-event",
			ClockRate: 48000,
		},
		{
			Kind:      "audio",
			MimeType:  "audio/telephone-event",
			ClockRate: 32000,
		},

		{
			Kind:      "audio",
			MimeType:  "audio/telephone-event",
			ClockRate: 16000,
		},
		{
			Kind:      "audio",
			MimeType:  "audio/telephone-event",
			ClockRate: 8000,
		},
		{
			Kind:      "video",
			MimeType:  "video/VP8",
			ClockRate: 90000,
			RtcpFeedback: []RtcpFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "goog-remb"},
				{Type: "transport-cc"},
			},
		},
		{
			Kind:      "video",
			MimeType:  "video/VP9",
			ClockRate: 90000,
			RtcpFeedback: []RtcpFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "goog-remb"},
				{Type: "transport-cc"},
			},
		},
		{
			Kind:      "video",
			MimeType:  "video/H264",
			ClockRate: 90000,
			Parameters: RtpCodecSpecificParameters{
				RtpParameter: h264.RtpParameter{
					PacketizationMode:     1,
					LevelAsymmetryAllowed: 1,
				},
			},
			RtcpFeedback: []RtcpFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "goog-remb"},
				{Type: "transport-cc"},
			},
		},
		{
			Kind:      "video",
			MimeType:  "video/H264",
			ClockRate: 90000,
			Parameters: RtpCodecSpecificParameters{
				RtpParameter: h264.RtpParameter{
					PacketizationMode:     0,
					LevelAsymmetryAllowed: 1,
				},
			},
			RtcpFeedback: []RtcpFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "goog-remb"},
				{Type: "transport-cc"},
			},
		},
		{
			Kind:      "video",
			MimeType:  "video/H265",
			ClockRate: 90000,
			Parameters: RtpCodecSpecificParameters{
				RtpParameter: h264.RtpParameter{
					PacketizationMode:     1,
					LevelAsymmetryAllowed: 1,
				},
			},
			RtcpFeedback: []RtcpFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "goog-remb"},
				{Type: "transport-cc"},
			},
		},
		{
			Kind:      "video",
			MimeType:  "video/H265",
			ClockRate: 90000,
			Parameters: RtpCodecSpecificParameters{
				RtpParameter: h264.RtpParameter{
					PacketizationMode:     0,
					LevelAsymmetryAllowed: 1,
				},
			},
			RtcpFeedback: []RtcpFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "goog-remb"},
				{Type: "transport-cc"},
			},
		},
	},
	HeaderExtensions: []*RtpHeaderExtension{
		{
			Kind:             "audio",
			Uri:              "urn:ietf:params:rtp-hdrext:sdes:mid",
			PreferredId:      1,
			PreferredEncrypt: false,
			Direction:        Direction_Sendrecv,
		},
		{
			Kind:             "video",
			Uri:              "urn:ietf:params:rtp-hdrext:sdes:mid",
			PreferredId:      1,
			PreferredEncrypt: false,
			Direction:        Direction_Sendrecv,
		},
		{
			Kind:             "video",
			Uri:              "urn:ietf:params:rtp-hdrext:sdes:rtp-stream-id",
			PreferredId:      2,
			PreferredEncrypt: false,
			Direction:        Direction_Recvonly,
		},
		{
			Kind:             "video",
			Uri:              "urn:ietf:params:rtp-hdrext:sdes:repaired-rtp-stream-id",
			PreferredId:      3,
			PreferredEncrypt: false,
			Direction:        Direction_Recvonly,
		},
		{
			Kind:             "audio",
			Uri:              "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time",
			PreferredId:      4,
			PreferredEncrypt: false,
			Direction:        Direction_Sendrecv,
		},
		{
			Kind:             "video",
			Uri:              "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time",
			PreferredId:      4,
			PreferredEncrypt: false,
			Direction:        Direction_Sendrecv,
		},
		// NOTE: For audio we just enable transport-wide-cc-01 when receiving media.
		{
			Kind:             "audio",
			Uri:              "http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01",
			PreferredId:      5,
			PreferredEncrypt: false,
			Direction:        Direction_Recvonly,
		},
		{
			Kind:             "video",
			Uri:              "http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01",
			PreferredId:      5,
			PreferredEncrypt: false,
			Direction:        Direction_Sendrecv,
		},
		// NOTE: Remove this once framemarking draft becomes RFC.
		{
			Kind:             "video",
			Uri:              "http://tools.ietf.org/html/draft-ietf-avtext-framemarking-07",
			PreferredId:      6,
			PreferredEncrypt: false,
			Direction:        Direction_Sendrecv,
		},
		{
			Kind:             "video",
			Uri:              "urn:ietf:params:rtp-hdrext:framemarking",
			PreferredId:      7,
			PreferredEncrypt: false,
			Direction:        Direction_Sendrecv,
		},
		{
			Kind:             "audio",
			Uri:              "urn:ietf:params:rtp-hdrext:ssrc-audio-level",
			PreferredId:      10,
			PreferredEncrypt: false,
			Direction:        Direction_Sendrecv,
		},
		{
			Kind:             "video",
			Uri:              "urn:3gpp:video-orientation",
			PreferredId:      11,
			PreferredEncrypt: false,
			Direction:        Direction_Sendrecv,
		},
		{
			Kind:             "video",
			Uri:              "urn:ietf:params:rtp-hdrext:toffset",
			PreferredId:      12,
			PreferredEncrypt: false,
			Direction:        Direction_Sendrecv,
		},
	},
}

func init() {
	if err := validateRtpCapabilities(&supportedRtpCapabilities); err != nil {
		panic(err)
	}
}

func GetSupportedRtpCapabilities() (rtpCapabilities RtpCapabilities) {
	clone(supportedRtpCapabilities, &rtpCapabilities)

	return
}
