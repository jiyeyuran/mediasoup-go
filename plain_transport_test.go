package mediasoup

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestPlainTransportTestingSuite(t *testing.T) {
	suite.Run(t, new(PlainTransportTestingSuite))
}

type PlainTransportTestingSuite struct {
	TestingSuite
	router    *Router
	transport *PlainTransport
}

func (suite *PlainTransportTestingSuite) SetupTest() {
	suite.router = CreateRouter()
	suite.transport, _ = suite.router.CreatePlainTransport(func(o *PlainTransportOptions) {
		o.ListenIp = TransportListenIp{
			Ip:          "127.0.0.1",
			AnnouncedIp: "4.4.4.4",
		}
		o.RtcpMux = Bool(false)
	})
}

func (suite *PlainTransportTestingSuite) TearDownTest() {
	suite.router.Close()
}

func (suite *PlainTransportTestingSuite) TestCreatePlainTransport_Succeeds() {
	router := suite.router
	dump, _ := suite.router.Dump()
	suite.Equal([]string{suite.transport.Id()}, dump.TransportIds)

	appData := H{"foo": "bar"}
	onObserverNewTransport := NewMockFunc(suite.T())
	router.Observer().Once("newtransport", onObserverNewTransport.Fn())
	transport1, err := router.CreatePlainTransport(func(o *PlainTransportOptions) {
		o.ListenIp = TransportListenIp{
			Ip:          "127.0.0.1",
			AnnouncedIp: "9.9.9.1",
		}
		o.RtcpMux = Bool(true)
		o.EnableSctp = true
		o.AppData = appData
	})
	suite.NoError(err)

	onObserverNewTransport.ExpectCalled()
	onObserverNewTransport.ExpectCalledWith(transport1)
	suite.False(transport1.Closed())
	suite.Equal(appData, transport1.AppData())
	suite.Equal("9.9.9.1", transport1.Tuple().LocalIp)
	suite.NotZero(transport1.Tuple().LocalPort)
	suite.EqualValues("udp", transport1.Tuple().Protocol)
	suite.Empty(transport1.RtcpTuple())
	suite.Equal(SctpParameters{
		Port:               5000,
		OS:                 1024,
		MIS:                1024,
		MaxMessageSize:     262144,
		IsDataChannel:      false,
		SctpBufferedAmount: 0,
		SendBufferSize:     262144,
	}, transport1.SctpParameters())
	suite.EqualValues("new", transport1.SctpState())
	suite.Empty(transport1.SrtpParameters())

	data1, _ := transport1.Dump()

	suite.Equal(transport1.Id(), data1.Id)
	suite.False(data1.Direct)
	suite.Empty(data1.ProducerIds)
	suite.Empty(data1.ConsumerIds)
	suite.Equal(transport1.Tuple(), data1.Tuple)
	suite.Equal(transport1.RtcpTuple(), data1.RtcpTuple)
	suite.Equal(transport1.SctpParameters(), data1.SctpParameters)
	suite.EqualValues("new", data1.SctpState)
	suite.NotNil(data1.RecvRtpHeaderExtensions)
	suite.NotNil(data1.RtpListener)

	transport1.Close()
	suite.True(transport1.Closed())

	_, err = router.CreatePlainTransport(func(o *PlainTransportOptions) {
		o.ListenIp = TransportListenIp{
			Ip: "127.0.0.1",
		}
	})
	suite.NoError(err)

	transport2, err := router.CreatePlainTransport(func(o *PlainTransportOptions) {
		o.ListenIp = TransportListenIp{
			Ip: "127.0.0.1",
		}
		o.RtcpMux = Bool(false)
	})
	suite.NoError(err)
	suite.False(transport2.Closed())
	suite.Equal(H{}, transport2.AppData())
	suite.Equal("127.0.0.1", transport2.Tuple().LocalIp)
	suite.NotZero(transport2.Tuple().LocalPort)
	suite.Equal("udp", transport2.Tuple().Protocol)
	suite.Equal("127.0.0.1", transport2.RtcpTuple().LocalIp)
	suite.NotZero(transport2.RtcpTuple().LocalPort)
	suite.Equal("udp", transport2.RtcpTuple().Protocol)
	suite.Empty(transport2.SctpParameters())
	suite.Empty(transport2.SctpState())

	data2, _ := transport2.Dump()

	suite.Equal(transport2.Id(), data2.Id)
	suite.False(data2.Direct)
	suite.Equal(transport2.Tuple(), data2.Tuple)
	suite.Equal(transport2.RtcpTuple(), data2.RtcpTuple)
	suite.Empty(data2.SctpState)
}

