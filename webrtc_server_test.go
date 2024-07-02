package mediasoup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestWebRtcServerTestingSuite(t *testing.T) {
	suite.Run(t, new(WebRtcServerTestingSuite))
}

type WebRtcServerTestingSuite struct {
	TestingSuite
}

func TestWorkerCreateWebRtcServer(t *testing.T) {
	t.Run("succeeds", func(t *testing.T) {
		worker := CreateTestWorker()
		port1 := GetFreeUdpPort()
		port2 := GetFreeTcpPort()

		webRtcServer, err := worker.CreateWebRtcServer(WebRtcServerOptions{
			ListenInfos: []WebRtcServerListenInfo{
				{Protocol: TransportProtocol_Udp, Ip: "127.0.0.1", Port: port1},
				{Protocol: TransportProtocol_Tcp, Ip: "127.0.0.1", Port: port2},
			},
			AppData: H{"foo": 123},
		})
		require.NoError(t, err)
		assert.Equal(t, H{"foo": 123}, webRtcServer.AppData())

		dump, err := worker.Dump()
		require.NoError(t, err)
		assert.Equal(t, WorkerDump{
			Pid:             worker.Pid(),
			WebRtcServerIds: []string{webRtcServer.Id()},
			RouterIds:       []string{},
		}, dump)

		webRtcServerDump, _ := webRtcServer.Dump()
		assert.Equal(t, WebRtcServerDump{
			Id:                        webRtcServer.Id(),
			UdpSockets:                []NetAddr{{Ip: "127.0.0.1", Port: port1}},
			TcpServers:                []NetAddr{{Ip: "127.0.0.1", Port: port2}},
			WebRtcTransportIds:        []string{},
			LocalIceUsernameFragments: []LocalIceUsernameFragment{},
			TupleHashes:               []TupleHash{},
		}.String(), webRtcServerDump.String())

		worker.Close()
		assert.True(t, webRtcServer.Closed())
		assert.Len(t, worker.webRtcServersForTesting(), 0)
	})

	t.Run("worker.createWebRtcServer() with unavailable listenInfos rejects with Error", func(t *testing.T) {
		worker1 := CreateTestWorker()
		worker2 := CreateTestWorker()
		defer worker1.Close()
		defer worker2.Close()

		port1 := GetFreeUdpPort()
		port2 := GetFreeUdpPort()

		// Using an unavailable listen IP.
		_, err := worker1.CreateWebRtcServer(WebRtcServerOptions{
			ListenInfos: []WebRtcServerListenInfo{
				{Protocol: TransportProtocol_Udp, Ip: "127.0.0.1", Port: port1},
				{Protocol: TransportProtocol_Udp, Ip: "1.2.3.4", Port: port2},
			},
		})
		require.Error(t, err)

		// Using the same UDP port in two listenInfos.
		_, err = worker1.CreateWebRtcServer(WebRtcServerOptions{
			ListenInfos: []WebRtcServerListenInfo{
				{Protocol: TransportProtocol_Udp, Ip: "127.0.0.1", Port: port1},
				{Protocol: TransportProtocol_Udp, Ip: "127.0.0.1", AnnouncedIp: "1.2.3.4", Port: port1},
			},
		})
		require.Error(t, err)

		_, err = worker1.CreateWebRtcServer(WebRtcServerOptions{
			ListenInfos: []WebRtcServerListenInfo{
				{Protocol: TransportProtocol_Udp, Ip: "127.0.0.1", Port: port1},
			},
		})
		require.NoError(t, err)

		// Using the same UDP port in a second Worker.
		_, err = worker2.CreateWebRtcServer(WebRtcServerOptions{
			ListenInfos: []WebRtcServerListenInfo{
				{Protocol: TransportProtocol_Udp, Ip: "127.0.0.1", Port: port1},
			},
		})
		require.Error(t, err)
	})

	t.Run("worker.createWebRtcServer() rejects with InvalidStateError if Worker is closed", func(t *testing.T) {
		worker := CreateTestWorker()
		worker.Close()
		port := GetFreeUdpPort()
		_, err := worker.CreateWebRtcServer(WebRtcServerOptions{
			ListenInfos: []WebRtcServerListenInfo{
				{Protocol: TransportProtocol_Udp, Ip: "127.0.0.1", Port: port},
			},
		})
		assert.IsType(t, InvalidStateError{}, err)
	})
}

