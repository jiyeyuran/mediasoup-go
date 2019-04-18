package mediasoup

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/jiyeyuran/mediasoup-go/mediasoup/h264profile"
	"github.com/stretchr/testify/assert"
)

var testWebRtcMediaCodecs = []RtpCodecCapability{
	{
		Kind:      "audio",
		MimeType:  "audio/opus",
		ClockRate: 48000,
		Channels:  2,
		Parameters: &RtpCodecParameter{
			Useinbandfec: 1,
		},
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
		Parameters: &RtpCodecParameter{
			RtpH264Parameter: h264profile.RtpH264Parameter{
				LevelAsymmetryAllowed: 1,
				PacketizationMode:     1,
				ProfileLevelId:        "4d0032",
			},
		},
	},
}

func setupWebRtcTest(t *testing.T) (router *Router, transport *WebRtcTransport) {
	router, err := worker.CreateRouter(testWebRtcMediaCodecs)
	assert.NoError(t, err)

	transport, err = router.CreateWebRtcTransport(CreateWebRtcTransportParams{
		ListenIps: []ListenIp{
			{Ip: "127.0.0.1", AnnouncedIp: "9.9.9.1"},
		},
	})
	assert.NoError(t, err)

	return
}

func TestRouterCreateWebRtcTransport_Succeeds(t *testing.T) {
	router, transport := setupWebRtcTest(t)

	var dump struct {
		TransportIds []string
	}
	router.Dump().Unmarshal(&dump)

	assert.Equal(t, dump.TransportIds, []string{transport.Id()})

	var tt Transport
	called := 0
	router.Observer().Once("newtransport", func(transport Transport) {
		tt = transport
		called++
	})

	transport1, err := router.CreateWebRtcTransport(CreateWebRtcTransportParams{
		ListenIps: []ListenIp{
			{Ip: "127.0.0.1", AnnouncedIp: "9.9.9.1"},
			{Ip: "127.0.0.1", AnnouncedIp: "9.9.9.2"},
			{Ip: "127.0.0.1"},
		},
		EnableTcp: true,
		PreferUdp: true,
		AppData:   H{"foo": "bar"},
	})
	assert.NoError(t, err)

	assert.Equal(t, called, 1)
	assert.Equal(t, tt, transport1)
	assert.NotEmpty(t, transport1.Id())
	assert.False(t, transport1.Closed())
	assert.Equal(t, transport1.AppData(), H{"foo": "bar"})
	assert.Equal(t, transport1.IceRole(), "controlled")
	assert.NotEmpty(t, transport1.IceParameters())
	assert.True(t, transport1.IceParameters().IceLite)
	assert.NotEmpty(t, transport1.IceParameters().UsernameFragment)
	assert.NotEmpty(t, transport1.IceParameters().Password)
	assert.NotEmpty(t, transport1.IceCandidates())
	assert.Len(t, transport1.IceCandidates(), 6)

	iceCandidates := transport1.IceCandidates()

	assert.Equal(t, iceCandidates[0].Ip, "9.9.9.1")
	assert.Equal(t, iceCandidates[0].Protocol, "udp")
	assert.Equal(t, iceCandidates[0].Type, "host")
	assert.Empty(t, iceCandidates[0].TcpType)
	assert.Equal(t, iceCandidates[1].Ip, "9.9.9.1")
	assert.Equal(t, iceCandidates[1].Protocol, "tcp")
	assert.Equal(t, iceCandidates[1].Type, "host")
	assert.Equal(t, iceCandidates[1].TcpType, "passive")
	assert.Equal(t, iceCandidates[2].Ip, "9.9.9.2")
	assert.Equal(t, iceCandidates[2].Protocol, "udp")
	assert.Equal(t, iceCandidates[2].Type, "host")
	assert.Empty(t, iceCandidates[2].TcpType)
	assert.Equal(t, iceCandidates[3].Ip, "9.9.9.2")
	assert.Equal(t, iceCandidates[3].Protocol, "tcp")
	assert.Equal(t, iceCandidates[3].Type, "host")
	assert.Equal(t, iceCandidates[3].TcpType, "passive")
	assert.Equal(t, iceCandidates[4].Ip, "127.0.0.1")
	assert.Equal(t, iceCandidates[4].Protocol, "udp")
	assert.Equal(t, iceCandidates[4].Type, "host")
	assert.Empty(t, iceCandidates[4].TcpType)
	assert.Equal(t, iceCandidates[5].Ip, "127.0.0.1")
	assert.Equal(t, iceCandidates[5].Protocol, "tcp")
	assert.Equal(t, iceCandidates[5].Type, "host")
	assert.Equal(t, iceCandidates[5].TcpType, "passive")
	assert.Greater(t, iceCandidates[0].Priority, iceCandidates[1].Priority)
	assert.Greater(t, iceCandidates[2].Priority, iceCandidates[1].Priority)
	assert.Greater(t, iceCandidates[2].Priority, iceCandidates[3].Priority)
	assert.Greater(t, iceCandidates[4].Priority, iceCandidates[3].Priority)
	assert.Greater(t, iceCandidates[4].Priority, iceCandidates[5].Priority)
	assert.Equal(t, transport.IceState(), "new")
	assert.Empty(t, transport.IceSelectedTuple())
	assert.NotEmpty(t, transport.DtlsParameters())
	assert.NotEmpty(t, transport.DtlsParameters().Fingerprints)
	assert.Equal(t, transport.DtlsState(), "new")
	assert.Empty(t, transport.DtlsRemoteCert())

	var data1 struct {
		Id                  string
		ProducerIds         []string
		ConsumerIds         []string
		IceRole             string
		IceParameters       IceParameters
		IceCandidates       []IceCandidate
		IceState            string
		IceSelectedTuple    *TransportTuple
		DtlsParameters      DtlsParameters
		DtlsState           string
		RtpHeaderExtensions interface{}
		RtpListener         interface{}
	}

	transport1.Dump().Unmarshal(&data1)

	assert.Equal(t, data1.Id, transport1.Id())
	assert.Empty(t, data1.ProducerIds)
	assert.Empty(t, data1.ConsumerIds)
	assert.Equal(t, data1.IceRole, transport1.IceRole())
	assert.Equal(t, data1.IceParameters, transport1.IceParameters())
	assert.Equal(t, data1.IceCandidates, transport1.IceCandidates())
	assert.Equal(t, data1.IceState, transport1.IceState())
	assert.Equal(t, data1.IceSelectedTuple, transport1.IceSelectedTuple())
	assert.Equal(t, data1.DtlsParameters, transport1.DtlsParameters())
	assert.Equal(t, data1.DtlsState, transport1.DtlsState())
	assert.NotNil(t, data1.RtpHeaderExtensions)
	assert.NotNil(t, data1.RtpListener)

	transport1.Close()
	assert.True(t, transport1.Closed())

	_, err = router.CreateWebRtcTransport(CreateWebRtcTransportParams{
		ListenIps: []ListenIp{
			{Ip: "127.0.0.1"},
		},
	})
	assert.NoError(t, err)
}

