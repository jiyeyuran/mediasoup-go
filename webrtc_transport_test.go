package mediasoup

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestWebRtcTransportTestingSuite(t *testing.T) {
	suite.Run(t, new(WebRtcTransportTestingSuite))
}

type WebRtcTransportTestingSuite struct {
	TestingSuite
	router    *Router
	transport *WebRtcTransport
}

func (suite *WebRtcTransportTestingSuite) SetupTest() {
	suite.router = CreateRouter()
	suite.transport, _ = suite.router.CreateWebRtcTransport(WebRtcTransportOptions{
		ListenIps: []TransportListenIp{
			{Ip: "127.0.0.1", AnnouncedIp: "9.9.9.1"},
		},
		EnableTcp: false,
	})
}

func (suite *WebRtcTransportTestingSuite) TearDownTest() {
	suite.router.Close()
}

func (suite *WebRtcTransportTestingSuite) TestCreateWebRtcTransport_Succeeds() {
	router := suite.router
	dump, _ := suite.router.Dump()
	suite.Equal([]string{suite.transport.Id()}, dump.TransportIds)

	appData := H{"foo": "bar"}
	onObserverNewTransport := NewMockFunc(suite.T())
	router.Observer().Once("newtransport", onObserverNewTransport.Fn())
	transport1, err := router.CreateWebRtcTransport(WebRtcTransportOptions{
		ListenIps: []TransportListenIp{
			{Ip: "127.0.0.1", AnnouncedIp: "9.9.9.1"},
			{Ip: "0.0.0.0", AnnouncedIp: "9.9.9.2"},
			{Ip: "127.0.0.1"},
		},
		EnableTcp:          true,
		PreferUdp:          true,
		EnableSctp:         true,
		NumSctpStreams:     NumSctpStreams{OS: 2048, MIS: 2048},
		MaxSctpMessageSize: 1000000,
		AppData:            appData,
	})
	suite.NoError(err)

	onObserverNewTransport.ExpectCalled()
	onObserverNewTransport.ExpectCalledWith(transport1)
	suite.False(transport1.Closed())
	suite.Equal(appData, transport1.AppData())
	suite.EqualValues("controlled", transport1.IceRole())
	suite.True(transport1.IceParameters().IceLite)
	suite.NotEmpty(transport1.IceParameters().UsernameFragment)
	suite.NotEmpty(transport1.IceParameters().Password)
	suite.Equal(SctpParameters{
		Port:               5000,
		OS:                 2048,
		MIS:                2048,
		MaxMessageSize:     1000000,
		IsDataChannel:      true,
		SctpBufferedAmount: 0,
		SendBufferSize:     262144,
	}, transport1.SctpParameters())
	suite.Len(transport1.IceCandidates(), 6)

	iceCandidates := transport1.IceCandidates()

	suite.Equal(iceCandidates[0].Ip, "9.9.9.1")
	suite.EqualValues(iceCandidates[0].Protocol, "udp")
	suite.Equal(iceCandidates[0].Type, "host")
	suite.Empty(iceCandidates[0].TcpType)
	suite.Equal(iceCandidates[1].Ip, "9.9.9.1")
	suite.EqualValues(iceCandidates[1].Protocol, "tcp")
	suite.Equal(iceCandidates[1].Type, "host")
	suite.EqualValues(iceCandidates[1].TcpType, "passive")
	suite.Equal(iceCandidates[2].Ip, "9.9.9.2")
	suite.EqualValues(iceCandidates[2].Protocol, "udp")
	suite.Equal(iceCandidates[2].Type, "host")
	suite.Empty(iceCandidates[2].TcpType)
	suite.Equal(iceCandidates[3].Ip, "9.9.9.2")
	suite.EqualValues(iceCandidates[3].Protocol, "tcp")
	suite.Equal(iceCandidates[3].Type, "host")
	suite.Equal(iceCandidates[3].TcpType, "passive")
	suite.Equal(iceCandidates[4].Ip, "127.0.0.1")
	suite.EqualValues(iceCandidates[4].Protocol, "udp")
	suite.Equal(iceCandidates[4].Type, "host")
	suite.Empty(iceCandidates[4].TcpType)
	suite.Equal(iceCandidates[5].Ip, "127.0.0.1")
	suite.EqualValues(iceCandidates[5].Protocol, "tcp")
	suite.Equal(iceCandidates[5].Type, "host")
	suite.Equal(iceCandidates[5].TcpType, "passive")
	suite.Greater(iceCandidates[0].Priority, iceCandidates[1].Priority)
	suite.Greater(iceCandidates[2].Priority, iceCandidates[1].Priority)
	suite.Greater(iceCandidates[2].Priority, iceCandidates[3].Priority)
	suite.Greater(iceCandidates[4].Priority, iceCandidates[3].Priority)
	suite.Greater(iceCandidates[4].Priority, iceCandidates[5].Priority)

	suite.EqualValues("new", transport1.IceState())
	suite.Nil(transport1.IceSelectedTuple())
	suite.NotEmpty(transport1.DtlsParameters())
	suite.EqualValues("auto", transport1.DtlsParameters().Role)
	suite.NotEmpty(transport1.DtlsParameters().Fingerprints)
	suite.EqualValues("new", transport1.DtlsState())
	suite.Zero(transport1.DtlsRemoteCert())
	suite.EqualValues("new", transport1.SctpState())

	data1, _ := transport1.Dump()

	suite.Equal(data1.Id, transport1.Id())
	suite.Empty(data1.ProducerIds)
	suite.Empty(data1.ConsumerIds)
	suite.Equal(data1.IceRole, transport1.IceRole())
	suite.Equal(data1.IceParameters, transport1.IceParameters())
	suite.Equal(data1.IceCandidates, transport1.IceCandidates())
	suite.Equal(data1.IceState, transport1.IceState())
	suite.Equal(data1.IceSelectedTuple, transport1.IceSelectedTuple())
	suite.Equal(data1.DtlsParameters, transport1.DtlsParameters())
	suite.Equal(data1.DtlsState, transport1.DtlsState())
	suite.Equal(data1.SctpParameters, transport1.SctpParameters())
	suite.Equal(data1.SctpState, transport1.SctpState())
	suite.NotNil(data1.RecvRtpHeaderExtensions)
	suite.NotNil(data1.RtpListener)

	transport1.Close()
	suite.True(transport1.Closed())

	_, err = router.CreateWebRtcTransport(WebRtcTransportOptions{
		ListenIps: []TransportListenIp{
			{Ip: "127.0.0.1"},
		},
	})
	suite.NoError(err)
}

