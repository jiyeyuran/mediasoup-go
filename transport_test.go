package mediasoup

import (
	"regexp"
	"slices"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createWebRtcTransport(router *Router, options ...func(o *WebRtcTransportOptions)) *Transport {
	if router == nil {
		router = createRouter(newTestWorker())
	}
	o := &WebRtcTransportOptions{
		ListenInfos: []TransportListenInfo{
			{IP: "127.0.0.1", AnnouncedAddress: "9.9.9.1", Port: 0},
		},
	}
	for _, option := range options {
		option(o)
	}
	transport, err := router.CreateWebRtcTransport(o)
	if err != nil {
		panic(err)
	}
	return transport
}

func createPlainTransport(router *Router, options ...func(o *PlainTransportOptions)) *Transport {
	if router == nil {
		router = createRouter(newTestWorker())
	}
	o := &PlainTransportOptions{
		ListenInfo: TransportListenInfo{
			IP:               "127.0.0.1",
			AnnouncedAddress: "4.4.4.4",
		},
	}
	for _, option := range options {
		option(o)
	}
	transport, err := router.CreatePlainTransport(o)
	if err != nil {
		panic(err)
	}
	return transport
}

func createDirectTransport(router *Router) *Transport {
	if router == nil {
		router = createRouter(newTestWorker())
	}
	transport, err := router.CreateDirectTransport(nil)
	if err != nil {
		panic(err)
	}
	return transport
}

func TestTransportClose(t *testing.T) {
	t.Run("close normally", func(t *testing.T) {
		m := MockedHandler{}
		defer m.AssertExpectations(t)

		m.On("OnClose").Times(1)

		router := createRouter(newTestWorker())
		transport := createWebRtcTransport(router)

		transport.OnClose(m.OnClose)
		assert.NoError(t, transport.Close())

		dump, _ := router.Dump()
		assert.Empty(t, dump.TransportIds)
		assert.True(t, transport.Closed())

		// methods are rejected if closed
		_, err := transport.GetStats()
		assert.Error(t, err)
	})

	t.Run("router closed", func(t *testing.T) {
		m := MockedHandler{}
		defer m.AssertExpectations(t)

		m.On("OnClose").Times(1)

		router := createRouter(newTestWorker())
		transport := createWebRtcTransport(router)
		transport.OnClose(m.OnClose)
		router.Close()
	})
}

func TestTransportDump(t *testing.T) {
	transport := createWebRtcTransport(nil)
	dump, err := transport.Dump()
	assert.NoError(t, err)
	assert.Equal(t, transport.Id(), dump.Id)
}

func TestTransportGetStats(t *testing.T) {
	transport := createWebRtcTransport(nil)
	stats, err := transport.GetStats()
	assert.NoError(t, err)
	baseStats := stats.BaseTransportStats
	assert.Equal(t, "webrtc-transport", baseStats.Type)
	assert.NotZero(t, baseStats.Timestamp)
	baseStats.Type = ""
	baseStats.Timestamp = 0
	baseStats.TransportId = ""
	assert.Empty(t, baseStats)

	webrtcStats := stats.WebRtcTransportStats
	assert.EqualValues(t, "controlled", webrtcStats.IceRole)
	assert.EqualValues(t, "new", webrtcStats.IceState)
	assert.EqualValues(t, "new", webrtcStats.DtlsState)
	assert.Empty(t, webrtcStats.IceSelectedTuple)

	transport = createPlainTransport(nil)
	stats, _ = transport.GetStats()
	assert.Equal(t, "plain-transport", stats.Type)
	assert.NotZero(t, stats.Timestamp)
	assert.Equal(t, "4.4.4.4", stats.PlainTransportStats.Tuple.LocalAddress)
	assert.NotZero(t, stats.PlainTransportStats.Tuple.LocalPort)

	transport = createDirectTransport(nil)
	stats, _ = transport.GetStats()
	assert.Equal(t, "direct-transport", stats.Type)
	assert.NotZero(t, stats.Timestamp)
}

func TestTransportConnect(t *testing.T) {
	transport := createWebRtcTransport(nil)
	dtlsRemoteParameters := DtlsParameters{
		Fingerprints: []DtlsFingerprint{
			{
				Algorithm: "sha-256",
				Value:     "82:5A:68:3D:36:C3:0A:DE:AF:E7:32:43:D2:88:83:57:AC:2D:65:E5:80:C4:B6:FB:AF:1A:A0:21:9F:6D:0C:AD",
			},
		},
		Role: DtlsRoleClient,
	}
	err := transport.Connect(&TransportConnectOptions{
		DtlsParameters: &dtlsRemoteParameters,
	})
	assert.NoError(t, err)

	t.Logf("transport type: %s", transport.Type())

	err = transport.Connect(&TransportConnectOptions{
		DtlsParameters: &dtlsRemoteParameters,
	})
	assert.Error(t, err)
	assert.EqualValues(t, "server", transport.Data().DtlsParameters.Role)
}

func TestTransportRestartIce(t *testing.T) {
	transport := createWebRtcTransport(nil)
	previousIceUsernameFragment := transport.Data().IceParameters.UsernameFragment
	previousIcePassword := transport.Data().IceParameters.Password
	iceParameters, err := transport.RestartIce()
	assert.NoError(t, err)
	assert.NotEmpty(t, iceParameters.UsernameFragment)
	assert.NotEmpty(t, iceParameters.Password)
	assert.True(t, iceParameters.IceLite)
	assert.NotEmpty(t, transport.Data().IceParameters.UsernameFragment)
	assert.NotEmpty(t, transport.Data().IceParameters.Password)
	assert.NotEqual(t, transport.Data().IceParameters.UsernameFragment, previousIceUsernameFragment)
	assert.NotEqual(t, transport.Data().IceParameters.Password, previousIcePassword)

	transport = createDirectTransport(nil)
	_, err = transport.RestartIce()
	assert.Error(t, err)
}

func TestTransportTestSetMaxIncomingBitrate(t *testing.T) {
	transport := createWebRtcTransport(nil)
	err := transport.SetMaxIncomingBitrate(100000)
	assert.NoError(t, err)

	transport = createDirectTransport(nil)
	err = transport.SetMaxIncomingBitrate(100000)
	assert.Error(t, err)
}

func TestTransportSendRtcp(t *testing.T) {
	transport := createDirectTransport(nil)
	err := transport.SendRtcp([]byte("test"))
	assert.NoError(t, err)
}

func TestTransportEnableTraceEvent(t *testing.T) {
	transport := createDirectTransport(nil)
	err := transport.EnableTraceEvent([]TransportTraceEventType{"foo", "probation"})
	assert.NoError(t, err)
	data, _ := transport.Dump()
	assert.Equal(t, []TransportTraceEventType{"probation"}, data.TraceEventTypes)

	err = transport.EnableTraceEvent(nil)
	assert.NoError(t, err)
	data, _ = transport.Dump()
	assert.Empty(t, data.TraceEventTypes)

	err = transport.EnableTraceEvent([]TransportTraceEventType{"probation", "FOO", "bwe", "BAR"})
	assert.NoError(t, err)
	data, _ = transport.Dump()
	assert.Equal(t, []TransportTraceEventType{"probation", "bwe"}, data.TraceEventTypes)

	err = transport.EnableTraceEvent(nil)
	assert.NoError(t, err)
	data, _ = transport.Dump()
	assert.Empty(t, data.TraceEventTypes)
}

func TestTransportProduce(t *testing.T) {
	router := createRouter(nil)
	transport := createWebRtcTransport(router)
	aproducer := createAudioProducer(transport)
	vproducer := createVideoProducer(transport)

	assert.NotEmpty(t, aproducer.Id())
	assert.False(t, aproducer.Closed())
	assert.Equal(t, MediaKindAudio, aproducer.Kind())
	assert.NotEmpty(t, aproducer.RtpParameters())
	assert.Equal(t, ProducerSimple, aproducer.Type())
	assert.Equal(t, ProducerSimulcast, vproducer.Type())

	// Private API.
	assert.NotEmpty(t, aproducer.ConsumableRtpParameters())
	assert.False(t, aproducer.Paused())
	assert.Empty(t, aproducer.Score())
	assert.Equal(t, H{"foo": 1, "bar": "2"}, aproducer.AppData())

	dump, _ := transport.Dump()
	assert.ElementsMatch(t, []string{aproducer.Id(), vproducer.Id()}, dump.ProducerIds)

	routerDump, _ := router.Dump()
	assert.True(t, slices.ContainsFunc(
		routerDump.MapProducerIdConsumerIds,
		func(kvs KeyValues[string, string]) bool {
			return kvs.Key == aproducer.Id()
		}),
	)
	assert.Empty(t, routerDump.MapConsumerIdProducerId)

	producer := router.GetProducerById(aproducer.Id())
	assert.Equal(t, aproducer, producer)
	producer = router.GetProducerById(vproducer.Id())
	assert.Equal(t, vproducer, producer)
}

func TestTransportProduceError(t *testing.T) {
	transport := createWebRtcTransport(nil)

	t.Run("wrong media kind", func(t *testing.T) {
		_, err := transport.Produce(&ProducerOptions{
			Kind: "chicken",
		})
		assert.Error(t, err)
	})

	t.Run("missing or empty rtpParameters.codecs", func(t *testing.T) {
		_, err := transport.Produce(&ProducerOptions{
			Kind: MediaKindAudio,
			RtpParameters: &RtpParameters{
				Encodings: []*RtpEncodingParameters{
					{Ssrc: 1111},
				},
				Rtcp: &RtcpParameters{Cname: "qwerty"},
			},
		})
		assert.Error(t, err)
	})

	t.Run("missing or empty rtpParameters.encodings", func(t *testing.T) {
		_, err := transport.Produce(&ProducerOptions{
			Kind: MediaKindVideo,
			RtpParameters: &RtpParameters{
				Codecs: []*RtpCodecParameters{
					{
						MimeType:    "video/h264",
						PayloadType: 112,
						ClockRate:   90000,
						Parameters: RtpCodecSpecificParameters{
							PacketizationMode: 1,
							ProfileLevelId:    "4d0032",
						},
					},
					{
						MimeType:    "video/rtx",
						PayloadType: 113,
						ClockRate:   90000,
						Parameters:  RtpCodecSpecificParameters{Apt: 112},
					},
				},
				Rtcp: &RtcpParameters{
					Cname: "qwerty",
				},
			},
		})
		assert.Error(t, err)
	})

	t.Run("wrong apt in RTX codec", func(t *testing.T) {
		_, err := transport.Produce(&ProducerOptions{
			Kind: "video",
			RtpParameters: &RtpParameters{
				Codecs: []*RtpCodecParameters{
					{
						MimeType:    "video/h264",
						PayloadType: 112,
						ClockRate:   90000,
						Parameters: RtpCodecSpecificParameters{
							PacketizationMode: 1,
							ProfileLevelId:    "4d0032",
						},
					},
					{
						MimeType:    "video/rtx",
						PayloadType: 113,
						ClockRate:   90000,
						Parameters:  RtpCodecSpecificParameters{Apt: 111},
					},
				},
				Encodings: []*RtpEncodingParameters{
					{Ssrc: 6666, Rtx: &RtpEncodingRtx{Ssrc: 6667}},
				},
				Rtcp: &RtcpParameters{
					Cname: "video-1",
				},
			},
		})
		assert.Error(t, err)
	})

	t.Run("unsupportd mime type", func(t *testing.T) {
		_, err := transport.Produce(&ProducerOptions{
			Kind: MediaKindAudio,
			RtpParameters: &RtpParameters{
				Codecs: []*RtpCodecParameters{
					{
						MimeType:    "audio/ISAC",
						PayloadType: 108,
						ClockRate:   32000,
					},
				},
				Encodings: []*RtpEncodingParameters{
					{Ssrc: 1111},
				},
				Rtcp: &RtcpParameters{Cname: "audio"},
			},
		})
		assert.Error(t, err)
	})

	t.Run("unsupportd codec parameters", func(t *testing.T) {
		_, err := transport.Produce(&ProducerOptions{
			Kind: MediaKindVideo,
			RtpParameters: &RtpParameters{
				Codecs: []*RtpCodecParameters{
					{
						MimeType:    "video/h264",
						PayloadType: 112,
						ClockRate:   90000,
						Parameters: RtpCodecSpecificParameters{
							PacketizationMode: 1,
							ProfileLevelId:    "CHICKEN",
						},
					},
					{
						MimeType:    "video/rtx",
						PayloadType: 113,
						ClockRate:   90000,
						Parameters:  RtpCodecSpecificParameters{Apt: 112},
					},
				},
				Encodings: []*RtpEncodingParameters{
					{Ssrc: 6666, Rtx: &RtpEncodingRtx{Ssrc: 6667}},
				},
			},
		})
		assert.Error(t, err)
	})

	t.Run("mid already used", func(t *testing.T) {
		// mid is AUDIO
		_ = createAudioProducer(transport)

		_, err := transport.Produce(&ProducerOptions{
			Kind: "audio",
			RtpParameters: &RtpParameters{
				Mid: "AUDIO",
				Codecs: []*RtpCodecParameters{
					{
						MimeType:    "audio/opus",
						PayloadType: 111,
						ClockRate:   48000,
						Channels:    2,
					},
				},
				Encodings: []*RtpEncodingParameters{
					{Ssrc: 33333333},
				},
				Rtcp: &RtcpParameters{Cname: "audio-2"},
			},
		})
		assert.Error(t, err)
	})

	t.Run("ssrc already used", func(t *testing.T) {
		// occupy ssrc: 22222222
		_ = createVideoProducer(transport)
		_, err := transport.Produce(&ProducerOptions{
			Kind: "video",
			RtpParameters: &RtpParameters{
				Mid: "VIDEO2",
				Codecs: []*RtpCodecParameters{
					{
						MimeType:    "video/h264",
						PayloadType: 112,
						ClockRate:   90000,
						Parameters: RtpCodecSpecificParameters{
							PacketizationMode: 1,
							ProfileLevelId:    "4d0032",
						},
					},
				},
				HeaderExtensions: []*RtpHeaderExtensionParameters{
					{
						Uri: "urn:ietf:params:rtp-hdrext:sdes:mid",
						Id:  10,
					},
				},
				Encodings: []*RtpEncodingParameters{
					{Ssrc: 22222222},
				},
			},
		})
		assert.Error(t, err)
	})
}

func TestTransportConsume(t *testing.T) {
	router := createRouter(nil)
	transport := createWebRtcTransport(router)
	aproducer := createAudioProducer(transport)
	aconsumer := createConsumer(transport, aproducer.Id())

	assert.NotEmpty(t, aconsumer.Id())
	assert.Equal(t, aproducer.Id(), aconsumer.ProducerId())
	assert.False(t, aconsumer.Closed())
	assert.EqualValues(t, MediaKindAudio, aconsumer.Kind())
	assert.NotEmpty(t, aconsumer.RtpParameters())
	assert.Equal(t, "0", aconsumer.RtpParameters().Mid)
	assert.Len(t, aconsumer.RtpParameters().Codecs, 1)
	assert.Equal(t, &RtpCodecParameters{
		MimeType:    "audio/opus",
		ClockRate:   48000,
		PayloadType: 100,
		Channels:    2,
		Parameters: RtpCodecSpecificParameters{
			Useinbandfec: 1,
			Usedtx:       1,
		},
		RtcpFeedback: []*RtcpFeedback{{Type: "nack"}},
	}, aconsumer.RtpParameters().Codecs[0])

	assert.Equal(t, ConsumerSimple, aconsumer.Type())
	assert.False(t, aconsumer.Paused())
	assert.False(t, aconsumer.ProducerPaused())
	assert.Equal(t, ConsumerScore{Score: 10, ProducerScore: 0, ProducerScores: []uint8{0}}, aconsumer.Score())
	assert.Nil(t, aconsumer.CurrentLayers())
	assert.Nil(t, aconsumer.PreferredLayers())
	assert.Equal(t, H{"baz": "LOL"}, aconsumer.AppData())

	vproducer := createVideoProducer(transport)
	vproducer.Pause()
	ok, _ := CanConsume(vproducer.ConsumableRtpParameters(), consumerDeviceCapabilities)
	assert.True(t, ok)
	vconsumer := createConsumer(transport, vproducer.Id(), func(co *ConsumerOptions) {
		co.Paused = true
		co.PreferredLayers = &ConsumerLayers{SpatialLayer: 12}
	})
	assert.NotEmpty(t, vconsumer.Id())
	assert.Equal(t, vproducer.Id(), vconsumer.ProducerId())
	assert.False(t, vconsumer.Closed())
	assert.Equal(t, MediaKindVideo, vconsumer.Kind())
	assert.Equal(t, "1", vconsumer.RtpParameters().Mid)
	assert.Len(t, vconsumer.RtpParameters().Codecs, 2)
	assert.Equal(t, &RtpCodecParameters{
		MimeType:    "video/H264",
		ClockRate:   90000,
		PayloadType: 103,
		Parameters: RtpCodecSpecificParameters{
			PacketizationMode: 1,
			ProfileLevelId:    "4d0032",
		},
		RtcpFeedback: []*RtcpFeedback{
			{Type: "nack"},
			{Type: "nack", Parameter: "pli"},
			{Type: "ccm", Parameter: "fir"},
			{Type: "goog-remb"},
		},
	}, vconsumer.RtpParameters().Codecs[0])
	assert.Equal(t, &RtpCodecParameters{
		MimeType:    "video/rtx",
		ClockRate:   90000,
		PayloadType: 104,
		Parameters: RtpCodecSpecificParameters{
			Apt: 103,
		},
	}, vconsumer.RtpParameters().Codecs[1])

	assert.Equal(t, ConsumerSimulcast, vconsumer.Type())
	assert.True(t, vconsumer.Paused())
	assert.True(t, vconsumer.ProducerPaused())
	assert.EqualValues(t, 1, vconsumer.Priority())
	assert.Equal(t, ConsumerScore{Score: 10, ProducerScore: 0, ProducerScores: []uint8{0, 0, 0, 0}}, vconsumer.Score())
	assert.Nil(t, vconsumer.CurrentLayers())
	assert.Equal(t, H{"baz": "LOL"}, vconsumer.AppData())

	pipeConsumer := createConsumer(transport, vproducer.Id(), func(co *ConsumerOptions) {
		co.Pipe = true
	})

	assert.NotEmpty(t, pipeConsumer.Id())
	assert.Equal(t, vproducer.Id(), pipeConsumer.ProducerId())
	assert.False(t, pipeConsumer.Closed())
	assert.Equal(t, MediaKindVideo, pipeConsumer.Kind())
	assert.Empty(t, pipeConsumer.RtpParameters().Mid)
	assert.Len(t, pipeConsumer.RtpParameters().Codecs, 2)
	assert.Equal(t, &RtpCodecParameters{
		MimeType:    "video/H264",
		ClockRate:   90000,
		PayloadType: 103,
		Parameters: RtpCodecSpecificParameters{
			PacketizationMode: 1,
			ProfileLevelId:    "4d0032",
		},
		RtcpFeedback: []*RtcpFeedback{
			{Type: "nack"},
			{Type: "nack", Parameter: "pli"},
			{Type: "ccm", Parameter: "fir"},
			{Type: "goog-remb"},
		},
	}, pipeConsumer.RtpParameters().Codecs[0])
	assert.Equal(t, &RtpCodecParameters{
		MimeType:    "video/rtx",
		ClockRate:   90000,
		PayloadType: 104,
		Parameters: RtpCodecSpecificParameters{
			Apt: 103,
		},
	}, pipeConsumer.RtpParameters().Codecs[1])
	assert.Equal(t, ConsumerPipe, pipeConsumer.Type())
	assert.False(t, pipeConsumer.Paused())
	assert.True(t, pipeConsumer.ProducerPaused())
	assert.EqualValues(t, 1, pipeConsumer.Priority())
	assert.Equal(t, ConsumerScore{Score: 10, ProducerScore: 10, ProducerScores: []uint8{0, 0, 0, 0}}, pipeConsumer.Score())
	assert.Nil(t, pipeConsumer.PreferredLayers())
	assert.Nil(t, pipeConsumer.CurrentLayers())

	routerDump, _ := router.Dump()

	assert.True(t, slices.ContainsFunc(
		routerDump.MapProducerIdConsumerIds,
		func(kvs KeyValues[string, string]) bool {
			return kvs.Key == aproducer.Id() && slices.Contains(kvs.Values, aconsumer.Id())
		}),
	)
	index := slices.IndexFunc(routerDump.MapProducerIdConsumerIds, func(kvs KeyValues[string, string]) bool {
		return kvs.Key == vproducer.Id()
	})
	assert.ElementsMatch(t, []string{vconsumer.Id(), pipeConsumer.Id()}, routerDump.MapProducerIdConsumerIds[index].Values)
	assert.True(t, slices.ContainsFunc(
		routerDump.MapConsumerIdProducerId,
		func(kv KeyValue[string, string]) bool {
			return kv.Key == aconsumer.Id() && kv.Value == aproducer.Id()
		}),
	)
	assert.True(t, slices.ContainsFunc(
		routerDump.MapConsumerIdProducerId,
		func(kv KeyValue[string, string]) bool {
			return kv.Key == vconsumer.Id() && kv.Value == vproducer.Id()
		}),
	)
	assert.True(t, slices.ContainsFunc(
		routerDump.MapConsumerIdProducerId,
		func(kv KeyValue[string, string]) bool {
			return kv.Key == pipeConsumer.Id() && kv.Value == vproducer.Id()
		}),
	)

	transportDump, _ := transport.Dump()

	assert.Equal(t, transport.Id(), transportDump.Id)
	assert.ElementsMatch(t, []string{aconsumer.Id(), vconsumer.Id(), pipeConsumer.Id()}, transportDump.ConsumerIds)

	for i, c := range []*Consumer{aconsumer, vconsumer, pipeConsumer} {
		consumer := router.GetConsumerById(c.Id())
		assert.Equal(t, consumer, c, i)
		assert.True(t, ok, i)
	}
}

func TestTransportConsumeCreatedWithEnableRtx(t *testing.T) {
	transport := createWebRtcTransport(nil)
	audioProducer := createAudioProducer(transport)
	audioConsumer, _ := transport.Consume(&ConsumerOptions{
		ProducerId:      audioProducer.Id(),
		RtpCapabilities: consumerDeviceCapabilities,
		EnableRtx:       ref(true),
	})
	assert.Equal(t, MediaKindAudio, audioConsumer.Kind())
	assert.Len(t, audioConsumer.RtpParameters().Codecs, 1)
	assert.Equal(t, []*RtpCodecParameters{
		{
			MimeType:    "audio/opus",
			PayloadType: 100,
			ClockRate:   48000,
			Channels:    2,
			Parameters: RtpCodecSpecificParameters{
				Useinbandfec: 1,
				Usedtx:       1,
			},
			RtcpFeedback: []*RtcpFeedback{{Type: "nack"}},
		},
	}, audioConsumer.RtpParameters().Codecs)
}

func TestTransportConsumeCreatedWithUserProvidedMid(t *testing.T) {
	transport := createWebRtcTransport(nil)
	audioProducer := createAudioProducer(transport)
	audioConsumer1, _ := transport.Consume(&ConsumerOptions{
		ProducerId:      audioProducer.Id(),
		RtpCapabilities: consumerDeviceCapabilities,
	})
	re := regexp.MustCompile("^[0-9]+")
	assert.True(t, re.MatchString(audioConsumer1.RtpParameters().Mid))

	audioConsumer2, _ := transport.Consume(&ConsumerOptions{
		ProducerId:      audioProducer.Id(),
		Mid:             "custom-mid",
		RtpCapabilities: consumerDeviceCapabilities,
	})
	assert.Equal(t, "custom-mid", audioConsumer2.RtpParameters().Mid)

	audioConsumer3, _ := transport.Consume(&ConsumerOptions{
		ProducerId:      audioProducer.Id(),
		RtpCapabilities: consumerDeviceCapabilities,
	})
	assert.True(t, re.MatchString(audioConsumer3.RtpParameters().Mid))
	mid1, _ := strconv.Atoi(audioConsumer1.RtpParameters().Mid)
	assert.Equal(t, strconv.Itoa(mid1+1), audioConsumer3.RtpParameters().Mid)
}

func TestTransportConsumeUnsupportedError(t *testing.T) {
	router := createRouter(nil)
	transport := createWebRtcTransport(router)
	audioProducer := createAudioProducer(transport)

	invalidDeviceCapabilities := &RtpCapabilities{
		Codecs: []*RtpCodecCapability{
			{
				Kind:                 MediaKindAudio,
				MimeType:             "audio/ISAC",
				ClockRate:            32000,
				PreferredPayloadType: 100,
				Channels:             1,
			},
		},
	}

	assert.False(t, router.CanConsume(audioProducer.Id(), invalidDeviceCapabilities))

	_, err := transport.Consume(&ConsumerOptions{
		ProducerId:      audioProducer.Id(),
		RtpCapabilities: invalidDeviceCapabilities,
	})
	assert.Error(t, err)

	invalidDeviceCapabilities = &RtpCapabilities{}

	assert.False(t, router.CanConsume(audioProducer.Id(), invalidDeviceCapabilities))

	_, err = transport.Consume(&ConsumerOptions{
		ProducerId:      audioProducer.Id(),
		RtpCapabilities: invalidDeviceCapabilities,
	})
	assert.Error(t, err)
}

func TestTransportProduceData(t *testing.T) {
	router := createRouter(newTestWorker())
	transport1 := createWebRtcTransport(router, func(o *WebRtcTransportOptions) { o.EnableSctp = true })
	transport2 := createPlainTransport(router, func(o *PlainTransportOptions) { o.EnableSctp = true })

	for i, transport := range []*Transport{transport1, transport2} {
		dataProducer, err := transport.ProduceData(&DataProducerOptions{
			SctpStreamParameters: &SctpStreamParameters{StreamId: uint16(i + 1), MaxRetransmits: ref[uint16](3)},
			Label:                "foo",
			Protocol:             "bar",
			AppData:              H{"foo": 1, "bar": "2"},
		})
		assert.NoError(t, err)
		assert.False(t, dataProducer.Closed())
		assert.Equal(t, DataProducerSctp, dataProducer.Type())
		assert.NotNil(t, dataProducer.SctpStreamParameters())
		assert.EqualValues(t, i+1, dataProducer.SctpStreamParameters().StreamId)
		assert.False(t, *dataProducer.SctpStreamParameters().Ordered)
		assert.Zero(t, dataProducer.SctpStreamParameters().MaxPacketLifeTime)
		assert.EqualValues(t, 3, *dataProducer.SctpStreamParameters().MaxRetransmits)
		assert.Equal(t, "foo", dataProducer.Label())
		assert.Equal(t, "bar", dataProducer.Protocol())
		assert.Equal(t, H{"foo": 1, "bar": "2"}, dataProducer.AppData())

		routerDump, _ := router.Dump()
		assert.True(t, slices.ContainsFunc(
			routerDump.MapDataProducerIdDataConsumerIds,
			func(kvs KeyValues[string, string]) bool {
				return kvs.Key == dataProducer.Id()
			}),
		)
		assert.Empty(t, routerDump.MapDataConsumerIdDataProducerId)

		transportDump, _ := transport.Dump()
		assert.Equal(t, transport.Id(), transportDump.Id)
		assert.ElementsMatch(t, []string{dataProducer.Id()}, transportDump.DataProducerIds)
		assert.Empty(t, transportDump.DataConsumerIds)

		p := router.GetDataProducerById(dataProducer.Id())
		assert.Equal(t, dataProducer, p)
	}

	_, err := transport1.ProduceData(&DataProducerOptions{})
	assert.Equal(t, ErrMissSctpStreamParameters, err)

	// used streamId
	_, err = transport1.ProduceData(&DataProducerOptions{
		SctpStreamParameters: &SctpStreamParameters{StreamId: 1},
	})
	assert.Error(t, err)

	// with ordered and maxPacketLifeTime rejects
	_, err = transport1.ProduceData(&DataProducerOptions{
		SctpStreamParameters: &SctpStreamParameters{
			StreamId:          999,
			Ordered:           ref(true),
			MaxPacketLifeTime: ref[uint16](4000),
		},
	})
	assert.Error(t, err)
}

func TestTransportConsumeData(t *testing.T) {
	worker := newTestWorker()
	router1 := createRouter(worker)
	router2 := createRouter(worker)
	transport1 := createPlainTransport(router1, func(o *PlainTransportOptions) { o.EnableSctp = true })
	transport2 := createDirectTransport(router2)

	for transport, router := range map[*Transport]*Router{
		transport1: router1,
		transport2: router2,
	} {
		dataProducer, _ := transport.ProduceData(&DataProducerOptions{
			SctpStreamParameters: &SctpStreamParameters{
				StreamId:          12345,
				Ordered:           ref(false),
				MaxPacketLifeTime: ref[uint16](5000),
			},
			Label:    "foo",
			Protocol: "bar",
			AppData:  H{"foo": 1, "bar": "2"},
		})
		dataConsumer, err := transport.ConsumeData(&DataConsumerOptions{
			DataProducerId:    dataProducer.Id(),
			MaxPacketLifeTime: 4000,
			AppData:           H{"baz": "LOL"},
		})
		assert.NoError(t, err, transport.Type())
		assert.Equal(t, dataProducer.Id(), dataConsumer.DataProducerId())
		assert.False(t, dataConsumer.Closed())
		if transport.Type() == TransportDirect {
			assert.Equal(t, DataConsumerDirect, dataConsumer.Type())
		} else {
			assert.Equal(t, DataConsumerSctp, dataConsumer.Type())
			assert.False(t, *dataConsumer.SctpStreamParameters().Ordered)
			assert.EqualValues(t, 4000, *dataConsumer.SctpStreamParameters().MaxPacketLifeTime)
			assert.Zero(t, dataConsumer.SctpStreamParameters().MaxRetransmits)
		}
		assert.Equal(t, "foo", dataConsumer.Label())
		assert.Equal(t, "bar", dataConsumer.Protocol())
		assert.Equal(t, H{"baz": "LOL"}, dataConsumer.AppData())

		routerDump, _ := router.Dump()
		assert.True(t, slices.ContainsFunc(
			routerDump.MapDataProducerIdDataConsumerIds,
			func(kvs KeyValues[string, string]) bool {
				return kvs.Key == dataProducer.Id() && slices.Contains(kvs.Values, dataConsumer.Id())
			}),
		)
		assert.True(t, slices.ContainsFunc(
			routerDump.MapDataConsumerIdDataProducerId,
			func(kv KeyValue[string, string]) bool {
				return kv.Key == dataConsumer.Id() && kv.Value == dataProducer.Id()
			}),
		)

		transportDump, _ := transport.Dump()
		assert.Equal(t, transport.Id(), transportDump.Id)
		assert.Empty(t, transportDump.ProducerIds)
		assert.Equal(t, []string{dataConsumer.Id()}, transportDump.DataConsumerIds)

		c := router.GetDataConsumerById(dataConsumer.Id())
		assert.Equal(t, dataConsumer, c)
	}
}
