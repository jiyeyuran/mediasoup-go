package mediasoup

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/errgroup"
)

func TestPipeTransportTestingSuite(t *testing.T) {
	suite.Run(t, new(PipeTransportTestingSuite))
}

type PipeTransportTestingSuite struct {
	TestingSuite
	router1       *Router
	router2       *Router
	transport1    ITransport
	transport2    ITransport
	audioProducer *Producer
	videoProducer *Producer
	dataProducer  *DataProducer
}

func (suite *PipeTransportTestingSuite) SetupTest() {
	suite.router1 = CreateRouter()
	suite.router2 = CreateRouter()

	suite.transport1, _ = suite.router1.CreateWebRtcTransport(WebRtcTransportOptions{
		ListenIps: []TransportListenIp{
			{Ip: "127.0.0.1"},
		},
		EnableSctp: true,
	})
	suite.transport2, _ = suite.router2.CreateWebRtcTransport(WebRtcTransportOptions{
		ListenIps: []TransportListenIp{
			{Ip: "127.0.0.1"},
		},
		EnableSctp: true,
	})
	suite.audioProducer = CreateAudioProducer(suite.transport1)
	suite.videoProducer = CreateVP8Producer(suite.transport1)
	suite.dataProducer, _ = suite.transport1.ProduceData(DataProducerOptions{
		SctpStreamParameters: &SctpStreamParameters{
			StreamId:          666,
			Ordered:           Bool(false),
			MaxPacketLifeTime: 5000,
		},
		Label:    "foo",
		Protocol: "bar",
	})

	// Pause the videoProducer.
	suite.videoProducer.Pause()
}

func (suite *PipeTransportTestingSuite) TearDownTest() {
	suite.router1.Close()
	suite.router2.Close()
}

func (suite *PipeTransportTestingSuite) TestRouterPipeToRouter_SucceedsWithAudio() {
	result, err := suite.router1.PipeToRouter(PipeToRouterOptions{
		ProducerId: suite.audioProducer.Id(),
		Router:     suite.router2,
	})
	suite.NoError(err)

	pipeConsumer, pipeProducer := result.PipeConsumer, result.PipeProducer

	dump, _ := suite.router1.Dump()

	// There shoud should be two Transports in router1:
	// - WebRtcTransport for audioProducer and videoProducer.
	// - PipeTransport between router1 and router2.
	suite.Len(dump.TransportIds, 2)

	dump, _ = suite.router1.Dump()

	// There shoud should be two Transports in router2:
	// - WebRtcTransport for audioConsumer and videoConsumer.
	// - pipeTransport between router2 and router1.
	suite.Len(dump.TransportIds, 2)

	suite.False(pipeConsumer.Closed())
	suite.EqualValues("audio", pipeConsumer.Kind())
	suite.NotNil(pipeConsumer.RtpParameters())
	suite.Zero(pipeConsumer.RtpParameters().Mid)
	suite.EqualValues([]*RtpCodecParameters{
		{
			MimeType:     "audio/opus",
			ClockRate:    48000,
			PayloadType:  100,
			Channels:     2,
			Parameters:   RtpCodecSpecificParameters{Useinbandfec: 1, Usedtx: 1},
			RtcpFeedback: []RtcpFeedback{},
		},
	}, pipeConsumer.RtpParameters().Codecs)
	suite.EqualValues([]RtpHeaderExtensionParameters{
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
	}, pipeConsumer.RtpParameters().HeaderExtensions)
	suite.EqualValues("pipe", pipeConsumer.Type())
	suite.False(pipeConsumer.Paused())
	suite.False(pipeConsumer.ProducerPaused())
	suite.Equal(&ConsumerScore{
		Score:          10,
		ProducerScore:  10,
		ProducerScores: []uint16{},
	}, pipeConsumer.Score())

	suite.Equal(suite.audioProducer.Id(), pipeProducer.Id())
	suite.False(pipeProducer.Closed())
	suite.EqualValues("audio", pipeProducer.Kind())
	suite.Zero(pipeProducer.RtpParameters().Mid)
	suite.EqualValues([]*RtpCodecParameters{
		{
			MimeType:     "audio/opus",
			ClockRate:    48000,
			PayloadType:  100,
			Channels:     2,
			Parameters:   RtpCodecSpecificParameters{Useinbandfec: 1, Usedtx: 1},
			RtcpFeedback: []RtcpFeedback{},
		},
	}, pipeProducer.RtpParameters().Codecs)
	suite.EqualValues([]RtpHeaderExtensionParameters{
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
	}, pipeProducer.RtpParameters().HeaderExtensions)
	suite.False(pipeProducer.Paused())
}

