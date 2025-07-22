package mediasoup

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func createRouter(worker *Worker) *Router {
	if worker == nil {
		worker = newTestWorker()
	}
	router, err := worker.CreateRouter(&RouterOptions{
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
					LevelAsymmetryAllowed: 1,
					PacketizationMode:     1,
					ProfileLevelId:        "4d0032",
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}
	return router
}

func TestRouter_UpdateMediaCodecs(t *testing.T) {
	router := createRouter(nil)

	assert.NotNil(t, router.RtpCapabilities())
	// 3 codecs + 2 RTX codecs.
	assert.Len(t, router.RtpCapabilities().Codecs, 5)
	assert.NotNil(t, router.RtpCapabilities().HeaderExtensions)

	err := router.UpdateMediaCodecs([]*RtpCodecCapability{})
	require.NoError(t, err)

	assert.NotNil(t, router.RtpCapabilities())
	assert.Len(t, router.RtpCapabilities().Codecs, 0)
	assert.NotNil(t, router.RtpCapabilities().HeaderExtensions)
}

func TestRouterClose(t *testing.T) {
	t.Run("close normally", func(t *testing.T) {
		mymock := new(MockedHandler)
		defer mymock.AssertExpectations(t)

		mymock.On("OnClose", mock.IsType(context.Background())).Once()

		worker := newTestWorker()
		router, _ := worker.CreateRouter(&RouterOptions{})
		router.OnClose(mymock.OnClose)
		assert.NoError(t, router.Close())
		assert.True(t, router.Closed())
	})

	t.Run("worker closed", func(t *testing.T) {
		mymock := new(MockedHandler)
		defer mymock.AssertExpectations(t)

		mymock.On("OnClose", mock.IsType(context.Background())).Once()

		worker := newTestWorker()
		router, _ := worker.CreateRouter(&RouterOptions{})
		router.OnClose(mymock.OnClose)
		worker.Close()
		assert.True(t, router.Closed())
	})
}

func TestRouterDump(t *testing.T) {
	worker := newTestWorker()
	router, _ := worker.CreateRouter(&RouterOptions{})
	dump, err := router.Dump()
	require.NoError(t, err)
	assert.Equal(t, router.Id(), dump.Id)
}

func TestCreateWebRtcTransport(t *testing.T) {
	mymock := new(MockedHandler)
	defer mymock.AssertExpectations(t)

	mymock.On("OnNewTransport", mock.IsType(context.Background()), mock.IsType(&Transport{})).Times(2)

	worker := newTestWorker()
	router, _ := worker.CreateRouter(&RouterOptions{})
	router.OnNewTransport(mymock.OnNewTransport)
	transport1, err := router.CreateWebRtcTransport(&WebRtcTransportOptions{
		ListenInfos: []TransportListenInfo{
			{Ip: "127.0.0.1"},
		},
	})
	require.NoError(t, err)
	webRtcServer, _ := worker.CreateWebRtcServer(&WebRtcServerOptions{
		ListenInfos: []*TransportListenInfo{
			{Protocol: TransportProtocolUDP, Ip: "127.0.0.1", Port: 0},
		},
	})
	transport2, err := router.CreateWebRtcTransport(&WebRtcTransportOptions{
		WebRtcServer: webRtcServer,
	})
	require.NoError(t, err)
	dump, _ := router.Dump()
	assert.ElementsMatch(t, []string{transport1.Id(), transport2.Id()}, dump.TransportIds)

	transport := router.GetTransportById(transport1.Id())
	require.Equal(t, transport1, transport)

	transport = router.GetTransportById(transport2.Id())
	require.Equal(t, transport2, transport)

	router.Close()
	transport = router.GetTransportById(transport1.Id())
	require.Nil(t, transport)
}

func TestCreateWebRtcTransportWithPortRange(t *testing.T) {
	worker := newTestWorker()
	router, _ := worker.CreateRouter(&RouterOptions{})

	portRange := TransportPortRange{Min: 11111, Max: 11112}

	transport1, err := router.CreateWebRtcTransport(&WebRtcTransportOptions{
		ListenInfos: []TransportListenInfo{
			{Protocol: TransportProtocolUDP, Ip: "127.0.0.1", PortRange: portRange},
		},
	})
	require.NoError(t, err)

	iceCandidate1 := transport1.Data().IceCandidates[0]
	assert.Equal(t, "127.0.0.1", iceCandidate1.Address)
	assert.True(t, iceCandidate1.Port >= portRange.Min && iceCandidate1.Port <= portRange.Max)
	assert.Equal(t, TransportProtocolUDP, iceCandidate1.Protocol)

	transport2, err := router.CreateWebRtcTransport(&WebRtcTransportOptions{
		ListenInfos: []TransportListenInfo{
			{Protocol: TransportProtocolUDP, Ip: "127.0.0.1", PortRange: portRange},
		},
	})
	require.NoError(t, err)

	iceCandidate2 := transport2.Data().IceCandidates[0]
	assert.Equal(t, "127.0.0.1", iceCandidate2.Address)
	assert.True(t, iceCandidate2.Port >= portRange.Min && iceCandidate2.Port <= portRange.Max)
	assert.EqualValues(t, TransportProtocolUDP, iceCandidate2.Protocol)

	// No more available ports so it must fail
	_, err = router.CreateWebRtcTransport(&WebRtcTransportOptions{
		ListenInfos: []TransportListenInfo{
			{Protocol: TransportProtocolUDP, Ip: "127.0.0.1", PortRange: portRange},
		},
	})
	assert.Error(t, err)
}

func TestCreatePlainTransport(t *testing.T) {
	mymock := new(MockedHandler)
	defer mymock.AssertExpectations(t)

	mymock.On("OnNewTransport", mock.IsType(context.Background()), mock.IsType(&Transport{})).Once()

	worker := newTestWorker()
	router, _ := worker.CreateRouter(&RouterOptions{})
	router.OnNewTransport(mymock.OnNewTransport)
	transport, err := router.CreatePlainTransport(&PlainTransportOptions{
		ListenInfo: TransportListenInfo{
			Ip: "127.0.0.1",
		},
	})
	require.NoError(t, err)
	dump, _ := router.Dump()
	assert.Contains(t, dump.TransportIds, transport.Id())
}

func TestCreatePipeTransport(t *testing.T) {
	router1 := createRouter(nil)
	transport1, _ := router1.CreateWebRtcTransport(&WebRtcTransportOptions{
		ListenInfos: []TransportListenInfo{
			{Ip: "127.0.0.1"},
		},
		EnableSctp: true,
	})
	videoProducer, _ := transport1.Produce(&ProducerOptions{
		Kind: MediaKindVideo,
		RtpParameters: &RtpParameters{
			Mid: "VIDEO",
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "video/VP8",
					PayloadType: 112,
					ClockRate:   90000,
					RtcpFeedback: []*RtcpFeedback{
						{Type: "nack"},
						{Type: "nack", Parameter: "pli"},
						{Type: "goog-remb"},
						{Type: "lalala"},
					},
				},
			},
			HeaderExtensions: []*RtpHeaderExtensionParameters{
				{
					Uri: "urn:ietf:params:rtp-hdrext:sdes:mid",
					Id:  10,
				},
				{
					Uri: "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time",
					Id:  11,
				},
				{
					Uri: "urn:3gpp:video-orientation",
					Id:  13,
				},
			},
			Encodings: []*RtpEncodingParameters{
				{Ssrc: 22222222},
				{Ssrc: 22222223},
				{Ssrc: 22222224},
			},
			Rtcp: &RtcpParameters{
				Cname: "FOOBAR",
			},
		},
		AppData: H{"foo": "bar2"},
	})

	videoProducer.Pause()

	t.Run("enable rtx", func(t *testing.T) {
		mymock := new(MockedHandler)
		defer mymock.AssertExpectations(t)

		mymock.On("OnNewTransport", mock.IsType(context.Background()), mock.IsType(&Transport{}))
		router1.OnNewTransport(mymock.OnNewTransport)

		pipeTransport, err := router1.CreatePipeTransport(&PipeTransportOptions{
			ListenInfo: TransportListenInfo{Ip: "127.0.0.1"},
			EnableRtx:  true,
		})
		assert.NoError(t, err)
		assert.Nil(t, pipeTransport.Data().PipeTransportData.SrtpParameters)

		// No SRTP enabled so passing srtpParameters must fail.
		err = pipeTransport.Connect(&TransportConnectOptions{
			Ip:   "127.0.0.2",
			Port: ref[uint16](9999),
			SrtpParameters: &SrtpParameters{
				CryptoSuite: "AEAD_AES_256_GCM",
				KeyBase64:   "YTdjcDBvY2JoMGY5YXNlNDc0eDJsdGgwaWRvNnJsamRrdG16aWVpZHphdHo=",
			},
		})
		assert.Error(t, err)

		pipeConsumer, err := pipeTransport.Consume(&ConsumerOptions{
			ProducerId: videoProducer.Id(),
		})
		assert.NoError(t, err)
		assert.False(t, pipeConsumer.Closed())
		assert.EqualValues(t, "video", pipeConsumer.Kind())
		assert.NotNil(t, pipeConsumer.RtpParameters())
		assert.Zero(t, pipeConsumer.RtpParameters().Mid)
		assert.Equal(t, []*RtpCodecParameters{
			{
				MimeType:    "video/VP8",
				ClockRate:   90000,
				PayloadType: 101,
				RtcpFeedback: []*RtcpFeedback{
					{Type: "nack", Parameter: ""},
					{Type: "nack", Parameter: "pli"},
					{Type: "ccm", Parameter: "fir"},
				},
			},
			{
				MimeType:    "video/rtx",
				ClockRate:   90000,
				PayloadType: 102,
				Parameters: RtpCodecSpecificParameters{
					Apt: 101,
				},
				RtcpFeedback: []*RtcpFeedback{},
			},
		}, pipeConsumer.RtpParameters().Codecs)
		assert.Equal(t, []*RtpHeaderExtensionParameters{
			// TODO: Enable when DD is sendrecv.
			// {
			// 	Uri:     "https://aomediacodec.github.io/av1-rtp-spec/#dependency-descriptor-rtp-header-extension",
			// 	Id:      8,
			// 	Encrypt: false,
			// },
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
			{
				Uri:     "http://www.webrtc.org/experiments/rtp-hdrext/abs-capture-time",
				Id:      13,
				Encrypt: false,
			},
			{
				Uri:     "http://www.webrtc.org/experiments/rtp-hdrext/playout-delay",
				Id:      14,
				Encrypt: false,
			},
		}, pipeConsumer.RtpParameters().HeaderExtensions)
		assert.EqualValues(t, "pipe", pipeConsumer.Type())
		assert.False(t, pipeConsumer.Paused())
		assert.True(t, pipeConsumer.ProducerPaused())
		assert.Equal(t, ConsumerScore{
			Score:          10,
			ProducerScore:  10,
			ProducerScores: []int{},
		}, pipeConsumer.Score())

		pipeTransport.Close()
	})

	t.Run("invalid srtpParameters must fail", func(t *testing.T) {
		pipeTransport, _ := router1.CreatePipeTransport(&PipeTransportOptions{
			ListenInfo: TransportListenInfo{Ip: "127.0.0.1"},
			EnableRtx:  true,
		})
		err := pipeTransport.Connect(&TransportConnectOptions{
			Ip:   "127.0.0.2",
			Port: ref[uint16](9999),
			SrtpParameters: &SrtpParameters{
				CryptoSuite: AEAD_AES_256_GCM,
				KeyBase64:   "YTdjcDBvY2JoMGY5YXNlNDc0eDJsdGgwaWRvNnJsamRrdG16aWVpZHphdHo=",
			},
		})
		assert.Error(t, err)
	})

	t.Run("enable srtp", func(t *testing.T) {
		pipeTransport, err := router1.CreatePipeTransport(&PipeTransportOptions{
			ListenInfo: TransportListenInfo{Ip: "127.0.0.1"},
			EnableSrtp: true,
		})
		assert.NoError(t, err)
		// The master length of AEAD_AES_256_GCM.
		assert.Len(t, pipeTransport.Data().PipeTransportData.SrtpParameters.KeyBase64, 60)

		// Missing srtpParameters.
		err = pipeTransport.Connect(&TransportConnectOptions{
			Ip:   "127.0.0.2",
			Port: ref[uint16](9999),
		})
		assert.Error(t, err)

		// Missing srtpParameters.crypto
		err = pipeTransport.Connect(&TransportConnectOptions{
			Ip:   "127.0.0.2",
			Port: ref[uint16](9999),
			SrtpParameters: &SrtpParameters{
				KeyBase64: "YTdjcDBvY2JoMGY5YXNlNDc0eDJsdGgwaWRvNnJsamRrdG16aWVpZHphdHo=",
			},
		})
		assert.Error(t, err)

		// Invalid srtpParameters.keyBase64.
		err = pipeTransport.Connect(&TransportConnectOptions{
			Ip:   "127.0.0.2",
			Port: ref[uint16](9999),
			SrtpParameters: &SrtpParameters{
				CryptoSuite: "AEAD_AES_256_GCM",
			},
		})
		assert.Error(t, err)

		// Invalid srtpParameters.crypto
		err = pipeTransport.Connect(&TransportConnectOptions{
			Ip:   "127.0.0.2",
			Port: ref[uint16](9999),
			SrtpParameters: &SrtpParameters{
				CryptoSuite: "FOO",
				KeyBase64:   "YTdjcDBvY2JoMGY5YXNlNDc0eDJsdGgwaWRvNnJsamRrdG16aWVpZHphdHo=",
			},
		})
		assert.Error(t, err)

		// Valid srtpParameters.
		err = pipeTransport.Connect(&TransportConnectOptions{
			Ip:   "127.0.0.2",
			Port: ref[uint16](9999),
			SrtpParameters: &SrtpParameters{
				CryptoSuite: "AEAD_AES_256_GCM",
				KeyBase64:   "YTdjcDBvY2JoMGY5YXNlNDc0eDJsdGgwaWRvNnJsamRrdG16aWVpZHphdHo=",
			},
		})
		assert.NoError(t, err)

		pipeTransport.Close()
	})

	t.Run("fixed port", func(t *testing.T) {
		port := pickUdpPort()
		pipeTransport, _ := router1.CreatePipeTransport(&PipeTransportOptions{
			ListenInfo: TransportListenInfo{Ip: "127.0.0.1", Port: port},
		})

		assert.Equal(t, port, pipeTransport.Data().PipeTransportData.Tuple.LocalPort)
		pipeTransport.Close()
	})
}

