package mediasoup

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestWebRtcServerClose(t *testing.T) {
	worker := newTestWorker()
	server, _ := worker.CreateWebRtcServer(&WebRtcServerOptions{
		ListenInfos: []*TransportListenInfo{
			{Protocol: TransportProtocolUDP, Ip: "127.0.0.1", Port: pickUdpPort()},
		},
	})

	closed := make(chan struct{})
	server.OnClose(func(ctx context.Context) { close(closed) })

	assert.False(t, server.Closed())
	require.NoError(t, server.Close())
	assert.True(t, server.Closed())

	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for the close event")
	}

	dump, _ := worker.Dump()
	assert.Empty(t, dump.WebRtcServerIds)

	// The worker tracks its servers to close them on its own shutdown and drops
	// them from the close event, so a closed server must not be retained.
	_, tracked := worker.webRtcServers.Load(server.Id())
	assert.False(t, tracked)

	// Closing twice is a no-op rather than an error.
	assert.NoError(t, server.Close())
}

// Closing a WebRtcServer must close the transports it was handling traffic for.
func TestWebRtcServerCloseClosesItsTransports(t *testing.T) {
	worker := newTestWorker()
	server, err := worker.CreateWebRtcServer(&WebRtcServerOptions{
		ListenInfos: []*TransportListenInfo{
			{Protocol: TransportProtocolUDP, Ip: "127.0.0.1", Port: 0},
		},
	})
	require.NoError(t, err)

	router, err := worker.CreateRouter(&RouterOptions{})
	require.NoError(t, err)

	transport, err := router.CreateWebRtcTransport(&WebRtcTransportOptions{
		WebRtcServer: server,
	})
	require.NoError(t, err)

	transportClosed := make(chan struct{})
	transport.OnClose(func(ctx context.Context) { close(transportClosed) })

	require.NoError(t, server.Close())

	select {
	case <-transportClosed:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for the transport close event")
	}

	assert.True(t, transport.Closed())
}

// A WebRtcServer belongs to a Worker, so it goes down with it. Its transports are
// left to their routers, which the worker closes too.
func TestWebRtcServerWorkerClosed(t *testing.T) {
	worker := newTestWorker()
	server, err := worker.CreateWebRtcServer(&WebRtcServerOptions{
		ListenInfos: []*TransportListenInfo{
			{Protocol: TransportProtocolUDP, Ip: "127.0.0.1", Port: 0},
		},
	})
	require.NoError(t, err)

	router, err := worker.CreateRouter(&RouterOptions{})
	require.NoError(t, err)

	transport, err := router.CreateWebRtcTransport(&WebRtcTransportOptions{
		WebRtcServer: server,
	})
	require.NoError(t, err)

	serverClosed := make(chan struct{})
	server.OnClose(func(ctx context.Context) { close(serverClosed) })

	worker.Close()

	select {
	case <-serverClosed:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for the server close event")
	}

	assert.True(t, server.Closed())
	assert.True(t, router.Closed())
	assert.True(t, transport.Closed())
}

func TestWebRtcServerDump(t *testing.T) {
	worker := newTestWorker()
	server, _ := worker.CreateWebRtcServer(&WebRtcServerOptions{
		ListenInfos: []*TransportListenInfo{
			{Protocol: TransportProtocolUDP, Ip: "127.0.0.1", Port: pickTcpPort()},
		},
	})
	dump, err := server.Dump()
	require.NoError(t, err)
	assert.Equal(t, server.Id(), dump.Id)
}

func TestCreateWebRtcTransportWithWebRtcServer(t *testing.T) {
	mymock := new(MockedHandler)
	defer mymock.AssertExpectations(t)

	mymock.On("OnNewTransport", mock.IsType(context.Background()), mock.IsType(&Transport{})).Once()

	worker := newTestWorker()
	router, _ := worker.CreateRouter(&RouterOptions{})
	router.OnNewTransport(mymock.OnNewTransport)

	// Create WebRtcServer
	webRtcServer, err := worker.CreateWebRtcServer(&WebRtcServerOptions{
		ListenInfos: []*TransportListenInfo{
			{Protocol: TransportProtocolUDP, Ip: "127.0.0.1", Port: 0},
			{Protocol: TransportProtocolTCP, Ip: "127.0.0.1", Port: 0, AnnouncedAddress: "media1.foo.org", ExposeInternalIp: true},
		},
	})
	require.NoError(t, err)
	defer webRtcServer.Close()

	// Create transport with WebRtcServer
	transport, err := router.CreateWebRtcTransport(&WebRtcTransportOptions{
		WebRtcServer: webRtcServer,
		EnableUdp:    ref(false),
		EnableTcp:    true,
		AppData:      map[string]any{"foo": "bar"},
	})
	require.NoError(t, err)
	defer transport.Close()

	// Verify transport
	assert.False(t, transport.Closed())
	assert.Equal(t, map[string]any{"foo": "bar"}, transport.AppData())

	// Verify ICE candidates
	iceCandidates := transport.Data().IceCandidates
	assert.Len(t, iceCandidates, 2)
	assert.Equal(t, "media1.foo.org", iceCandidates[0].Address)
	assert.Equal(t, TransportProtocolTCP, iceCandidates[0].Protocol)
	assert.Equal(t, "passive", iceCandidates[0].TcpType)
	assert.Equal(t, "127.0.0.1", iceCandidates[1].Address)
	assert.Equal(t, TransportProtocolTCP, iceCandidates[1].Protocol)
	assert.Equal(t, "passive", iceCandidates[1].TcpType)
	assert.True(t, iceCandidates[0].Priority > iceCandidates[1].Priority)

	// Verify dump
	dump, err := router.Dump()
	require.NoError(t, err)
	assert.Contains(t, dump.TransportIds, transport.Id())

	// Close transport and verify
	assert.NoError(t, transport.Close())
	assert.True(t, transport.Closed())

	// Verify WebRtcServer dump after close
	serverDump, err := webRtcServer.Dump()
	require.NoError(t, err)
	assert.NotContains(t, serverDump.WebRtcTransportIds, transport.Id())
	assert.Len(t, serverDump.UdpSockets, 1)
	assert.Len(t, serverDump.TcpServers, 1)
	assert.NotZero(t, serverDump.UdpSockets[0].Port)
	assert.NotZero(t, serverDump.TcpServers[0].Port)
}