func TestRouterCreateWebRtcTransport_TypeError(t *testing.T) {
	router, _ := setupWebRtcTest(t)

	_, err := router.CreateWebRtcTransport(CreateWebRtcTransportParams{})
	assert.IsType(t, err, NewTypeError(""))

	_, err = router.CreateWebRtcTransport(CreateWebRtcTransportParams{
		ListenIps: []ListenIp{
			{Ip: "127.0.0.1"},
		},
		AppData: "NOT-AN-OBJECT",
	})
	assert.IsType(t, err, NewTypeError(""))
}

func TestRouterCreateWebRtcTransport_WithNonBindableIPRejectsWithError(t *testing.T) {
	router, _ := setupWebRtcTest(t)

	_, err := router.CreateWebRtcTransport(CreateWebRtcTransportParams{
		ListenIps: []ListenIp{
			{Ip: "8.8.8.8"},
		},
	})

	assert.Error(t, err)
}

func TestWebRtcTransportGetStats_Succeeds(t *testing.T) {
	_, transport := setupWebRtcTest(t)

	data, _ := transport.GetStats()

	assert.Len(t, data, 1)
	assert.Equal(t, data[0].Type, "transport")
	assert.NotEmpty(t, data[0].TransportId)
	assert.NotEmpty(t, data[0].Timestamp)
	assert.Equal(t, data[0].IceRole, "controlled")
	assert.Equal(t, data[0].IceState, "new")
	assert.Equal(t, data[0].DtlsState, "new")
	assert.Empty(t, data[0].BytesReceived)
	assert.Empty(t, data[0].BytesSent)
	assert.Empty(t, data[0].IceSelectedTuple)
	assert.Empty(t, data[0].AvailableIncomingBitrate)
	assert.Empty(t, data[0].AvailableOutgoingBitrate)
	assert.Empty(t, data[0].MaxIncomingBitrate)
}

