package main

import (
	"encoding/json"
	"log"

	"github.com/jiyeyuran/mediasoup-go"
	"github.com/jiyeyuran/mediasoup-go/h264"
)

var logger = mediasoup.NewLogger("ExampleApp")

var consumerDeviceCapabilities = mediasoup.RtpCapabilities{
	Codecs: []*mediasoup.RtpCodecCapability{
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
			Parameters: mediasoup.RtpCodecSpecificParameters{
				RtpParameter: h264.RtpParameter{
					LevelAsymmetryAllowed: 1,
					PacketizationMode:     Uint8(1),
					ProfileLevelId:        "4d0032",
				},
			},
			RtcpFeedback: []mediasoup.RtcpFeedback{
				{Type: "nack", Parameter: ""},
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "goog-remb", Parameter: ""},
			},
		},
		{
			MimeType:             "video/rtx",
			Kind:                 "video",
			PreferredPayloadType: 102,
			ClockRate:            90000,
			Parameters: mediasoup.RtpCodecSpecificParameters{
				Apt: 101,
			},
		},
	},
}

func main() {
	worker, err := mediasoup.NewWorker()
	if err != nil {
		panic(err)
	}
	worker.On("died", func(err error) {
		log.Printf("died: %s", err)
	})

	dump, _ := worker.Dump()
	log.Printf("dump: %+v", dump)

	usage, err := worker.GetResourceUsage()
	if err != nil {
		panic(err)
	}
	data, _ := json.Marshal(usage)
	log.Printf("usage: %s", data)

	router, err := worker.CreateRouter(mediasoup.RouterOptions{
		MediaCodecs: []*mediasoup.RtpCodecCapability{
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
				Parameters: mediasoup.RtpCodecSpecificParameters{
					RtpParameter: h264.RtpParameter{
						LevelAsymmetryAllowed: 1,
						PacketizationMode:     Uint8(1),
						ProfileLevelId:        "4d0032",
					},
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	sendTransport, err := router.CreateWebRtcTransport(mediasoup.WebRtcTransportOptions{
		ListenIps: []mediasoup.TransportListenIp{
			{Ip: "0.0.0.0", AnnouncedIp: "192.168.1.101"}, // AnnouncedIp is optional
		},
	})
	if err != nil {
		panic(err)
	}

	producer, err := sendTransport.Produce(mediasoup.ProducerOptions{
		Kind: mediasoup.MediaKind_Audio,
		RtpParameters: mediasoup.RtpParameters{
			Mid: "VIDEO",
			Codecs: []*mediasoup.RtpCodecParameters{
				{
					MimeType:    "video/h264",
					PayloadType: 112,
					ClockRate:   90000,
					Parameters: mediasoup.RtpCodecSpecificParameters{
						RtpParameter: h264.RtpParameter{
							PacketizationMode: Uint8(1),
							ProfileLevelId:    "4d0032",
						},
					},
					RtcpFeedback: []mediasoup.RtcpFeedback{
						{Type: "nack", Parameter: ""},
						{Type: "nack", Parameter: "pli"},
						{Type: "goog-remb", Parameter: ""},
					},
				},
				{
					MimeType:    "video/rtx",
					PayloadType: 113,
					ClockRate:   90000,
					Parameters:  mediasoup.RtpCodecSpecificParameters{Apt: 112},
				},
			},
			HeaderExtensions: []mediasoup.RtpHeaderExtensionParameters{
				{
					Uri: "urn:ietf:params:rtp-hdrext:sdes:mid",
					Id:  10,
				},
				{
					Uri: "urn:3gpp:video-orientation",
					Id:  13,
				},
			},
			Encodings: []mediasoup.RtpEncodingParameters{
				{Ssrc: 22222222, Rtx: &mediasoup.RtpEncodingRtx{Ssrc: 22222223}},
				{Ssrc: 22222224, Rtx: &mediasoup.RtpEncodingRtx{Ssrc: 22222225}},
				{Ssrc: 22222226, Rtx: &mediasoup.RtpEncodingRtx{Ssrc: 22222227}},
				{Ssrc: 22222228, Rtx: &mediasoup.RtpEncodingRtx{Ssrc: 22222229}},
			},
			Rtcp: mediasoup.RtcpParameters{
				Cname: "video-1",
			},
		},
		AppData: mediasoup.H{"foo": 1, "bar": "2"},
	})
	if err != nil {
		panic(err)
	}

	recvTransport, err := router.CreateWebRtcTransport(mediasoup.WebRtcTransportOptions{
		ListenIps: []mediasoup.TransportListenIp{
			{Ip: "0.0.0.0", AnnouncedIp: "192.168.1.101"}, // AnnouncedIp is optional
		},
	})
	if err != nil {
		panic(err)
	}

	producer.Pause()

	// TODO: sent back producer id to client

	consumer, err := recvTransport.Consume(mediasoup.ConsumerOptions{
		ProducerId:      producer.Id(),
		RtpCapabilities: consumerDeviceCapabilities,
		AppData:         mediasoup.H{"baz": "LOL"},
	})
	if err != nil {
		panic(err)
	}

	consumerDump, _ := consumer.Dump()
	data, _ = json.Marshal(consumerDump)
	log.Printf("consumer, %s", data)

	wait := make(chan struct{})
	<-wait
}