func (suite *PlainTransportTestingSuite) TestCreatePlainTransport_TypeError() {
	_, err := suite.router.CreatePlainTransport()
	suite.IsType(NewTypeError(""), err)

	_, err = suite.router.CreatePlainTransport(func(o *PlainTransportOptions) {
		o.ListenIp = TransportListenIp{
			Ip: "123",
		}
	})
	suite.IsType(NewTypeError(""), err)
}

func (suite *PlainTransportTestingSuite) TestCreatePlainTransport_EnableSrtpSucceeds() {
	router := suite.router

	transport1, _ := router.CreatePlainTransport(func(o *PlainTransportOptions) {
		o.ListenIp = TransportListenIp{
			Ip: "127.0.0.1",
		}
		o.EnableSrtp = true
	})

	suite.NotNil(transport1.SrtpParameters())
	suite.EqualValues("AES_CM_128_HMAC_SHA1_80", transport1.SrtpParameters().CryptoSuite)
	suite.Len(transport1.SrtpParameters().KeyBase64, 40)

	// Missing srtpParameters.
	err := transport1.Connect(TransportConnectOptions{
		Ip:   "127.0.0.2",
		Port: 9999,
	})
	suite.Error(err)

	// Missing srtpParameters.cryptoSuite.
	err = transport1.Connect(TransportConnectOptions{
		Ip:   "127.0.0.2",
		Port: 9999,
		SrtpParameters: &SrtpParameters{
			KeyBase64: "ZnQ3eWJraDg0d3ZoYzM5cXN1Y2pnaHU5NWxrZTVv",
		},
	})
	suite.Error(err)

	// Missing srtpParameters.keyBase64.
	err = transport1.Connect(TransportConnectOptions{
		Ip:   "127.0.0.2",
		Port: 9999,
		SrtpParameters: &SrtpParameters{
			CryptoSuite: "AES_CM_128_HMAC_SHA1_80",
		},
	})
	suite.Error(err)

	// Invalid srtpParameters.cryptoSuite.
	err = transport1.Connect(TransportConnectOptions{
		Ip:   "127.0.0.2",
		Port: 9999,
		SrtpParameters: &SrtpParameters{
			CryptoSuite: "FOO",
			KeyBase64:   "ZnQ3eWJraDg0d3ZoYzM5cXN1Y2pnaHU5NWxrZTVv",
		},
	})
	suite.Error(err)

	// Valid srtpParameters. And let"s update the crypto suite.
	err = transport1.Connect(TransportConnectOptions{
		Ip:   "127.0.0.2",
		Port: 9999,
		SrtpParameters: &SrtpParameters{
			CryptoSuite: "AES_CM_128_HMAC_SHA1_32",
			KeyBase64:   "ZnQ3eWJraDg0d3ZoYzM5cXN1Y2pnaHU5NWxrZTVv",
		},
	})
	suite.NoError(err)
	suite.EqualValues("AES_CM_128_HMAC_SHA1_32", transport1.SrtpParameters().CryptoSuite)

	transport1.Close()
}

func (suite *PlainTransportTestingSuite) TestCreatePlainTransport_NonBindableIpError() {
	router := suite.router
	_, err := router.CreatePlainTransport(func(o *PlainTransportOptions) {
		o.ListenIp = TransportListenIp{
			Ip: "8.8.8.8",
		}
	})
	suite.Error(err)
}