func (suite *WebRtcTransportTestingSuite) TestCreateWebRtcTransport_TypeError() {
	_, err := suite.router.CreateWebRtcTransport(WebRtcTransportOptions{})
	suite.IsType(NewTypeError(""), err)

	_, err = suite.router.CreateWebRtcTransport(WebRtcTransportOptions{
		ListenIps: []TransportListenIp{
			{Ip: "123"},
		},
	})
	suite.IsType(NewTypeError(""), err)
}

func (suite *WebRtcTransportTestingSuite) TestCreateWebRtcTransport_NonBindableIpError() {
	router := suite.router
	_, err := router.CreateWebRtcTransport(WebRtcTransportOptions{
		ListenIps: []TransportListenIp{
			{Ip: "8.8.8.8"},
		},
	})
	suite.Error(err)
}

func (suite *WebRtcTransportTestingSuite) TestGetStats_Succeeds() {
	data, _ := suite.transport.GetStats()

	suite.Len(data, 1)
	suite.Equal("webrtc-transport", data[0].Type)
	suite.NotZero(data[0].Timestamp)
	suite.EqualValues("controlled", data[0].IceRole)
	suite.EqualValues("new", data[0].IceState)
	suite.EqualValues("new", data[0].DtlsState)
	suite.Empty(data[0].SctpState)
	suite.Zero(data[0].BytesReceived)
	suite.Zero(data[0].RecvBitrate)
	suite.Zero(data[0].BytesSent)
	suite.Zero(data[0].SendBitrate)
	suite.Zero(data[0].RtpRecvBitrate)
	suite.Zero(data[0].RtpBytesSent)
	suite.Zero(data[0].RtpSendBitrate)
	suite.Zero(data[0].RtxBytesReceived)
	suite.Zero(data[0].RtxRecvBitrate)
	suite.Zero(data[0].RtxBytesSent)
	suite.Zero(data[0].RtxSendBitrate)
	suite.Zero(data[0].ProbationBytesSent)
	suite.Zero(data[0].ProbationSendBitrate)
	suite.Empty(data[0].IceSelectedTuple)
	suite.Zero(data[0].MaxIncomingBitrate)
	suite.Zero(data[0].RecvBitrate)
	suite.Zero(data[0].SendBitrate)
}

