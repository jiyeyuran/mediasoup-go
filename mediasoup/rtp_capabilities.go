package mediasoup

import (
	"github.com/jinzhu/copier"
	h264 "github.com/jiyeyuran/mediasoup-go/mediasoup/h264profile"
)

type RtpCapabilities struct {
	Codecs           []RtpCodecCapability `json:"codecs,omitempty"`
	HeaderExtensions []RtpHeaderExtension `json:"headerExtensions,omitempty"`
	FecMechanisms    []string             `json:"fecMechanisms,omitempty"`
}

type RtpParameters struct {
	Mid              string               `json:"mid,omitempty"` // AUDIO, VIDEO
	Codecs           []RtpCodecCapability `json:"codecs,omitempty"`
	HeaderExtensions []RtpHeaderExtension `json:"headerExtensions,omitempty"`
	Encodings        []RtpEncoding        `json:"encodings,omitempty"`
	Rtcp             RtcpConfiguation     `json:"rtcp,omitempty"`
}

type RtpMappingParameters struct {
	Codecs           []RtpMappingCodec     `json:"codecs,omitempty"`
	HeaderExtensions []RtpMappingHeaderExt `json:"headerExtensions,omitempty"`
	Encodings        []RtpMappingEncoding  `json:"encodings,omitempty"`
}

type RtpMappingCodec struct {
	PayloadType       int `json:"payloadType,omitempty"`
	MappedPayloadType int `json:"mappedPayloadType,omitempty"`
}

type RtpMappingHeaderExt struct {
	Id       int `json:"id,omitempty"`
	MappedId int `json:"mappedId,omitempty"`
}

type RtpMappingEncoding struct {
	Rid        string `json:"rid,omitempty"`
	Ssrc       uint32 `json:"ssrc,omitempty"`
	MappedSsrc uint32 `json:"mappedSsrc,omitempty"`
}

type RtpCodecCapability struct {
	Kind                 string             `json:"kind,omitempty"`
	MimeType             string             `json:"mimeType,omitempty"`
	ClockRate            int                `json:"clockRate,omitempty"`
	Channels             int                `json:"channels,omitempty"`
	PayloadType          int                `json:"payloadType,omitempty"`
	PreferredPayloadType int                `json:"preferredPayloadType,omitempty"`
	Parameters           *RtpCodecParameter `json:"parameters,omitempty"`
	RtcpFeedback         []RtcpFeedback     `json:"rtcpFeedback,omitempty"`
}

type RtcpFeedback struct {
	Type      string `json:"type,omitempty"`
	Parameter string `json:"parameter,omitempty"`
}

type RtpCodecParameter struct {
	h264.RtpH264Parameter     // used by h264 codec
	Apt                   int `json:"apt,omitempty"`                    // used by rtx codec
	Useinbandfec          int `json:"useinbandfec,omitempty"`           // used by opus
	Minptime              int `json:"minptime,omitempty"`               // used by opus
	XGoogleStartBitrate   int `json:"x-google-start-bitrate,omitempty"` // used by video
}

type RtpHeaderExtension struct {
	Id               int    `json:"id,omitempty"`
	Kind             string `json:"kind,omitempty"`
	Uri              string `json:"uri,omitempty"`
	PreferredId      int    `json:"preferredId,omitempty"`
	PreferredEncrypt bool   `json:"preferredEncrypt,omitempty"`
}

type RtpEncoding struct {
	Rid              string       `json:"rid,omitempty"`
	Ssrc             uint32       `json:"ssrc,omitempty"`
	MaxBitrate       uint32       `json:"maxBitrate,omitempty"`
	CodecPayloadType uint32       `json:"codecPayloadType,omitempty"`
	Rtx              *RtpEncoding `json:"rtx,omitempty"`
}

type RtcpConfiguation struct {
	Cname       string `json:"cname,omitempty"`
	ReducedSize bool   `json:"reducedSize,omitempty"`
	Mux         *bool  `json:"mux,omitempty"`
}

var supportedRtpCapabilities = RtpCapabilities{
	Codecs: []RtpCodecCapability{
		{
			Kind:      "audio",
			MimeType:  "audio/opus",
			ClockRate: 48000,
			Channels:  2,
		},
		{
			Kind:                 "audio",
			MimeType:             "audio/PCMU",
			PreferredPayloadType: 0,
			ClockRate:            8000,
		},
		{
			Kind:                 "audio",
			MimeType:             "audio/PCMA",
			PreferredPayloadType: 8,
			ClockRate:            8000,
		},
		{
			Kind:      "audio",
			MimeType:  "audio/ISAC",
			ClockRate: 32000,
		},
		{
			Kind:      "audio",
			MimeType:  "audio/ISAC",
			ClockRate: 16000,
		},
		{
			Kind:                 "audio",
			MimeType:             "audio/G722",
			PreferredPayloadType: 9,
			ClockRate:            8000,
		},
		{
			Kind:      "audio",
			MimeType:  "audio/iLBC",
			ClockRate: 8000,
		},
		{
			Kind:      "audio",
			MimeType:  "audio/SILK",
			ClockRate: 24000,
		},
		{
			Kind:      "audio",
			MimeType:  "audio/SILK",
			ClockRate: 16000,
		},
		{
			Kind:      "audio",
			MimeType:  "audio/SILK",
			ClockRate: 12000,
		},
		{
			Kind:      "audio",
			MimeType:  "audio/SILK",
			ClockRate: 8000,
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
			},
		},
		{
			Kind:      "video",
			MimeType:  "video/H264",
			ClockRate: 90000,
			Parameters: &RtpCodecParameter{
				RtpH264Parameter: h264.RtpH264Parameter{
					PacketizationMode:     1,
					LevelAsymmetryAllowed: 1,
				},
			},
			RtcpFeedback: []RtcpFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "goog-remb"},
			},
		},
		{
			Kind:      "video",
			MimeType:  "video/H264",
			ClockRate: 90000,
			Parameters: &RtpCodecParameter{
				RtpH264Parameter: h264.RtpH264Parameter{
					PacketizationMode:     0,
					LevelAsymmetryAllowed: 1,
				},
			},
			RtcpFeedback: []RtcpFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "goog-remb"},
			},
		},
		{
			Kind:      "video",
			MimeType:  "video/H265",
			ClockRate: 90000,
			Parameters: &RtpCodecParameter{
				RtpH264Parameter: h264.RtpH264Parameter{
					PacketizationMode:     1,
					LevelAsymmetryAllowed: 1,
				},
			},
			RtcpFeedback: []RtcpFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "goog-remb"},
			},
		},
		{
			Kind:      "video",
			MimeType:  "video/H265",
			ClockRate: 90000,
			Parameters: &RtpCodecParameter{
				RtpH264Parameter: h264.RtpH264Parameter{
					PacketizationMode:     0,
					LevelAsymmetryAllowed: 1,
				},
			},
			RtcpFeedback: []RtcpFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "goog-remb"},
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
			Kind:             "audio",
			Uri:              "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time",
			PreferredId:      3,
			PreferredEncrypt: false,
		},
		{
			Kind:             "video",
			Uri:              "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time",
			PreferredId:      3,
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
		{
			Kind:             "video",
			Uri:              "urn:ietf:params:rtp-hdrext:sdes:repaired-rtp-stream-id",
			PreferredId:      7,
			PreferredEncrypt: false,
		},
	},
}

func GetSupportedRtpCapabilities() (rtpCapabilities RtpCapabilities) {
	copier.Copy(&rtpCapabilities, &supportedRtpCapabilities)

	return
}