func TestCreateDirectTransport(t *testing.T) {
	mymock := new(MockedHandler)
	defer mymock.AssertExpectations(t)

	mymock.On("OnNewTransport", mock.IsType(context.Background()), mock.IsType(&Transport{})).Once()

	worker := newTestWorker()
	router, _ := worker.CreateRouter(&RouterOptions{})
	router.OnNewTransport(mymock.OnNewTransport)
	transport, err := router.CreateDirectTransport(&DirectTransportOptions{})
	require.NoError(t, err)
	dump, _ := router.Dump()
	assert.Contains(t, dump.TransportIds, transport.Id())
}

func TestCreateActiveSpeakerObserver(t *testing.T) {
	mymock := new(MockedHandler)
	defer mymock.AssertExpectations(t)

	mymock.On("OnNewRtpObserver", mock.IsType(context.Background()), mock.IsType(&RtpObserver{})).Once()

	worker := newTestWorker()
	router, _ := worker.CreateRouter(&RouterOptions{})
	router.OnNewRtpObserver(mymock.OnNewRtpObserver)
	rtpObserver, err := router.CreateActiveSpeakerObserver(nil)
	require.NoError(t, err)
	assert.NotEmpty(t, rtpObserver.Id())

	o := router.GetRtpObserverById(rtpObserver.Id())
	require.Equal(t, rtpObserver, o)

	router.Close()
	o = router.GetRtpObserverById(rtpObserver.Id())
	require.Nil(t, o)
}