func (suite *PlainTransportTestingSuite) TestGetStats_Succeeds() {
	data, _ := suite.transport.GetStats()

	suite.Len(data, 1)
	suite.Equal("plain-rtp-transport", data[0].Type)
	suite.NotZero(data[0].Timestamp)
	suite.Zero(data[0].BytesReceived)
	suite.Zero(data[0].RecvBitrate)
	suite.Zero(data[0].BytesSent)
	suite.Zero(data[0].SendBitrate)
	suite.Zero(data[0].RtpRecvBitrate)
	suite.Zero(data[0].RtpBytesSent)
	suite.Zero(data[0].RtpSendBitrate)
	suite.Zero(data[0].RtxBytesReceived)
	suite.Zero(data[0].RtxRecvBitrate)
	suite.Zero(data[0].RtxSendBitrate)
	suite.Zero(data[0].ProbationBytesSent)
	suite.Zero(data[0].ProbationSendBitrate)
	suite.Equal("4.4.4.4", data[0].Tuple.LocalIp)
	suite.NotZero(data[0].Tuple.LocalPort)
	suite.Empty(data[0].RtcpTuple)
	suite.Zero(data[0].RecvBitrate)
	suite.Zero(data[0].SendBitrate)
}

func (suite *PlainTransportTestingSuite) TestConnect_Succeeds() {
	transport := suite.transport

	err := transport.Connect(TransportConnectOptions{
		Ip:       "1.2.3.4",
		Port:     1234,
		RtcpPort: 1235,
	})
	suite.NoError(err)

	// Must fail if connected.
	err = transport.Connect(TransportConnectOptions{
		Ip:       "1.2.3.4",
		Port:     1234,
		RtcpPort: 1235,
	})
	suite.Error(err)

	suite.Equal("1.2.3.4", transport.Tuple().RemoteIp)
	suite.EqualValues(1234, transport.Tuple().RemotePort)
	suite.Equal("udp", transport.Tuple().Protocol)
	suite.Equal("1.2.3.4", transport.RtcpTuple().RemoteIp)
	suite.EqualValues(1235, transport.RtcpTuple().RemotePort)
	suite.Equal("udp", transport.RtcpTuple().Protocol)
}

func (suite *PlainTransportTestingSuite) TestConnect_RejectsWithTypeError() {
	transport := suite.transport

	// No SRTP enabled so passing srtpParameters must fail.
	err := transport.Connect(TransportConnectOptions{
		Ip:       "127.0.0.2",
		Port:     9998,
		RtcpPort: 9999,
		SrtpParameters: &SrtpParameters{
			CryptoSuite: "AES_CM_128_HMAC_SHA1_80",
			KeyBase64:   "ZnQ3eWJraDg0d3ZoYzM5cXN1Y2pnaHU5NWxrZTVv",
		},
	})
	suite.Error(err)

	err = transport.Connect(TransportConnectOptions{})
	suite.Error(err)

	err = transport.Connect(TransportConnectOptions{
		Ip: "::::1234",
	})
	suite.Error(err)

	err = transport.Connect(TransportConnectOptions{
		Ip:   "127.0.0.1",
		Port: 1234,
	})
	suite.Error(err)

	err = transport.Connect(TransportConnectOptions{
		Ip:       "127.0.0.1",
		RtcpPort: 1235,
	})
	suite.Error(err)
}

func (suite *PlainTransportTestingSuite) TestMethodsRejectIfClosed() {
	transport := suite.transport
	onObserverClose := NewMockFunc(suite.T())

	transport.Observer().Once("close", onObserverClose.Fn())
	transport.Close()

	onObserverClose.ExpectCalled()
	suite.True(transport.Closed())

	_, err := transport.Dump()
	suite.Error(err)

	_, err = transport.GetStats()
	suite.Error(err)

	err = transport.Connect(TransportConnectOptions{})
	suite.Error(err)
}

func (suite *PlainTransportTestingSuite) TestEmitsRoutercloseIfRouterIsClosed() {
	router := suite.router
	transport := suite.transport
	onObserverClose := NewMockFunc(suite.T())
	transport.Observer().Once("close", onObserverClose.Fn())
	router.Close()
	onObserverClose.ExpectCalled()
	suite.True(transport.Closed())
}

func (suite *PlainTransportTestingSuite) TestEmitsRoutercloseIfWorkerIsClosed() {
	worker := CreateTestWorker()
	router := CreateRouter(worker)
	transport, _ := router.CreatePlainTransport(func(o *PlainTransportOptions) {
		o.ListenIp = TransportListenIp{
			Ip: "127.0.0.1",
		}
	})
	onObserverClose := NewMockFunc(suite.T())
	transport.Observer().Once("close", onObserverClose.Fn())
	worker.Close()
	onObserverClose.ExpectCalled()
	suite.True(transport.Closed())
}
