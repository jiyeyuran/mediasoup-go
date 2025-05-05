package mediasoup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebRtcServerClose(t *testing.T) {
	worker := newTestWorker()
	server, _ := worker.CreateWebRtcServer(&WebRtcServerOptions{
		ListenInfos: []*TransportListenInfo{
			{Protocol: TransportProtocolUDP, Ip: "127.0.0.1", Port: pickUdpPort()},
		},
	})
	err := server.Close()
	assert.NoError(t, err)
	dump, _ := worker.Dump()
	assert.Empty(t, dump.WebRtcServerIds)
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
