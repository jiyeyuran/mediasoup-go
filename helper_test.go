package mediasoup

import "github.com/jiyeyuran/mediasoup-go/h264"

func CreateRouter() *Router {
	router, err := worker.CreateRouter(RouterOptions{
		MediaCodecs: []*RtpCodecCapability{
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
				MimeType:  "video/H264",
				ClockRate: 90000,
				Parameters: RtpCodecSpecificParameters{
					RtpParameter: h264.RtpParameter{
						LevelAsymmetryAllowed: 1,
						PacketizationMode:     1,
						ProfileLevelId:        "4d0032",
					},
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}
	return router
}

func CreateAudioProducer(tranpsort ITransport) *Producer {
	producer, err := tranpsort.Produce(ProducerOptions{
		Kind: MediaKind_Audio,
		RtpParameters: RtpParameters{
			Mid: "AUDIO",
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "audio/opus",
					PayloadType: 111,
					ClockRate:   48000,
					Channels:    2,
					Parameters: RtpCodecSpecificParameters{
						Useinbandfec: 1,
						Usedtx:       1,
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
			Encodings: []RtpEncodingParameters{{Ssrc: 11111111, Dtx: true}},
			Rtcp: RtcpParameters{
				Cname: "audio-1",
			},
		},
		AppData: H{"foo": 1, "bar": "2"},
	})

	if err != nil {
		panic(err)
	}

	return producer
}

func CreateVideoProducer(tranpsort ITransport) *Producer {
	producer, err := tranpsort.Produce(ProducerOptions{
		Kind: MediaKind_Video,
		RtpParameters: RtpParameters{
			Mid: "VIDEO",
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "video/h264",
					PayloadType: 112,
					ClockRate:   90000,
					Parameters: RtpCodecSpecificParameters{
						RtpParameter: h264.RtpParameter{
							PacketizationMode: 1,
							ProfileLevelId:    "4d0032",
						},
					},
					RtcpFeedback: []RtcpFeedback{
						{Type: "nack", Parameter: ""},
						{Type: "nack", Parameter: "pli"},
						{Type: "goog-remb", Parameter: ""},
					},
				},
				{
					MimeType:    "video/rtx",
					PayloadType: 113,
					ClockRate:   90000,
					Parameters:  RtpCodecSpecificParameters{Apt: 112},
				},
			},
			HeaderExtensions: []RtpHeaderExtensionParameters{
				{
					Uri: "urn:ietf:params:rtp-hdrext:sdes:mid",
					Id:  10,
				},
				{
					Uri: "urn:3gpp:video-orientation",
					Id:  13,
				},
			},
			Encodings: []RtpEncodingParameters{
				{Ssrc: 22222222, Rtx: &RtpEncodingRtx{Ssrc: 22222223}},
				{Ssrc: 22222224, Rtx: &RtpEncodingRtx{Ssrc: 22222225}},
				{Ssrc: 22222226, Rtx: &RtpEncodingRtx{Ssrc: 22222227}},
				{Ssrc: 22222228, Rtx: &RtpEncodingRtx{Ssrc: 22222229}},
			},
			Rtcp: RtcpParameters{
				Cname: "video-1",
			},
		},
		AppData: H{"foo": 1, "bar": "2"},
	})

	if err != nil {
		panic(err)
	}

	return producer
}