func (suite *WebRtcTransportTestingSuite) TestConnect_Succeeds() {
	dtlsRemoteParameters := DtlsParameters{
		Fingerprints: []DtlsFingerprint{
			{
				Algorithm: "sha-256",
				Value:     "82:5A:68:3D:36:C3:0A:DE:AF:E7:32:43:D2:88:83:57:AC:2D:65:E5:80:C4:B6:FB:AF:1A:A0:21:9F:6D:0C:AD",
			},
		},
		Role: DtlsRole_Client,
	}

	transport := suite.transport

	err := transport.Connect(TransportConnectOptions{
		DtlsParameters: &dtlsRemoteParameters,
	})
	suite.NoError(err)

	err = transport.Connect(TransportConnectOptions{
		DtlsParameters: &dtlsRemoteParameters,
	})
	suite.Error(err)
	suite.EqualValues("server", transport.DtlsParameters().Role)
}

func (suite *WebRtcTransportTestingSuite) TestConnect_RejectsWithTypeError() {
	transport := suite.transport

	err := transport.Connect(TransportConnectOptions{})
	suite.IsType(NewTypeError(""), err)

	dtlsRemoteParameters := DtlsParameters{
		Fingerprints: []DtlsFingerprint{
			{
				Algorithm: "sha-256000",
				Value:     "82:5A:68:3D:36:C3:0A:DE:AF:E7:32:43:D2:88:83:57:AC:2D:65:E5:80:C4:B6:FB:AF:1A:A0:21:9F:6D:0C:AD",
			},
		},
		Role: "client",
	}

	err = transport.Connect(TransportConnectOptions{
		DtlsParameters: &dtlsRemoteParameters,
	})
	suite.IsType(NewTypeError(""), err)

	dtlsRemoteParameters = DtlsParameters{
		Fingerprints: []DtlsFingerprint{
			{
				Algorithm: "sha-256",
				Value:     "82:5A:68:3D:36:C3:0A:DE:AF:E7:32:43:D2:88:83:57:AC:2D:65:E5:80:C4:B6:FB:AF:1A:A0:21:9F:6D:0C:AD",
			},
		},
		Role: "chicken",
	}
	err = transport.Connect(TransportConnectOptions{
		DtlsParameters: &dtlsRemoteParameters,
	})
	suite.IsType(NewTypeError(""), err)

	dtlsRemoteParameters = DtlsParameters{
		Fingerprints: []DtlsFingerprint{},
		Role:         "chicken",
	}
	err = transport.Connect(TransportConnectOptions{
		DtlsParameters: &dtlsRemoteParameters,
	})
	suite.IsType(NewTypeError(""), err)

	err = transport.Connect(TransportConnectOptions{
		DtlsParameters: &dtlsRemoteParameters,
	})
	suite.IsType(NewTypeError(""), err)
	suite.EqualValues("auto", transport.DtlsParameters().Role)
}

func (suite *WebRtcTransportTestingSuite) TestSetMaxIncomingBitrate_Succeeds() {
	transport := suite.transport
	err := transport.SetMaxIncomingBitrate(100000)
	suite.NoError(err)
}

func (suite *WebRtcTransportTestingSuite) TestRestartIce_Succeeds() {
	transport := suite.transport
	previousIceUsernameFragment := transport.IceParameters().UsernameFragment
	previousIcePassword := transport.IceParameters().Password

	iceParameters, err := transport.RestartIce()

	suite.NoError(err)
	suite.NotEmpty(iceParameters.UsernameFragment)
	suite.NotEmpty(iceParameters.Password)
	suite.True(iceParameters.IceLite)
	suite.NotEmpty(transport.IceParameters().UsernameFragment)
	suite.NotEmpty(transport.IceParameters().Password)
	suite.NotEqual(transport.IceParameters().UsernameFragment, previousIceUsernameFragment)
	suite.NotEqual(transport.IceParameters().Password, previousIcePassword)
}

func (suite *WebRtcTransportTestingSuite) TestEnableTraceEvent_Succeeds() {
	transport := suite.transport

	transport.EnableTraceEvent("foo", "probation")
	data, _ := transport.Dump()
	suite.Equal("probation", data.TraceEventTypes)

	transport.EnableTraceEvent()
	data, _ = transport.Dump()
	suite.Zero(data.TraceEventTypes)

	transport.EnableTraceEvent("probation", "FOO", "bwe", "BAR")
	data, _ = transport.Dump()
	suite.Equal("probation,bwe", data.TraceEventTypes)

	transport.EnableTraceEvent()
	data, _ = transport.Dump()
	suite.Zero(data.TraceEventTypes)
}