func TestWebRtcTransportConnect_Succeeds(t *testing.T) {
	_, transport := setupWebRtcTest(t)

	dtlsRemoteParameters := DtlsParameters{
		Fingerprints: []DtlsFingerprint{
			{
				Algorithm: "sha-256",
				Value:     "82:5A:68:3D:36:C3:0A:DE:AF:E7:32:43:D2:88:83:57:AC:2D:65:E5:80:C4:B6:FB:AF:1A:A0:21:9F:6D:0C:AD",
			},
		},
		Role: "client",
	}

	err := transport.Connect(transportConnectParams{
		DtlsParameters: &dtlsRemoteParameters,
	})
	assert.NoError(t, err)

	err = transport.Connect(transportConnectParams{
		DtlsParameters: &dtlsRemoteParameters,
	})
	assert.Error(t, err)

	assert.Equal(t, transport.DtlsParameters().Role, "server")
}

func TestWebRtcTransportConnect_TypeError(t *testing.T) {
	_, transport := setupWebRtcTest(t)

	err := transport.Connect(transportConnectParams{})
	assert.IsType(t, err, NewTypeError(""))

	dtlsRemoteParameters := DtlsParameters{
		Fingerprints: []DtlsFingerprint{
			{
				Algorithm: "sha-256000",
				Value:     "82:5A:68:3D:36:C3:0A:DE:AF:E7:32:43:D2:88:83:57:AC:2D:65:E5:80:C4:B6:FB:AF:1A:A0:21:9F:6D:0C:AD",
			},
		},
		Role: "client",
	}

	err = transport.Connect(transportConnectParams{
		DtlsParameters: &dtlsRemoteParameters,
	})
	assert.IsType(t, err, NewTypeError(""))

	dtlsRemoteParameters = DtlsParameters{
		Fingerprints: []DtlsFingerprint{
			{
				Algorithm: "sha-256",
				Value:     "82:5A:68:3D:36:C3:0A:DE:AF:E7:32:43:D2:88:83:57:AC:2D:65:E5:80:C4:B6:FB:AF:1A:A0:21:9F:6D:0C:AD",
			},
		},
		Role: "chicken",
	}

	err = transport.Connect(transportConnectParams{
		DtlsParameters: &dtlsRemoteParameters,
	})
	assert.IsType(t, err, NewTypeError(""))

	err = transport.Connect(transportConnectParams{
		DtlsParameters: &dtlsRemoteParameters,
	})
	assert.IsType(t, err, NewTypeError(""))

	assert.Equal(t, transport.DtlsParameters().Role, "auto")
}

func TestWebRtcTransportSetMaxIncomingBitrate_Succeeds(t *testing.T) {
	_, transport := setupWebRtcTest(t)

	err := transport.SetMaxIncomingBitrate(100000)

	assert.NoError(t, err)
}

func TestWebRtcTransportRestartIce_Succeeds(t *testing.T) {
	_, transport := setupWebRtcTest(t)

	previousIceUsernameFragment := transport.IceParameters().UsernameFragment
	previousIcePassword := transport.IceParameters().Password

	iceParameters, err := transport.RestartIce()

	assert.NoError(t, err)
	assert.NotEmpty(t, iceParameters.UsernameFragment)
	assert.NotEmpty(t, iceParameters.Password)
	assert.True(t, iceParameters.IceLite)
	assert.NotEmpty(t, transport.IceParameters().UsernameFragment)
	assert.NotEmpty(t, transport.IceParameters().Password)
	assert.NotEqual(t, transport.IceParameters().UsernameFragment, previousIceUsernameFragment)
	assert.NotEqual(t, transport.IceParameters().Password, previousIcePassword)
}

