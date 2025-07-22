package mediasoup

var supportedRtpCapabilities = RtpCapabilities{
	Codecs: []*RtpCodecCapability{
		{
			Kind:      MediaKindAudio,
			MimeType:  "audio/opus",
			ClockRate: 48000,
			Channels:  2,
			RtcpFeedback: []*RtcpFeedback{
				{Type: "nack"},
				{Type: "transport-cc"},
			},
		},
		{
			Kind:      MediaKindAudio,
			MimeType:  "audio/multiopus",
			ClockRate: 48000,
			Channels:  4,
			Parameters: RtpCodecSpecificParameters{
				ChannelMapping: "0,1,2,3",
				NumStreams:     2,
				CoupledStreams: 2,
			},
			RtcpFeedback: []*RtcpFeedback{
				{Type: "nack"},
				{Type: "transport-cc"},
			},
		},
		{
			Kind:      MediaKindAudio,
			MimeType:  "audio/multiopus",
			ClockRate: 48000,
			Channels:  6,
			Parameters: RtpCodecSpecificParameters{
				ChannelMapping: "0,4,1,2,3,5",
				NumStreams:     4,
				CoupledStreams: 2,
			},
			RtcpFeedback: []*RtcpFeedback{
				{Type: "nack"},
				{Type: "transport-cc"},
			},
		},
		{
			Kind:      MediaKindAudio,
			MimeType:  "audio/multiopus",
			ClockRate: 48000,
			Channels:  8,
			Parameters: RtpCodecSpecificParameters{
				ChannelMapping: "0,6,1,2,3,4,5,7",
				NumStreams:     5,
				CoupledStreams: 3,
			},
			RtcpFeedback: []*RtcpFeedback{
				{Type: "nack"},
				{Type: "transport-cc"},
			},
		},
		{
			Kind:                 MediaKindAudio,
			MimeType:             "audio/PCMU",
			PreferredPayloadType: 0,
			ClockRate:            8000,
			RtcpFeedback: []*RtcpFeedback{
				{Type: "transport-cc"},
			},
		},
		{
			Kind:                 MediaKindAudio,
			MimeType:             "audio/PCMA",
			PreferredPayloadType: 8,
			ClockRate:            8000,
			RtcpFeedback: []*RtcpFeedback{
				{Type: "transport-cc"},
			},
		},
		{
			Kind:      MediaKindAudio,
			MimeType:  "audio/ISAC",
			ClockRate: 32000,
			RtcpFeedback: []*RtcpFeedback{
				{Type: "transport-cc"},
			},
		},
		{
			Kind:      MediaKindAudio,
			MimeType:  "audio/ISAC",
			ClockRate: 16000,
			RtcpFeedback: []*RtcpFeedback{
				{Type: "transport-cc"},
			},
		},
		{
			Kind:                 MediaKindAudio,
			MimeType:             "audio/G722",
			PreferredPayloadType: 9,
			ClockRate:            8000,
			RtcpFeedback: []*RtcpFeedback{
				{Type: "transport-cc"},
			},
		},
		{
			Kind:      MediaKindAudio,
			MimeType:  "audio/iLBC",
			ClockRate: 8000,
			RtcpFeedback: []*RtcpFeedback{
				{Type: "transport-cc"},
			},
		},
		{
			Kind:      MediaKindAudio,
			MimeType:  "audio/SILK",
			ClockRate: 24000,
			RtcpFeedback: []*RtcpFeedback{
				{Type: "transport-cc"},
			},
		},
		{
			Kind:      MediaKindAudio,
			MimeType:  "audio/SILK",
			ClockRate: 16000,
			RtcpFeedback: []*RtcpFeedback{
				{Type: "transport-cc"},
			},
		},
		{
			Kind:      MediaKindAudio,
			MimeType:  "audio/SILK",
			ClockRate: 12000,
			RtcpFeedback: []*RtcpFeedback{
				{Type: "transport-cc"},
			},
		},
		{
			Kind:      MediaKindAudio,
			MimeType:  "audio/SILK",
			ClockRate: 8000,
			RtcpFeedback: []*RtcpFeedback{
				{Type: "transport-cc"},
			},
		},
		{
			Kind:                 MediaKindAudio,
			MimeType:             "audio/CN",
			PreferredPayloadType: 13,
			ClockRate:            32000,
		},
		{
			Kind:                 MediaKindAudio,
			MimeType:             "audio/CN",
			PreferredPayloadType: 13,
			ClockRate:            16000,
		},
		{
			Kind:                 MediaKindAudio,
			MimeType:             "audio/CN",
			PreferredPayloadType: 13,
			ClockRate:            8000,
		},
		{
			Kind:      MediaKindAudio,
			MimeType:  "audio/telephone-event",
			ClockRate: 48000,
		},
		{
			Kind:      MediaKindAudio,
			MimeType:  "audio/telephone-event",
			ClockRate: 32000,
		},

		{
			Kind:      MediaKindAudio,
			MimeType:  "audio/telephone-event",
			ClockRate: 16000,
		},
		{
			Kind:      MediaKindAudio,
			MimeType:  "audio/telephone-event",
			ClockRate: 8000,
		},
		{
			Kind:      MediaKindVideo,
			MimeType:  "video/VP8",
			ClockRate: 90000,
			RtcpFeedback: []*RtcpFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "goog-remb"},
				{Type: "transport-cc"},
			},
		},
		{
			Kind:      MediaKindVideo,
			MimeType:  "video/VP9",
			ClockRate: 90000,
			RtcpFeedback: []*RtcpFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "goog-remb"},
				{Type: "transport-cc"},
			},
		},
		{
			Kind:      MediaKindVideo,
			MimeType:  "video/H264",
			ClockRate: 90000,
			Parameters: RtpCodecSpecificParameters{
				LevelAsymmetryAllowed: 1,
			},
			RtcpFeedback: []*RtcpFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "goog-remb"},
				{Type: "transport-cc"},
			},
		},
		{
			Kind:      MediaKindVideo,
			MimeType:  "video/AV1",
			ClockRate: 90000,
			RtcpFeedback: []*RtcpFeedback{
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
			Kind:             MediaKindAudio,
			Uri:              "urn:ietf:params:rtp-hdrext:sdes:mid",
			PreferredId:      1,
			PreferredEncrypt: false,
			Direction:        MediaDirectionSendrecv,
		},
		{
			Kind:             MediaKindVideo,
			Uri:              "urn:ietf:params:rtp-hdrext:sdes:mid",
			PreferredId:      1,
			PreferredEncrypt: false,
			Direction:        MediaDirectionSendrecv,
		},
		{
			Kind:             MediaKindVideo,
			Uri:              "urn:ietf:params:rtp-hdrext:sdes:rtp-stream-id",
			PreferredId:      2,
			PreferredEncrypt: false,
			Direction:        MediaDirectionRecvonly,
		},
		{
			Kind:             MediaKindVideo,
			Uri:              "urn:ietf:params:rtp-hdrext:sdes:repaired-rtp-stream-id",
			PreferredId:      3,
			PreferredEncrypt: false,
			Direction:        MediaDirectionRecvonly,
		},
		{
			Kind:             MediaKindAudio,
			Uri:              "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time",
			PreferredId:      4,
			PreferredEncrypt: false,
			Direction:        MediaDirectionSendrecv,
		},
		{
			Kind:             MediaKindVideo,
			Uri:              "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time",
			PreferredId:      4,
			PreferredEncrypt: false,
			Direction:        MediaDirectionSendrecv,
		},
		// NOTE: For audio we just enable transport-wide-cc-01 when receiving media.
		{
			Kind:             MediaKindAudio,
			Uri:              "http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01",
			PreferredId:      5,
			PreferredEncrypt: false,
			Direction:        MediaDirectionRecvonly,
		},
		{
			Kind:             MediaKindVideo,
			Uri:              "http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01",
			PreferredId:      5,
			PreferredEncrypt: false,
			Direction:        MediaDirectionSendrecv,
		},
		{
			Kind:             MediaKindVideo,
			Uri:              "https://aomediacodec.github.io/av1-rtp-spec/#dependency-descriptor-rtp-header-extension",
			PreferredId:      8,
			PreferredEncrypt: false,
			Direction:        MediaDirectionRecvonly,
		},
		{
			Kind:             MediaKindAudio,
			Uri:              "urn:ietf:params:rtp-hdrext:ssrc-audio-level",
			PreferredId:      10,
			PreferredEncrypt: false,
			Direction:        MediaDirectionSendrecv,
		},
		{
			Kind:             MediaKindVideo,
			Uri:              "urn:3gpp:video-orientation",
			PreferredId:      11,
			PreferredEncrypt: false,
			Direction:        MediaDirectionSendrecv,
		},
		{
			Kind:             MediaKindVideo,
			Uri:              "urn:ietf:params:rtp-hdrext:toffset",
			PreferredId:      12,
			PreferredEncrypt: false,
			Direction:        MediaDirectionSendrecv,
		},
		{
			Kind:             MediaKindVideo,
			Uri:              "http://www.webrtc.org/experiments/rtp-hdrext/abs-capture-time",
			PreferredId:      13,
			PreferredEncrypt: false,
			Direction:        MediaDirectionSendrecv,
		},
		{
			Kind:             MediaKindAudio,
			Uri:              "http://www.webrtc.org/experiments/rtp-hdrext/abs-capture-time",
			PreferredId:      13,
			PreferredEncrypt: false,
			Direction:        MediaDirectionSendrecv,
		},
		{
			Kind:             MediaKindAudio,
			Uri:              "http://www.webrtc.org/experiments/rtp-hdrext/playout-delay",
			PreferredId:      14,
			PreferredEncrypt: false,
			Direction:        MediaDirectionSendrecv,
		},
		{
			Kind:             MediaKindVideo,
			Uri:              "http://www.webrtc.org/experiments/rtp-hdrext/playout-delay",
			PreferredId:      14,
			PreferredEncrypt: false,
			Direction:        MediaDirectionSendrecv,
		},
	},
}

func init() {
	if err := validateRtpCapabilities(&supportedRtpCapabilities); err != nil {
		panic(err)
	}
}

func GetSupportedRtpCapabilities() RtpCapabilities {
	return clone(supportedRtpCapabilities)
}