func (suite *WebRtcTransportTestingSuite) TestEvents_Succeeds() {
	transport := suite.transport

	// Private API.
	channel := transport.channel
	onIceStateChange := NewMockFunc(suite.T())
	transport.On("icestatechange", onIceStateChange.Fn())

	data, _ := json.Marshal(H{"iceState": "completed"})
	channel.Emit(transport.Id(), "icestatechange", data)

	onIceStateChange.ExpectCalled()
	onIceStateChange.ExpectCalledWith("completed")

	onIceSelectedTuple := NewMockFunc(suite.T())
	iceSelectedTuple := TransportTuple{
		LocalIp:    "1.1.1.1",
		LocalPort:  1111,
		RemoteIp:   "2.2.2.2",
		RemotePort: 2222,
		Protocol:   "udp",
	}
	data, _ = json.Marshal(H{"iceSelectedTuple": iceSelectedTuple})
	transport.On("iceselectedtuplechange", onIceSelectedTuple.Fn())
	channel.Emit(transport.Id(), "iceselectedtuplechange", data)

	onIceSelectedTuple.ExpectCalled()
	onIceSelectedTuple.ExpectCalledWith(iceSelectedTuple)

	onDtlsStateChange := NewMockFunc(suite.T())
	onDtlsStateChangeFn := onDtlsStateChange.Fn()
	transport.On("dtlsstatechange", onDtlsStateChangeFn)
	data, _ = json.Marshal(H{"dtlsState": "connecting"})
	channel.Emit(transport.Id(), "dtlsstatechange", data)

	onDtlsStateChange.ExpectCalledTimes(1)
	onDtlsStateChange.ExpectCalledWith("connecting")
	suite.EqualValues("connecting", transport.DtlsState())

	onDtlsStateChange.Reset()

	data, _ = json.Marshal(H{"dtlsState": "connected", "dtlsRemoteCert": "ABCD"})
	channel.Emit(transport.Id(), "dtlsstatechange", data)

	onDtlsStateChange.ExpectCalledTimes(1)
	onDtlsStateChange.ExpectCalledWith("connected")
	suite.EqualValues("connected", transport.DtlsState())
	suite.Equal("ABCD", transport.DtlsRemoteCert())
}

func (suite *WebRtcTransportTestingSuite) TestMethodsRejectIfClosed() {
	transport := suite.transport
	onObserverClose := NewMockFunc(suite.T())

	transport.Observer().Once("close", onObserverClose.Fn())
	transport.Close()

	onObserverClose.ExpectCalled()
	suite.True(transport.Closed())
	suite.EqualValues("closed", transport.IceState())
	suite.Nil(transport.IceSelectedTuple())
	suite.EqualValues("closed", transport.DtlsState())
	suite.Zero(transport.SctpState())

	_, err := transport.Dump()
	suite.Error(err)

	_, err = transport.GetStats()
	suite.Error(err)

	err = transport.Connect(TransportConnectOptions{})
	suite.Error(err)

	err = transport.SetMaxIncomingBitrate(100)
	suite.Error(err)

	_, err = transport.RestartIce()
	suite.Error(err)
}

func (suite *WebRtcTransportTestingSuite) TestEmitsRoutercloseIfRouterIsClosed() {
	router := suite.router
	transport, _ := router.CreateWebRtcTransport(WebRtcTransportOptions{
		ListenIps: []TransportListenIp{
			{Ip: "127.0.0.1"},
		},
		EnableSctp: true,
	})
	onObserverClose := NewMockFunc(suite.T())
	transport.Observer().Once("close", onObserverClose.Fn())
	router.Close()
	onObserverClose.ExpectCalled()
	suite.True(transport.Closed())
	suite.EqualValues("closed", transport.IceState())
	suite.Nil(transport.IceSelectedTuple())
	suite.EqualValues("closed", transport.DtlsState())
	suite.EqualValues("closed", transport.SctpState())
}

func (suite *WebRtcTransportTestingSuite) TestEmitsRoutercloseIfWorkerIsClosed() {
	worker := CreateTestWorker()
	router := CreateRouter(worker)
	transport, _ := router.CreateWebRtcTransport(WebRtcTransportOptions{
		ListenIps: []TransportListenIp{
			{Ip: "127.0.0.1"},
		},
	})
	onObserverClose := NewMockFunc(suite.T())
	transport.Observer().Once("close", onObserverClose.Fn())
	worker.Close()
	onObserverClose.ExpectCalled()
	suite.True(transport.Closed())
	suite.EqualValues("closed", transport.IceState())
	suite.Nil(transport.IceSelectedTuple())
	suite.EqualValues("closed", transport.DtlsState())
	suite.Zero(transport.SctpState())
}

/*


 */