func TestWebRtcServerCloseSucceeds(t *testing.T) {
	worker := CreateTestWorker()
	port := GetFreeUdpPort()
	webRtcServer, _ := worker.CreateWebRtcServer(WebRtcServerOptions{
		ListenInfos: []WebRtcServerListenInfo{
			{Protocol: TransportProtocol_Udp, Ip: "127.0.0.1", Port: port},
		},
	})
	onObserverClose := NewMockFunc(t)
	webRtcServer.Observer().Once("close", onObserverClose.Fn())
	webRtcServer.Close()
	worker.Close()

	onObserverClose.ExpectCalledTimes(1)
	assert.True(t, webRtcServer.Closed())
}

func TestWebRtcServerEmits(t *testing.T) {
	t.Run(`WebRtcServer emits "workerclose" if Worker is closed`, func(t *testing.T) {
		worker := CreateTestWorker()
		port := GetFreeUdpPort()
		webRtcServer, _ := worker.CreateWebRtcServer(WebRtcServerOptions{
			ListenInfos: []WebRtcServerListenInfo{
				{Protocol: TransportProtocol_Tcp, Ip: "127.0.0.1", Port: port},
			},
		})
		onObserverClose := NewMockFunc(t)
		webRtcServer.Observer().Once("close", onObserverClose.Fn())

		onObserverWorkerClose := NewMockFunc(t)
		webRtcServer.On("workerclose", onObserverWorkerClose.Fn())
		worker.Close()

		onObserverWorkerClose.ExpectCalledTimes(1)
		onObserverClose.ExpectCalledTimes(1)
		assert.True(t, webRtcServer.Closed())
	})
}