func TestWebRtcTransportEvents_Succeeds(t *testing.T) {
	_, transport := setupWebRtcTest(t)

	// Private API.
	channel := transport.channel

	called, iceState := 0, ""
	transport.On("icestatechange", func(state string) {
		called++
		iceState = state
	})

	data, _ := json.Marshal(H{"iceState": "completed"})

	channel.Emit(transport.Id(), "icestatechange", data)

	assert.Equal(t, called, 1)
	assert.Equal(t, iceState, "completed")

	iceSelectedTuple := TransportTuple{
		LocalIp:    "1.1.1.1",
		LocalPort:  1111,
		RemoteIp:   "2.2.2.2",
		RemotePort: 2222,
		Protocol:   "udp",
	}
	data, _ = json.Marshal(H{"iceSelectedTuple": iceSelectedTuple})

	called = 0
	transport.On("iceselectedtuplechange", func(tuple TransportTuple) {
		called++
		assert.Equal(t, iceSelectedTuple, tuple)
	})
	channel.Emit(transport.Id(), "iceselectedtuplechange", data)

	assert.Equal(t, called, 1)

	called, dtlsState := 0, ""
	transport.On("dtlsstatechange", func(state string) {
		called++
		dtlsState = state
	})
	data, _ = json.Marshal(H{"dtlsState": "connecting"})
	channel.Emit(transport.Id(), "dtlsstatechange", data)

	assert.Equal(t, called, 1)
	assert.Equal(t, dtlsState, "connecting")
	assert.Equal(t, transport.DtlsState(), "connecting")

	data, _ = json.Marshal(H{"dtlsState": "connected", "dtlsRemoteCert": "ABCD"})
	channel.Emit(transport.Id(), "dtlsstatechange", data)

	assert.Equal(t, called, 2)
	assert.Equal(t, dtlsState, "connected")
	assert.Equal(t, transport.DtlsState(), "connected")
	assert.Equal(t, transport.DtlsRemoteCert(), "ABCD")
}

func TestWebRtcTransport_MethodsRejectIfClosed(t *testing.T) {
	_, transport := setupWebRtcTest(t)

	called := 0
	transport.Observer().Once("close", func() { called++ })
	transport.Close()

	assert.Equal(t, called, 1)
	assert.True(t, transport.Closed())
	assert.Equal(t, transport.IceState(), "closed")
	assert.Empty(t, transport.IceSelectedTuple())
	assert.Equal(t, transport.DtlsState(), "closed")

	assert.Error(t, transport.Dump().Err())

	_, err := transport.GetStats()
	assert.Error(t, err)

	err = transport.Connect(transportConnectParams{})
	assert.Error(t, err)

	err = transport.SetMaxIncomingBitrate(0)
	assert.Error(t, err)

	_, err = transport.RestartIce()
	assert.Error(t, err)
}

func TestWebRtcTransport_EmitsIfRouterIsClosed(t *testing.T) {
	router2, _ := worker.CreateRouter(testWebRtcMediaCodecs)
	transport2, _ := router2.CreateWebRtcTransport(CreateWebRtcTransportParams{
		ListenIps: []ListenIp{
			{Ip: "127.0.0.1"},
		},
	})

	called := 0
	transport2.Observer().Once("close", func() { called++ })

	wg := sync.WaitGroup{}
	wg.Add(1)

	transport2.On("routerclose", func() {
		wg.Done()
	})

	router2.Close()

	wg.Wait()

	assert.Equal(t, called, 1)
	assert.True(t, transport2.Closed())
	assert.Equal(t, transport2.IceState(), "closed")
	assert.Empty(t, transport2.IceSelectedTuple())
	assert.Equal(t, transport2.DtlsState(), "closed")
}

func TestWebRtcTransport_EmitsIfWorkerIsClosed(t *testing.T) {
	_, transport := setupWebRtcTest(t)

	called := 0
	transport.Observer().Once("close", func() { called++ })

	wg := sync.WaitGroup{}
	wg.Add(1)

	transport.On("routerclose", func() {
		wg.Done()
	})

	worker.Close()

	wg.Wait()

	assert.Equal(t, called, 1)
	assert.True(t, transport.Closed())
	assert.Equal(t, transport.IceState(), "closed")
	assert.Empty(t, transport.IceSelectedTuple())
	assert.Equal(t, transport.DtlsState(), "closed")
}