func (suite *PipeTransportTestingSuite) TestRouterPipeToRouter_SucceedsWithVideo() {
	result, err := suite.router1.PipeToRouter(PipeToRouterOptions{
		ProducerId: suite.videoProducer.Id(),
		Router:     suite.router2,
	})
	suite.NoError(err)

	pipeConsumer, pipeProducer := result.PipeConsumer, result.PipeProducer

	dump, _ := suite.router1.Dump()

	// There shoud should be two Transports in router1:
	// - WebRtcTransport for audioProducer and videoProducer.
	// - PipeTransport between router1 and router2.
	suite.Len(dump.TransportIds, 2)

	dump, _ = suite.router1.Dump()

	// There shoud should be two Transports in router2:
	// - WebRtcTransport for audioConsumer and videoConsumer.
	// - pipeTransport between router2 and router1.
	suite.Len(dump.TransportIds, 2)

	suite.False(pipeConsumer.Closed())
	suite.EqualValues("video", pipeConsumer.Kind())
	suite.NotNil(pipeConsumer.RtpParameters())
	suite.Zero(pipeConsumer.RtpParameters().Mid)
	suite.EqualValues([]*RtpCodecParameters{
		{
			MimeType:    "video/VP8",
			ClockRate:   90000,
			PayloadType: 101,
			RtcpFeedback: []RtcpFeedback{
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
			},
		},
	}, pipeConsumer.RtpParameters().Codecs)
	suite.EqualValues([]RtpHeaderExtensionParameters{
		{
			Uri:     "http://tools.ietf.org/html/draft-ietf-avtext-framemarking-07",
			Id:      6,
			Encrypt: false,
		},
		{
			Uri:     "urn:ietf:params:rtp-hdrext:framemarking",
			Id:      7,
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
		{
			Uri:     "http://www.webrtc.org/experiments/rtp-hdrext/abs-capture-time",
			Id:      13,
			Encrypt: false,
		},
	}, pipeConsumer.RtpParameters().HeaderExtensions)
	suite.EqualValues("pipe", pipeConsumer.Type())
	suite.False(pipeConsumer.Paused())
	suite.True(pipeConsumer.ProducerPaused())
	suite.Equal(&ConsumerScore{
		Score:          10,
		ProducerScore:  10,
		ProducerScores: []uint16{},
	}, pipeConsumer.Score())

	suite.Equal(suite.videoProducer.Id(), pipeProducer.Id())
	suite.False(pipeProducer.Closed())
	suite.EqualValues("video", pipeProducer.Kind())
	suite.Zero(pipeProducer.RtpParameters().Mid)
	suite.EqualValues([]*RtpCodecParameters{
		{
			MimeType:    "video/VP8",
			ClockRate:   90000,
			PayloadType: 101,
			RtcpFeedback: []RtcpFeedback{
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
			},
		},
	}, pipeProducer.RtpParameters().Codecs)
	suite.EqualValues([]RtpHeaderExtensionParameters{
		{
			Uri:     "http://tools.ietf.org/html/draft-ietf-avtext-framemarking-07",
			Id:      6,
			Encrypt: false,
		},
		{
			Uri:     "urn:ietf:params:rtp-hdrext:framemarking",
			Id:      7,
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
		{
			Uri:     "http://www.webrtc.org/experiments/rtp-hdrext/abs-capture-time",
			Id:      13,
			Encrypt: false,
		},
	}, pipeProducer.RtpParameters().HeaderExtensions)
	suite.True(pipeProducer.Paused())
}

func (suite *PipeTransportTestingSuite) TestRouterCreatePipeTransport_WithEnableRtxSucceeds() {
	pipeTransport, err := suite.router1.CreatePipeTransport(PipeTransportOptions{
		ListenIp:  TransportListenIp{Ip: "127.0.0.1"},
		EnableRtx: true,
	})
	suite.NoError(err)
	suite.Empty(pipeTransport.SrtpParameters())

	// No SRTP enabled so passing srtpParameters must fail.
	err = pipeTransport.Connect(TransportConnectOptions{
		Ip:   "127.0.0.2",
		Port: 9999,
		SrtpParameters: &SrtpParameters{
			CryptoSuite: "AEAD_AES_256_GCM",
			KeyBase64:   "YTdjcDBvY2JoMGY5YXNlNDc0eDJsdGgwaWRvNnJsamRrdG16aWVpZHphdHo=",
		},
	})
	suite.Error(err)

	pipeConsumer, err := pipeTransport.Consume(ConsumerOptions{
		ProducerId: suite.videoProducer.Id(),
	})
	suite.NoError(err)
	suite.False(pipeConsumer.Closed())
	suite.EqualValues("video", pipeConsumer.Kind())
	suite.NotNil(pipeConsumer.RtpParameters())
	suite.Zero(pipeConsumer.RtpParameters().Mid)
	suite.EqualValues([]*RtpCodecParameters{
		{
			MimeType:    "video/VP8",
			ClockRate:   90000,
			PayloadType: 101,
			RtcpFeedback: []RtcpFeedback{
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
		},
	}, pipeConsumer.RtpParameters().Codecs)
	suite.EqualValues([]RtpHeaderExtensionParameters{
		{
			Uri:     "http://tools.ietf.org/html/draft-ietf-avtext-framemarking-07",
			Id:      6,
			Encrypt: false,
		},
		{
			Uri:     "urn:ietf:params:rtp-hdrext:framemarking",
			Id:      7,
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
		{
			Uri:     "http://www.webrtc.org/experiments/rtp-hdrext/abs-capture-time",
			Id:      13,
			Encrypt: false,
		},
	}, pipeConsumer.RtpParameters().HeaderExtensions)
	suite.EqualValues("pipe", pipeConsumer.Type())
	suite.False(pipeConsumer.Paused())
	suite.True(pipeConsumer.ProducerPaused())
	suite.Equal(&ConsumerScore{
		Score:          10,
		ProducerScore:  10,
		ProducerScores: []uint16{},
	}, pipeConsumer.Score())

	pipeTransport.Close()
}

func (suite *PipeTransportTestingSuite) TestRouterCreatePipeTransport_WithEnableSrtpSucceeds() {
	pipeTransport, err := suite.router1.CreatePipeTransport(PipeTransportOptions{
		ListenIp:   TransportListenIp{Ip: "127.0.0.1"},
		EnableSrtp: true,
	})
	suite.NoError(err)
	// The master length of AEAD_AES_256_GCM.
	suite.Len(pipeTransport.SrtpParameters().KeyBase64, 60)

	// Missing srtpParameters.
	err = pipeTransport.Connect(TransportConnectOptions{
		Ip:   "127.0.0.2",
		Port: 9999,
	})
	suite.Error(err)

	// Missing srtpParameters.cryptoSuite.
	err = pipeTransport.Connect(TransportConnectOptions{
		Ip:   "127.0.0.2",
		Port: 9999,
		SrtpParameters: &SrtpParameters{
			KeyBase64: "YTdjcDBvY2JoMGY5YXNlNDc0eDJsdGgwaWRvNnJsamRrdG16aWVpZHphdHo=",
		},
	})
	suite.Error(err)

	// Invalid srtpParameters.keyBase64.
	err = pipeTransport.Connect(TransportConnectOptions{
		Ip:   "127.0.0.2",
		Port: 9999,
		SrtpParameters: &SrtpParameters{
			CryptoSuite: "AEAD_AES_256_GCM",
		},
	})
	suite.Error(err)

	// Invalid srtpParameters.cryptoSuite.
	err = pipeTransport.Connect(TransportConnectOptions{
		Ip:   "127.0.0.2",
		Port: 9999,
		SrtpParameters: &SrtpParameters{
			CryptoSuite: "FOO",
			KeyBase64:   "YTdjcDBvY2JoMGY5YXNlNDc0eDJsdGgwaWRvNnJsamRrdG16aWVpZHphdHo=",
		},
	})
	suite.Error(err)

	// Valid srtpParameters.
	err = pipeTransport.Connect(TransportConnectOptions{
		Ip:   "127.0.0.2",
		Port: 9999,
		SrtpParameters: &SrtpParameters{
			CryptoSuite: "AEAD_AES_256_GCM",
			KeyBase64:   "YTdjcDBvY2JoMGY5YXNlNDc0eDJsdGgwaWRvNnJsamRrdG16aWVpZHphdHo=",
		},
	})
	suite.NoError(err)

	pipeTransport.Close()
}

func (suite *PipeTransportTestingSuite) TestTransportConsume_ForAPipeProducerSucceeds() {
	_, err := suite.router1.PipeToRouter(PipeToRouterOptions{
		ProducerId: suite.videoProducer.Id(),
		Router:     suite.router2,
	})
	suite.NoError(err)
	videoConsumer, err := suite.transport2.Consume(ConsumerOptions{
		ProducerId:      suite.videoProducer.Id(),
		RtpCapabilities: consumerDeviceCapabilities,
	})
	suite.NoError(err)

	suite.NoError(err)
	suite.False(videoConsumer.Closed())
	suite.EqualValues("video", videoConsumer.Kind())
	suite.NotNil(videoConsumer.RtpParameters())
	suite.Equal("0", videoConsumer.RtpParameters().Mid)
	suite.EqualValues([]*RtpCodecParameters{
		{
			MimeType:    "video/VP8",
			ClockRate:   90000,
			PayloadType: 101,
			RtcpFeedback: []RtcpFeedback{
				{Type: "nack", Parameter: ""},
				{Type: "ccm", Parameter: "fir"},
				{Type: "google-remb", Parameter: ""},
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
			RtcpFeedback: []RtcpFeedback{},
		},
	}, videoConsumer.RtpParameters().Codecs)
	suite.EqualValues([]RtpHeaderExtensionParameters{
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
	suite.Len(videoConsumer.RtpParameters().Encodings, 1)
	suite.NotNil(videoConsumer.RtpParameters().Encodings[0].Rtx)
	suite.NotZero(videoConsumer.RtpParameters().Encodings[0].Rtx.Ssrc)
	suite.EqualValues("simulcast", videoConsumer.Type())
	suite.False(videoConsumer.Paused())
	suite.True(videoConsumer.ProducerPaused())
	suite.Equal(&ConsumerScore{
		Score:          10,
		ProducerScore:  0,
		ProducerScores: []uint16{0, 0, 0},
	}, videoConsumer.Score())
}

func (suite *PipeTransportTestingSuite) TestProducerPauseAndProducerResumeAreTransmittedToPipeConsumer() {
	_, err := suite.router1.PipeToRouter(PipeToRouterOptions{
		ProducerId: suite.videoProducer.Id(),
		Router:     suite.router2,
	})
	suite.NoError(err)
	videoConsumer, err := suite.transport2.Consume(ConsumerOptions{
		ProducerId:      suite.videoProducer.Id(),
		RtpCapabilities: consumerDeviceCapabilities,
	})
	suite.NoError(err)

	suite.True(suite.videoProducer.Paused())
	suite.True(videoConsumer.ProducerPaused())
	suite.False(videoConsumer.Paused())

	observer := NewMockFunc(suite.T())
	videoConsumer.Once("producerresume", observer.Fn())

	suite.videoProducer.Resume()

	observer.ExpectCalled()
	suite.False(videoConsumer.ProducerPaused())
	suite.False(videoConsumer.Paused())

	videoConsumer.Once("producerpause", observer.Fn())

	suite.videoProducer.Pause()

	observer.ExpectCalled()
	suite.True(videoConsumer.ProducerPaused())
	suite.False(videoConsumer.Paused())
}

func (suite *PipeTransportTestingSuite) TestProducerCloseIsTransmittedToPipeConsumer() {
	_, err := suite.router1.PipeToRouter(PipeToRouterOptions{
		ProducerId: suite.videoProducer.Id(),
		Router:     suite.router2,
	})
	suite.NoError(err)
	videoConsumer, err := suite.transport2.Consume(ConsumerOptions{
		ProducerId:      suite.videoProducer.Id(),
		RtpCapabilities: consumerDeviceCapabilities,
	})
	suite.NoError(err)

	observer := NewMockFunc(suite.T())
	videoConsumer.Once("producerclose", observer.Fn())

	suite.videoProducer.Close()

	suite.True(suite.videoProducer.Closed())
	suite.True(videoConsumer.Closed())
	suite.Equal(1, observer.CalledTimes())
}

func (suite *PipeTransportTestingSuite) TestProducerPipeRouterSucceedsWithData() {
	result, err := suite.router1.PipeToRouter(PipeToRouterOptions{
		DataProducerId: suite.dataProducer.Id(),
		Router:         suite.router2,
		EnableSctp:     true,
	})
	suite.NoError(err)

	pipeDataConsumer, pipeDataProducer := result.PipeDataConsumer, result.PipeDataProducer
	dump, _ := suite.router1.Dump()

	// There shoud be two Transports in router1:
	// - WebRtcTransport for audioProducer, videoProducer and dataProducer.
	// - PipeTransport between router1 and router2.
	suite.Len(dump.TransportIds, 2)

	dump, _ = suite.router1.Dump()

	// There shoud be two Transports in router2:
	// - WebRtcTransport for audioConsumer, videoConsumer and dataConsumer.
	// - pipeTransport between router2 and router1.
	suite.Len(dump.TransportIds, 2)

	suite.NotZero(pipeDataConsumer.Id())
	suite.False(pipeDataConsumer.Closed())
	suite.EqualValues("sctp", pipeDataConsumer.Type())
	suite.NotNil(pipeDataConsumer.SctpStreamParameters())
	suite.False(*pipeDataConsumer.SctpStreamParameters().Ordered)
	suite.EqualValues(5000, pipeDataConsumer.SctpStreamParameters().MaxPacketLifeTime)
	suite.Zero(pipeDataConsumer.SctpStreamParameters().MaxRetransmits)
	suite.Equal("foo", pipeDataConsumer.Label())
	suite.Equal("bar", pipeDataConsumer.Protocol())

	suite.Equal(suite.dataProducer.Id(), pipeDataProducer.Id())
	suite.False(pipeDataProducer.Closed())
	suite.EqualValues("sctp", pipeDataProducer.Type())
	suite.NotNil(pipeDataProducer.SctpStreamParameters())
	suite.False(*pipeDataProducer.SctpStreamParameters().Ordered)
	suite.EqualValues(5000, pipeDataProducer.SctpStreamParameters().MaxPacketLifeTime)
	suite.Zero(pipeDataProducer.SctpStreamParameters().MaxRetransmits)
	suite.Equal("foo", pipeDataProducer.Label())
	suite.Equal("bar", pipeDataProducer.Protocol())
}

func (suite *PipeTransportTestingSuite) TestTransportDataConsumeForAPipeDataProducerSucceeds() {
	_, err := suite.router1.PipeToRouter(PipeToRouterOptions{
		DataProducerId: suite.dataProducer.Id(),
		Router:         suite.router2,
		EnableSctp:     true,
	})
	suite.NoError(err)
	dataConsumer, err := suite.transport2.ConsumeData(DataConsumerOptions{
		DataProducerId: suite.dataProducer.Id(),
	})
	suite.NoError(err)
	suite.NotZero(dataConsumer.Id())
	suite.False(dataConsumer.Closed())
	suite.EqualValues("sctp", dataConsumer.Type())
	suite.NotNil(dataConsumer.SctpStreamParameters())
	suite.False(*dataConsumer.SctpStreamParameters().Ordered)
	suite.EqualValues(5000, dataConsumer.SctpStreamParameters().MaxPacketLifeTime)
	suite.Zero(dataConsumer.SctpStreamParameters().MaxRetransmits)
	suite.Equal("foo", dataConsumer.Label())
	suite.Equal("bar", dataConsumer.Protocol())
}

func (suite *PipeTransportTestingSuite) TestDataProducerCloseIsTransmittedToPipeDataConsumer() {
	_, err := suite.router1.PipeToRouter(PipeToRouterOptions{
		DataProducerId: suite.dataProducer.Id(),
		Router:         suite.router2,
		EnableSctp:     true,
	})
	suite.NoError(err)
	dataConsumer, err := suite.transport2.ConsumeData(DataConsumerOptions{
		DataProducerId: suite.dataProducer.Id(),
	})
	suite.NoError(err)

	observer := NewMockFunc(suite.T())
	dataConsumer.Once("dataproducerclose", observer.Fn())

	suite.dataProducer.Close()

	suite.True(suite.dataProducer.Closed())
	observer.ExpectCalled()
	suite.True(dataConsumer.Closed())
}

func (suite *PipeTransportTestingSuite) TestPipeToRouter() {
	t := suite.T()
	t.Run(`router.pipeToRouter() called twice generates a single PipeTransport pair`, func(t *testing.T) {
		routerA := CreateRouter()
		routerB := CreateRouter()
		defer routerA.Close()
		defer routerB.Close()

		transportA1, _ := routerA.CreateWebRtcTransport(WebRtcTransportOptions{
			ListenIps: []TransportListenIp{{Ip: "127.0.0.1"}},
		})
		transportA2, _ := routerA.CreateWebRtcTransport(WebRtcTransportOptions{
			ListenIps: []TransportListenIp{{Ip: "127.0.0.1"}},
		})
		audioProducerA1 := CreateAudioProducer(transportA1)
		audioProducerA2 := CreateAudioProducer(transportA2)

		_, err := routerA.PipeToRouter(PipeToRouterOptions{
			ProducerId: audioProducerA1.Id(),
			Router:     routerB,
		})
		suite.NoError(err)
		_, err = routerA.PipeToRouter(PipeToRouterOptions{
			ProducerId: audioProducerA2.Id(),
			Router:     routerB,
		})
		suite.NoError(err)

		dump, _ := routerA.Dump()

		// There shoud be 3 Transports in routerA:
		// - WebRtcTransport for audioProducer1 and audioProducer2.
		// - PipeTransport between routerA and routerB.
		suite.Len(dump.TransportIds, 3)

		dump, _ = routerB.Dump()

		// There shoud be 1 Transport in routerB:
		// - PipeTransport between routerA and routerB.
		suite.Len(dump.TransportIds, 1)
	})

	t.Run("router.pipeToRouter() called in two Routers passing one to each other as argument generates a single a single PipeTransport pair", func(t *testing.T) {
		routerA := CreateRouter()
		routerB := CreateRouter()
		defer routerB.Close()

		transportA, _ := routerA.CreateWebRtcTransport(WebRtcTransportOptions{
			ListenIps: []TransportListenIp{{Ip: "127.0.0.1"}},
		})
		transportB, _ := routerB.CreateWebRtcTransport(WebRtcTransportOptions{
			ListenIps: []TransportListenIp{{Ip: "127.0.0.1"}},
		})
		audioProducerA := CreateAudioProducer(transportA)
		audioProducerB := CreateAudioProducer(transportB)
		pipeTransportsA := sync.Map{}
		pipeTransportsB := sync.Map{}

		routerA.Observer().On("newtransport", func(transport ITransport) {
			if _, ok := transport.(*PipeTransport); !ok {
				return
			}
			pipeTransportsA.Store(transport.Id(), transport)
			transport.Observer().On("close", func() {
				pipeTransportsA.Delete(transport.Id())
			})
		})
		routerB.Observer().On("newtransport", func(transport ITransport) {
			if _, ok := transport.(*PipeTransport); !ok {
				return
			}
			pipeTransportsB.Store(transport.Id(), transport)
			transport.Observer().On("close", func() {
				pipeTransportsB.Delete(transport.Id())
			})
		})

		group := new(errgroup.Group)
		group.Go(func() error {
			_, err := routerA.PipeToRouter(PipeToRouterOptions{
				ProducerId: audioProducerA.Id(),
				Router:     routerB,
			})
			return err
		})
		group.Go(func() error {
			_, err := routerB.PipeToRouter(PipeToRouterOptions{
				ProducerId: audioProducerB.Id(),
				Router:     routerA,
			})
			return err
		})
		suite.NoError(group.Wait())

		suite.EqualValues(1, syncMapLen(&pipeTransportsA))
		suite.EqualValues(1, syncMapLen(&pipeTransportsB))

		var pipeTransportA, pipeTransportB *PipeTransport

		pipeTransportsA.Range(func(key, value interface{}) bool {
			pipeTransportA = value.(*PipeTransport)
			return false
		})
		pipeTransportsB.Range(func(key, value interface{}) bool {
			pipeTransportB = value.(*PipeTransport)
			return false
		})
		suite.Equal(pipeTransportA.Tuple().LocalPort, pipeTransportB.Tuple().RemotePort)
		suite.Equal(pipeTransportB.Tuple().LocalPort, pipeTransportA.Tuple().RemotePort)

		routerA.Close()

		time.Sleep(time.Millisecond * 10)
		suite.EqualValues(0, syncMapLen(&pipeTransportsA))
		suite.EqualValues(0, syncMapLen(&pipeTransportsB))
	})
}