func TestRouterCreateWebRtcTransportWithWebRtcServer(t *testing.T) {
	t.Run(`router.createWebRtcTransport() with webRtcServer succeeds and transport is closed`, func(t *testing.T) {
		worker := CreateTestWorker()
		port1 := GetFreeUdpPort()
		port2 := GetFreeTcpPort()
		webRtcServer, _ := worker.CreateWebRtcServer(WebRtcServerOptions{
			ListenInfos: []WebRtcServerListenInfo{
				{Protocol: TransportProtocol_Udp, Ip: "127.0.0.1", Port: port1},
				{Protocol: TransportProtocol_Tcp, Ip: "127.0.0.1", Port: port2},
			},
		})
		onObserverWebRtcTransportHandled := NewMockFunc(t)
		onObserverWebRtcTransportUnhandled := NewMockFunc(t)
		webRtcServer.Observer().Once("webrtctransporthandled", onObserverWebRtcTransportHandled.Fn())
		webRtcServer.Observer().Once("webrtctransportunhandled", onObserverWebRtcTransportUnhandled.Fn())

		router, _ := worker.CreateRouter(RouterOptions{})
		onObserverNewTransport := NewMockFunc(t)
		router.Observer().Once("newtransport", onObserverNewTransport.Fn())

		transport, _ := router.CreateWebRtcTransport(WebRtcTransportOptions{
			WebRtcServer: webRtcServer,
			EnableTcp:    false,
			AppData:      H{"foo": "bar"},
		})
		routerDump, _ := router.Dump()
		assert.Equal(t, []string{transport.Id()}, routerDump.TransportIds)

		onObserverWebRtcTransportHandled.ExpectCalledTimes(1)
		onObserverWebRtcTransportHandled.ExpectCalledWith(transport)
		onObserverNewTransport.ExpectCalledTimes(1)
		onObserverNewTransport.ExpectCalledWith(transport)
		assert.False(t, transport.Closed())
		assert.Equal(t, H{"foo": "bar"}, transport.AppData())

		iceCandidates := transport.IceCandidates()
		require.Len(t, iceCandidates, 1)
		assert.Equal(t, "127.0.0.1", iceCandidates[0].Ip)
		assert.Equal(t, port1, iceCandidates[0].Port)
		assert.Equal(t, TransportProtocol_Udp, iceCandidates[0].Protocol)
		assert.Equal(t, "host", iceCandidates[0].Type)
		assert.Empty(t, iceCandidates[0].TcpType)

		assert.EqualValues(t, "new", transport.IceState())
		assert.Nil(t, transport.IceSelectedTuple())

		assert.Len(t, webRtcServer.webRtcTransportsForTesting(), 1)
		assert.Len(t, router.transportsForTesting(), 1)

		webRtcServerDump, _ := webRtcServer.Dump()
		localIceUsernameFragment := webRtcServerDump.LocalIceUsernameFragments[0].LocalIceUsernameFragment

		assert.JSONEq(t, WebRtcServerDump{
			Id:                 webRtcServer.Id(),
			UdpSockets:         []NetAddr{{Ip: "127.0.0.1", Port: port1}},
			TcpServers:         []NetAddr{{Ip: "127.0.0.1", Port: port2}},
			WebRtcTransportIds: []string{transport.Id()},
			LocalIceUsernameFragments: []LocalIceUsernameFragment{
				{LocalIceUsernameFragment: localIceUsernameFragment, WebRtcTransportId: transport.Id()},
			},
			TupleHashes: []TupleHash{},
		}.String(), webRtcServerDump.String())

		transport.Close()

		assert.True(t, transport.Closed())
		assert.Equal(t, 1, onObserverWebRtcTransportUnhandled.CalledTimes())
		onObserverWebRtcTransportUnhandled.ExpectCalledWith(transport)
		assert.Len(t, webRtcServer.webRtcTransportsForTesting(), 0)
		assert.Len(t, router.transportsForTesting(), 0)

		webRtcServerDump, _ = webRtcServer.Dump()
		assert.JSONEq(t, WebRtcServerDump{
			Id:                        webRtcServer.Id(),
			UdpSockets:                []NetAddr{{Ip: "127.0.0.1", Port: port1}},
			TcpServers:                []NetAddr{{Ip: "127.0.0.1", Port: port2}},
			WebRtcTransportIds:        []string{},
			LocalIceUsernameFragments: []LocalIceUsernameFragment{},
			TupleHashes:               []TupleHash{},
		}.String(), webRtcServerDump.String())

		worker.Close()
	})

	t.Run(`router.createWebRtcTransport() with webRtcServer succeeds and webRtcServer is closed`, func(t *testing.T) {
		worker := CreateTestWorker()
		port1 := GetFreeUdpPort()
		port2 := GetFreeTcpPort()
		webRtcServer, _ := worker.CreateWebRtcServer(WebRtcServerOptions{
			ListenInfos: []WebRtcServerListenInfo{
				{Protocol: TransportProtocol_Udp, Ip: "127.0.0.1", Port: port1},
				{Protocol: TransportProtocol_Tcp, Ip: "127.0.0.1", Port: port2},
			},
		})
		onObserverWebRtcTransportHandled := NewMockFunc(t)
		onObserverWebRtcTransportUnhandled := NewMockFunc(t)
		webRtcServer.Observer().Once("webrtctransporthandled", onObserverWebRtcTransportHandled.Fn())
		webRtcServer.Observer().Once("webrtctransportunhandled", onObserverWebRtcTransportUnhandled.Fn())

		router, _ := worker.CreateRouter(RouterOptions{})
		onObserverNewTransport := NewMockFunc(t)
		router.Observer().Once("newtransport", onObserverNewTransport.Fn())

		transport, _ := router.CreateWebRtcTransport(WebRtcTransportOptions{
			WebRtcServer: webRtcServer,
			EnableTcp:    true,
			EnableUdp:    Bool(false),
			AppData:      H{"foo": "bar"},
		})
		routerDump, _ := router.Dump()
		assert.Equal(t, []string{transport.Id()}, routerDump.TransportIds)

		onObserverWebRtcTransportHandled.ExpectCalledTimes(1)
		onObserverWebRtcTransportHandled.ExpectCalledWith(transport)
		onObserverNewTransport.ExpectCalledTimes(1)
		onObserverNewTransport.ExpectCalledWith(transport)
		assert.False(t, transport.Closed())
		assert.Equal(t, H{"foo": "bar"}, transport.AppData())

		iceCandidates := transport.IceCandidates()
		require.Len(t, iceCandidates, 1)
		assert.Equal(t, "127.0.0.1", iceCandidates[0].Ip)
		assert.Equal(t, port2, iceCandidates[0].Port)
		assert.Equal(t, TransportProtocol_Tcp, iceCandidates[0].Protocol)
		assert.Equal(t, "host", iceCandidates[0].Type)
		assert.Equal(t, "passive", iceCandidates[0].TcpType)

		assert.EqualValues(t, "new", transport.IceState())
		assert.Nil(t, transport.IceSelectedTuple())

		assert.Len(t, webRtcServer.webRtcTransportsForTesting(), 1)
		assert.Len(t, router.transportsForTesting(), 1)

		webRtcServerDump, _ := webRtcServer.Dump()
		localIceUsernameFragment := webRtcServerDump.LocalIceUsernameFragments[0].LocalIceUsernameFragment

		assert.JSONEq(t, WebRtcServerDump{
			Id:                 webRtcServer.Id(),
			UdpSockets:         []NetAddr{{Ip: "127.0.0.1", Port: port1}},
			TcpServers:         []NetAddr{{Ip: "127.0.0.1", Port: port2}},
			WebRtcTransportIds: []string{transport.Id()},
			LocalIceUsernameFragments: []LocalIceUsernameFragment{
				{LocalIceUsernameFragment: localIceUsernameFragment, WebRtcTransportId: transport.Id()},
			},
			TupleHashes: []TupleHash{},
		}.String(), webRtcServerDump.String())

		// Let's restart ICE in the transport so it should add a new entry in
		// localIceUsernameFragments in the WebRtcServer.
		transport.RestartIce()

		webRtcServerDump, _ = webRtcServer.Dump()
		localIceUsernameFragment1 := webRtcServerDump.LocalIceUsernameFragments[0].LocalIceUsernameFragment
		localIceUsernameFragment2 := webRtcServerDump.LocalIceUsernameFragments[1].LocalIceUsernameFragment

		assert.JSONEq(t, WebRtcServerDump{
			Id:                 webRtcServer.Id(),
			UdpSockets:         []NetAddr{{Ip: "127.0.0.1", Port: port1}},
			TcpServers:         []NetAddr{{Ip: "127.0.0.1", Port: port2}},
			WebRtcTransportIds: []string{transport.Id()},
			LocalIceUsernameFragments: []LocalIceUsernameFragment{
				{LocalIceUsernameFragment: localIceUsernameFragment1, WebRtcTransportId: transport.Id()},
				{LocalIceUsernameFragment: localIceUsernameFragment2, WebRtcTransportId: transport.Id()},
			},
			TupleHashes: []TupleHash{},
		}.String(), webRtcServerDump.String())

		onObserverClose := NewMockFunc(t)
		webRtcServer.Observer().Once("close", onObserverClose.Fn())

		onListenServerClose := NewMockFunc(t)
		transport.Once("listenserverclose", onListenServerClose.Fn())

		webRtcServer.Close()

		assert.True(t, webRtcServer.Closed())
		assert.Equal(t, 1, onObserverClose.CalledTimes())
		assert.Equal(t, 1, onListenServerClose.CalledTimes())
		assert.Equal(t, 1, onObserverWebRtcTransportUnhandled.CalledTimes())
		onObserverWebRtcTransportUnhandled.ExpectCalledWith(transport)
		assert.True(t, transport.Closed())
		assert.Len(t, webRtcServer.webRtcTransportsForTesting(), 0)
		assert.Len(t, router.transportsForTesting(), 0)

		workerDump, _ := worker.Dump()
		assert.JSONEq(t, WorkerDump{
			Pid:             worker.Pid(),
			WebRtcServerIds: []string{},
			RouterIds:       []string{router.Id()},
		}.String(), workerDump.String())

		routerDump, _ = router.Dump()
		assert.JSONEq(t, RouterDump{
			Id:           router.Id(),
			TransportIds: []string{},
		}.String(), routerDump.String())

		worker.Close()
	})
}