func TestCreateAudioLevelObserver(t *testing.T) {
	mymock := new(MockedHandler)
	defer mymock.AssertExpectations(t)

	mymock.On("OnNewRtpObserver", mock.IsType(context.Background()), mock.IsType(&RtpObserver{})).Once()

	worker := newTestWorker()
	router, _ := worker.CreateRouter(&RouterOptions{})
	router.OnNewRtpObserver(mymock.OnNewRtpObserver)
	rtpObserver, err := router.CreateAudioLevelObserver(nil)
	require.NoError(t, err)
	assert.NotEmpty(t, rtpObserver.Id())
}

func TestPipeToRouter(t *testing.T) {
	worker1 := newTestWorker()
	worker2 := newTestWorker()
	router1 := createRouter(worker1)
	router2 := createRouter(worker2)

	transport1, _ := router1.CreateWebRtcTransport(&WebRtcTransportOptions{
		ListenInfos: []TransportListenInfo{
			{Ip: "127.0.0.1"},
		},
		EnableSctp: true,
	})
	transport2, _ := router2.CreateWebRtcTransport(&WebRtcTransportOptions{
		ListenInfos: []TransportListenInfo{
			{Ip: "127.0.0.1"},
		},
		EnableSctp: true,
	})
	audioProducer := createAudioProducer(transport1)
	videoProducer, _ := transport1.Produce(&ProducerOptions{
		Kind: MediaKindVideo,
		RtpParameters: &RtpParameters{
			Mid: "VIDEO",
			Codecs: []*RtpCodecParameters{
				{
					MimeType:    "video/VP8",
					PayloadType: 112,
					ClockRate:   90000,
					RtcpFeedback: []*RtcpFeedback{
						{Type: "nack"},
						{Type: "nack", Parameter: "pli"},
						{Type: "goog-remb"},
						{Type: "lalala"},
					},
				},
			},
			HeaderExtensions: []*RtpHeaderExtensionParameters{
				{
					Uri: "urn:ietf:params:rtp-hdrext:sdes:mid",
					Id:  10,
				},
				{
					Uri: "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time",
					Id:  11,
				},
				{
					Uri: "urn:3gpp:video-orientation",
					Id:  13,
				},
			},
			Encodings: []*RtpEncodingParameters{
				{Ssrc: 22222222},
				{Ssrc: 22222223},
				{Ssrc: 22222224},
			},
			Rtcp: &RtcpParameters{
				Cname: "FOOBAR",
			},
		},
		AppData: H{"foo": "bar2"},
	})

	var myDeviceCapabilities = &RtpCapabilities{
		Codecs: []*RtpCodecCapability{
			{
				Kind:                 "audio",
				MimeType:             "audio/opus",
				PreferredPayloadType: 100,
				ClockRate:            48000,
				Channels:             2,
			},
			{
				Kind:                 "video",
				MimeType:             "video/VP8",
				PreferredPayloadType: 101,
				ClockRate:            90000,
				RtcpFeedback: []*RtcpFeedback{
					{Type: "nack"},
					{Type: "ccm", Parameter: "fir"},
					{Type: "transport-cc"},
				},
			},
			{
				Kind:                 "video",
				MimeType:             "video/rtx",
				PreferredPayloadType: 102,
				ClockRate:            90000,
				Parameters: RtpCodecSpecificParameters{
					Apt: 101,
				},
				RtcpFeedback: []*RtcpFeedback{},
			},
		},
		HeaderExtensions: []*RtpHeaderExtension{
			{
				Kind:             "video",
				Uri:              "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time",
				PreferredId:      4,
				PreferredEncrypt: false,
				Direction:        "sendrecv",
			},
			{
				Kind:             "video",
				Uri:              "http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01",
				PreferredId:      5,
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

	dataProducer, _ := transport1.ProduceData(&DataProducerOptions{
		SctpStreamParameters: &SctpStreamParameters{
			StreamId:          666,
			Ordered:           ref(false),
			MaxPacketLifeTime: ref[uint16](5000),
		},
		Label:    "foo",
		Protocol: "bar",
	})

	// Pause the videoProducer.
	videoProducer.Pause()

	t.Run("audio", func(t *testing.T) {
		result, err := router1.PipeToRouter(&PipeToRouterOptions{
			ProducerId: audioProducer.Id(),
			Router:     router2,
		})
		assert.NoError(t, err)

		time.Sleep(time.Millisecond)

		pipeConsumer, pipeProducer := result.PipeConsumer, result.PipeProducer

		dump, _ := router1.Dump()

		// There shoud should be two Transports in router1:
		// - WebRtcTransport for audioProducer and videoProducer.
		// - PipeTransport between router1 and router2.
		assert.Len(t, dump.TransportIds, 2)

		dump, _ = router1.Dump()

		// There shoud should be two Transports in router2:
		// - WebRtcTransport for audioConsumer and videoConsumer.
		// - pipeTransport between router2 and router1.
		assert.Len(t, dump.TransportIds, 2)

		assert.False(t, pipeConsumer.Closed())
		assert.Equal(t, MediaKindAudio, pipeConsumer.Kind())
		assert.NotNil(t, pipeConsumer.RtpParameters())
		assert.Empty(t, pipeConsumer.RtpParameters().Mid)
		assert.Equal(t, []*RtpCodecParameters{
			{
				MimeType:     "audio/opus",
				ClockRate:    48000,
				PayloadType:  100,
				Channels:     2,
				Parameters:   RtpCodecSpecificParameters{Useinbandfec: 1, Usedtx: 1},
				RtcpFeedback: []*RtcpFeedback{},
			},
		}, pipeConsumer.RtpParameters().Codecs)
		assert.Equal(t, []*RtpHeaderExtensionParameters{
			{
				Uri:     "urn:ietf:params:rtp-hdrext:ssrc-audio-level",
				Id:      10,
				Encrypt: false,
			},
			{
				Uri:     "http://www.webrtc.org/experiments/rtp-hdrext/abs-capture-time",
				Id:      13,
				Encrypt: false,
			},
			{
				Uri:     "http://www.webrtc.org/experiments/rtp-hdrext/playout-delay",
				Id:      14,
				Encrypt: false,
			},
		}, pipeConsumer.RtpParameters().HeaderExtensions)
		assert.EqualValues(t, ConsumerPipe, pipeConsumer.Type())
		assert.False(t, pipeConsumer.Paused())
		assert.False(t, pipeConsumer.ProducerPaused())
		assert.Equal(t, ConsumerScore{
			Score:          10,
			ProducerScore:  10,
			ProducerScores: []int{},
		}, pipeConsumer.Score())

		assert.Equal(t, audioProducer.Id(), pipeProducer.Id())
		assert.False(t, pipeProducer.Closed())
		assert.Equal(t, MediaKindAudio, pipeProducer.Kind())
		assert.Empty(t, pipeProducer.RtpParameters().Mid)
		assert.Equal(t, []*RtpCodecParameters{
			{
				MimeType:     "audio/opus",
				ClockRate:    48000,
				PayloadType:  100,
				Channels:     2,
				Parameters:   RtpCodecSpecificParameters{Useinbandfec: 1, Usedtx: 1},
				RtcpFeedback: []*RtcpFeedback{},
			},
		}, pipeProducer.RtpParameters().Codecs)
		assert.Equal(t, []*RtpHeaderExtensionParameters{
			{
				Uri:     "urn:ietf:params:rtp-hdrext:ssrc-audio-level",
				Id:      10,
				Encrypt: false,
			},
			{
				Uri:     "http://www.webrtc.org/experiments/rtp-hdrext/abs-capture-time",
				Id:      13,
				Encrypt: false,
			},
			{
				Uri:     "http://www.webrtc.org/experiments/rtp-hdrext/playout-delay",
				Id:      14,
				Encrypt: false,
			},
		}, pipeProducer.RtpParameters().HeaderExtensions)
		assert.False(t, pipeProducer.Paused())
	})

	t.Run("video", func(t *testing.T) {
		result, err := router1.PipeToRouter(&PipeToRouterOptions{
			ProducerId: videoProducer.Id(),
			Router:     router2,
		})
		assert.NoError(t, err)

		pipeConsumer, pipeProducer := result.PipeConsumer, result.PipeProducer

		dump, _ := router1.Dump()

		// There shoud should be two Transports in router1:
		// - WebRtcTransport for audioProducer and videoProducer.
		// - PipeTransport between router1 and router2.
		assert.Len(t, dump.TransportIds, 2)

		dump, _ = router1.Dump()

		// There shoud should be two Transports in router2:
		// - WebRtcTransport for audioConsumer and videoConsumer.
		// - pipeTransport between router2 and router1.
		assert.Len(t, dump.TransportIds, 2)

		assert.False(t, pipeConsumer.Closed())
		assert.EqualValues(t, "video", pipeConsumer.Kind())
		assert.NotNil(t, pipeConsumer.RtpParameters())
		assert.Zero(t, pipeConsumer.RtpParameters().Mid)
		assert.EqualValues(t, []*RtpCodecParameters{
			{
				MimeType:    "video/VP8",
				ClockRate:   90000,
				PayloadType: 101,
				RtcpFeedback: []*RtcpFeedback{
					{Type: "nack", Parameter: "pli"},
					{Type: "ccm", Parameter: "fir"},
				},
			},
		}, pipeConsumer.RtpParameters().Codecs)
		assert.EqualValues(t, []*RtpHeaderExtensionParameters{
			// TODO: Enable when DD is sendrecv.
			// {
			// 	Uri:     "https://aomediacodec.github.io/av1-rtp-spec/#dependency-descriptor-rtp-header-extension",
			// 	Id:      8,
			// 	Encrypt: false,
			// },
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
			{
				Uri:     "http://www.webrtc.org/experiments/rtp-hdrext/abs-capture-time",
				Id:      13,
				Encrypt: false,
			},
			{
				Uri:     "http://www.webrtc.org/experiments/rtp-hdrext/playout-delay",
				Id:      14,
				Encrypt: false,
			},
		}, pipeConsumer.RtpParameters().HeaderExtensions)
		assert.EqualValues(t, "pipe", pipeConsumer.Type())
		assert.False(t, pipeConsumer.Paused())
		assert.True(t, pipeConsumer.ProducerPaused())
		assert.Equal(t, ConsumerScore{
			Score:          10,
			ProducerScore:  10,
			ProducerScores: []int{},
		}, pipeConsumer.Score())

		assert.Equal(t, videoProducer.Id(), pipeProducer.Id())
		assert.False(t, pipeProducer.Closed())
		assert.Equal(t, MediaKindVideo, pipeProducer.Kind())
		assert.Zero(t, pipeProducer.RtpParameters().Mid)
		assert.Equal(t, []*RtpCodecParameters{
			{
				MimeType:    "video/VP8",
				ClockRate:   90000,
				PayloadType: 101,
				RtcpFeedback: []*RtcpFeedback{
					{Type: "nack", Parameter: "pli"},
					{Type: "ccm", Parameter: "fir"},
				},
			},
		}, pipeProducer.RtpParameters().Codecs)
		assert.Equal(t, []*RtpHeaderExtensionParameters{
			// TODO: Enable when DD is sendrecv.
			// {
			// 	Uri:     "https://aomediacodec.github.io/av1-rtp-spec/#dependency-descriptor-rtp-header-extension",
			// 	Id:      8,
			// 	Encrypt: false,
			// },
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
			{
				Uri:     "http://www.webrtc.org/experiments/rtp-hdrext/abs-capture-time",
				Id:      13,
				Encrypt: false,
			},
			{
				Uri:     "http://www.webrtc.org/experiments/rtp-hdrext/playout-delay",
				Id:      14,
				Encrypt: false,
			},
		}, pipeProducer.RtpParameters().HeaderExtensions)
		assert.True(t, pipeProducer.Paused())
	})

	t.Run("pipeToRouter fails if both Routers belong to the same Worker", func(t *testing.T) {
		router1bis := createRouter(worker1)
		_, err := router1.PipeToRouter(&PipeToRouterOptions{
			ProducerId: videoProducer.Id(),
			Router:     router1bis,
		})
		assert.Error(t, err)
		router1bis.Close()
	})

	t.Run("consume for a pipe Producer", func(t *testing.T) {
		router1.PipeToRouter(&PipeToRouterOptions{
			ProducerId: videoProducer.Id(),
			Router:     router2,
		})
		videoConsumer, _ := transport2.Consume(&ConsumerOptions{
			ProducerId:      videoProducer.Id(),
			RtpCapabilities: myDeviceCapabilities,
		})

		assert.False(t, videoConsumer.Closed())
		assert.Equal(t, MediaKindVideo, videoConsumer.Kind())
		assert.NotNil(t, videoConsumer.RtpParameters())
		assert.Equal(t, "0", videoConsumer.RtpParameters().Mid)
		assert.Equal(t, []*RtpCodecParameters{
			{
				MimeType:    "video/VP8",
				ClockRate:   90000,
				PayloadType: 101,
				RtcpFeedback: []*RtcpFeedback{
					{Type: "nack", Parameter: ""},
					{Type: "ccm", Parameter: "fir"},
					{Type: "transport-cc", Parameter: ""},
				},
			},
			{
				MimeType:    "video/rtx",
				ClockRate:   90000,
				PayloadType: 102,
				Parameters: RtpCodecSpecificParameters{
					Apt: 101,
				},
				RtcpFeedback: []*RtcpFeedback{},
			},
		}, videoConsumer.RtpParameters().Codecs)
		assert.Equal(t, []*RtpHeaderExtensionParameters{
			{
				Uri:     "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time",
				Id:      4,
				Encrypt: false,
			},
			{
				Uri:     "http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01",
				Id:      5,
				Encrypt: false,
			},
		}, videoConsumer.RtpParameters().HeaderExtensions)
		assert.Len(t, videoConsumer.RtpParameters().Encodings, 1)
		assert.NotNil(t, videoConsumer.RtpParameters().Encodings[0].Rtx)
		assert.NotZero(t, videoConsumer.RtpParameters().Encodings[0].Rtx.Ssrc)
		assert.Equal(t, ConsumerSimulcast, videoConsumer.Type())
		assert.False(t, videoConsumer.Paused())
		assert.True(t, videoConsumer.ProducerPaused())
		assert.Equal(t, ConsumerScore{
			Score:          10,
			ProducerScore:  0,
			ProducerScores: []int{0, 0, 0},
		}, videoConsumer.Score())
	})

	t.Run("producer pause and resume are transmitted to pipe Consumer", func(t *testing.T) {
		router1.PipeToRouter(&PipeToRouterOptions{
			ProducerId: videoProducer.Id(),
			Router:     router2,
		})
		videoConsumer, _ := transport2.Consume(&ConsumerOptions{
			ProducerId:      videoProducer.Id(),
			RtpCapabilities: myDeviceCapabilities,
		})

		assert.True(t, videoProducer.Paused())
		assert.True(t, videoConsumer.ProducerPaused())
		assert.False(t, videoConsumer.Paused())

		videoProducer.Resume()

		time.Sleep(time.Millisecond)

		assert.False(t, videoConsumer.ProducerPaused())
		assert.False(t, videoConsumer.Paused())

		videoProducer.Pause()

		time.Sleep(time.Millisecond)

		assert.True(t, videoConsumer.ProducerPaused())
		assert.False(t, videoConsumer.Paused())
	})

	t.Run("producer close is transmitted to pipe Consumer", func(t *testing.T) {
		router1.PipeToRouter(&PipeToRouterOptions{
			ProducerId: videoProducer.Id(),
			Router:     router2,
		})
		videoConsumer, _ := transport2.Consume(&ConsumerOptions{
			ProducerId:      videoProducer.Id(),
			RtpCapabilities: myDeviceCapabilities,
		})

		videoProducer.Close()
		assert.True(t, videoProducer.Closed())
		time.Sleep(time.Millisecond)
		assert.True(t, videoConsumer.Closed())
	})

	t.Run("data", func(t *testing.T) {
		// clear old transports
		for _, transports := range router1.mapRouterPipeTransports {
			for _, transport := range transports {
				transport.Close()
			}
		}
		result, err := router1.PipeToRouter(&PipeToRouterOptions{
			DataProducerId: dataProducer.Id(),
			Router:         router2,
		})
		assert.NoError(t, err)

		pipeDataConsumer, pipeDataProducer := result.PipeDataConsumer, result.PipeDataProducer
		dump, _ := router1.Dump()

		// There shoud be two Transports in router1:
		// - WebRtcTransport for audioProducer, videoProducer and dataProducer.
		// - PipeTransport between router1 and router2.
		assert.Len(t, dump.TransportIds, 2)

		dump, _ = router2.Dump()

		// There shoud be two Transports in router2:
		// - WebRtcTransport for audioConsumer, videoConsumer and dataConsumer.
		// - pipeTransport between router2 and router1.
		assert.Len(t, dump.TransportIds, 2)

		assert.NotZero(t, pipeDataConsumer.Id())
		assert.False(t, pipeDataConsumer.Closed())
		assert.Equal(t, DataConsumerSctp, pipeDataConsumer.Type())
		assert.NotNil(t, pipeDataConsumer.SctpStreamParameters())
		assert.False(t, *pipeDataConsumer.SctpStreamParameters().Ordered)
		assert.EqualValues(t, 5000, *pipeDataConsumer.SctpStreamParameters().MaxPacketLifeTime)
		assert.Zero(t, pipeDataConsumer.SctpStreamParameters().MaxRetransmits)
		assert.Equal(t, "foo", pipeDataConsumer.Label())
		assert.Equal(t, "bar", pipeDataConsumer.Protocol())

		assert.Equal(t, dataProducer.Id(), pipeDataProducer.Id())
		assert.False(t, pipeDataProducer.Closed())
		assert.Equal(t, DataProducerSctp, pipeDataProducer.Type())
		assert.NotNil(t, pipeDataProducer.SctpStreamParameters())
		assert.False(t, *pipeDataProducer.SctpStreamParameters().Ordered)
		assert.EqualValues(t, 5000, *pipeDataProducer.SctpStreamParameters().MaxPacketLifeTime)
		assert.Zero(t, pipeDataProducer.SctpStreamParameters().MaxRetransmits)
		assert.Equal(t, "foo", pipeDataProducer.Label())
		assert.Equal(t, "bar", pipeDataProducer.Protocol())
	})

	t.Run("dataConsume for a pipe DataProducer", func(t *testing.T) {
		router1.PipeToRouter(&PipeToRouterOptions{
			DataProducerId: dataProducer.Id(),
			Router:         router2,
		})
		dataConsumer, err := transport2.ConsumeData(&DataConsumerOptions{
			DataProducerId: dataProducer.Id(),
		})
		assert.NoError(t, err)
		assert.NotZero(t, dataConsumer.Id())
		assert.False(t, dataConsumer.Closed())
		assert.Equal(t, DataConsumerSctp, dataConsumer.Type())
		assert.NotNil(t, dataConsumer.SctpStreamParameters())
		assert.False(t, *dataConsumer.SctpStreamParameters().Ordered)
		assert.EqualValues(t, 5000, *dataConsumer.SctpStreamParameters().MaxPacketLifeTime)
		assert.Zero(t, dataConsumer.SctpStreamParameters().MaxRetransmits)
		assert.Equal(t, "foo", dataConsumer.Label())
		assert.Equal(t, "bar", dataConsumer.Protocol())
	})

	t.Run("dataProducer.close() is transmitted to pipe DataConsumer", func(t *testing.T) {
		router1.PipeToRouter(&PipeToRouterOptions{
			DataProducerId: dataProducer.Id(),
			Router:         router2,
		})

		dataConsumer, _ := transport2.ConsumeData(&DataConsumerOptions{
			DataProducerId: dataProducer.Id(),
		})

		dataProducer.Close()

		time.Sleep(time.Millisecond * 10)

		assert.True(t, dataProducer.Closed())
		assert.True(t, dataConsumer.Closed())
	})

	t.Run("PipeToRouter called twice generates a single PipeTransport pair", func(t *testing.T) {
		routerA := createRouter(nil)
		routerB := createRouter(nil)
		defer routerA.Close()
		defer routerB.Close()

		transportA1 := createWebRtcTransport(routerA)
		transportA2 := createWebRtcTransport(routerA)
		audioProducerA1 := createAudioProducer(transportA1)
		audioProducerA2 := createAudioProducer(transportA2)

		_, err := routerA.PipeToRouter(&PipeToRouterOptions{
			ProducerId: audioProducerA1.Id(),
			Router:     routerB,
		})
		assert.NoError(t, err)

		_, err = routerA.PipeToRouter(&PipeToRouterOptions{
			ProducerId: audioProducerA2.Id(),
			Router:     routerB,
		})
		assert.NoError(t, err)

		dump, _ := routerA.Dump()

		// There shoud be 3 Transports in routerA:
		// - WebRtcTransport for audioProducer1 and audioProducer2.
		// - PipeTransport between routerA and routerB.
		assert.Len(t, dump.TransportIds, 3)

		dump, _ = routerB.Dump()

		// There shoud be 1 Transport in routerB:
		// - PipeTransport between routerA and routerB.
		assert.Len(t, dump.TransportIds, 1)
	})

	t.Run("PipeToRouter called in two Routers passing one to each other as argument generates a single a single PipeTransport pair", func(t *testing.T) {
		routerA := createRouter(nil)
		routerB := createRouter(nil)
		defer routerB.Close()

		transportA := createWebRtcTransport(routerA)
		transportB := createWebRtcTransport(routerB)
		audioProducerA := createAudioProducer(transportA)
		audioProducerB := createAudioProducer(transportB)

		group := new(errgroup.Group)
		group.Go(func() error {
			_, err := routerA.PipeToRouter(&PipeToRouterOptions{
				ProducerId: audioProducerA.Id(),
				Router:     routerB,
			})
			return err
		})
		group.Go(func() error {
			_, err := routerB.PipeToRouter(&PipeToRouterOptions{
				ProducerId: audioProducerB.Id(),
				Router:     routerA,
			})
			return err
		})
		assert.NoError(t, group.Wait())

		assert.Len(t, routerA.mapRouterPipeTransports, 1)
		assert.Len(t, routerB.mapRouterPipeTransports, 1)

		pipeTransports := routerA.mapRouterPipeTransports[routerB]
		pipeTransportA := pipeTransports[0]
		pipeTransports = routerB.mapRouterPipeTransports[routerA]
		pipeTransportB := pipeTransports[0]

		dataA := pipeTransportA.Data().PipeTransportData
		dataB := pipeTransportB.Data().PipeTransportData

		assert.Equal(t, dataA.Tuple.LocalPort, dataB.Tuple.RemotePort)
		assert.Equal(t, dataB.Tuple.LocalPort, dataA.Tuple.RemotePort)

		routerA.Close()

		time.Sleep(time.Millisecond)

		assert.Empty(t, routerA.mapRouterPipeTransports)
		assert.Empty(t, routerB.mapRouterPipeTransports)
	})
}
