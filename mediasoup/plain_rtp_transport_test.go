package mediasoup

import (
	"testing"

	"github.com/jiyeyuran/mediasoup-go/mediasoup/h264profile"
	"github.com/stretchr/testify/assert"
)

var testPlainMediaCodecs = []RtpCodecCapability{
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

func TestCreatePlainRtpTransport_Succeeds(t *testing.T) {
	router, _ := worker.CreateRouter(testPlainMediaCodecs)

	transport, err := router.CreatePlainRtpTransport(CreatePlainRtpTransportParams{
		ListenIp: ListenIp{Ip: "127.0.0.1", AnnouncedIp: "4.4.4.4"},
		RtcpMux:  false,
	})
	defer transport.Close()

	assert.NoError(t, err)

	type Dump struct {
		TransportIds []string
	}
	var data Dump
	router.Dump().Unmarshal(&data)

	assertJSONEq(t, Dump{TransportIds: []string{transport.Id()}}, data)

	var plainTransport *PlainRtpTransport
	called := 0

	router.Observer().Once("newtransport", func(t *PlainRtpTransport) {
		plainTransport, called = t, called+1
	})

	appData := map[string]string{"foo": "bar"}

	transport1, _ := router.CreatePlainRtpTransport(CreatePlainRtpTransportParams{
		ListenIp: ListenIp{Ip: "127.0.0.1", AnnouncedIp: "9.9.9.1"},
		RtcpMux:  true,
		AppData:  appData,
	})

	assert.Equal(t, called, 1)
	assert.Equal(t, transport1, plainTransport)
	assert.False(t, transport1.Closed())
	assert.Equal(t, appData, transport1.AppData())
	assert.Equal(t, transport1.Tuple().LocalIp, "9.9.9.1")
	assert.NotEmpty(t, transport1.Tuple().LocalPort)
	assert.Equal(t, transport1.Tuple().Protocol, "udp")
	assert.Empty(t, transport1.RtcpTuple())

	var data1 map[string]interface{}
	transport1.Dump().Unmarshal(&data1)

	assert.Equal(t, data1["id"], transport1.Id())
	assert.Empty(t, data1["producerIds"])
	assert.Empty(t, data1["consumerIds"])
	assertJSONEq(t, data1["tuple"], transport1.Tuple())
	assertJSONEq(t, data1["rtcpTuple"], transport1.RtcpTuple())
	assert.NotNil(t, data1["rtpHeaderExtensions"])
	assert.NotNil(t, data1["rtpListener"])

	transport1.Close()

	assert.True(t, transport1.Closed())

	_, err = router.CreatePlainRtpTransport(CreatePlainRtpTransportParams{
		ListenIp: ListenIp{Ip: "127.0.0.1"},
	})
	assert.NoError(t, err)

	transport2, _ := router.CreatePlainRtpTransport(CreatePlainRtpTransportParams{
		ListenIp: ListenIp{Ip: "127.0.0.1"},
		RtcpMux:  false,
	})
	defer transport.Close()

	t.Log(transport2.data)

	assert.False(t, transport2.Closed())
	assert.Empty(t, transport2.AppData())
	assert.NotNil(t, transport2.Tuple())
	assert.Equal(t, transport2.Tuple().LocalIp, "127.0.0.1")
	assert.NotEmpty(t, transport2.Tuple().LocalPort)
	assert.Equal(t, transport2.Tuple().Protocol, "udp")
	assert.NotNil(t, transport2.RtcpTuple())
	assert.Equal(t, transport2.RtcpTuple().LocalIp, "127.0.0.1")
	assert.NotEmpty(t, transport2.RtcpTuple().LocalPort)
	assert.Equal(t, transport2.RtcpTuple().Protocol, "udp")

	var data2 map[string]interface{}
	transport2.Dump().Unmarshal(&data2)

	assert.Equal(t, data2["id"], transport2.Id())
	assertJSONEq(t, data2["tuple"], transport2.Tuple())
	assertJSONEq(t, data2["rtcpTuple"], transport2.RtcpTuple())
}

func TestCreatePlainRtpTransport_TypeError(t *testing.T) {
	router, _ := worker.CreateRouter(testPlainMediaCodecs)

	_, err := router.CreatePlainRtpTransport(CreatePlainRtpTransportParams{})
	assert.IsType(t, err, NewTypeError(""))

	_, err = router.CreatePlainRtpTransport(CreatePlainRtpTransportParams{
		ListenIp: ListenIp{Ip: "123"},
	})
	assert.IsType(t, err, NewTypeError(""))
}

func TestCreatePlainRtpTransport_Error(t *testing.T) {
	router, _ := worker.CreateRouter(testPlainMediaCodecs)

	_, err := router.CreatePlainRtpTransport(CreatePlainRtpTransportParams{
		ListenIp: ListenIp{Ip: "8.8.8.8"},
	})
	assert.Error(t, err)
}

func TestPlaintRtpTransport_GetStats_Succeeds(t *testing.T) {
	router, _ := worker.CreateRouter(testPlainMediaCodecs)

	transport, _ := router.CreatePlainRtpTransport(CreatePlainRtpTransportParams{
		ListenIp: ListenIp{Ip: "127.0.0.1", AnnouncedIp: "4.4.4.4"},
		RtcpMux:  false,
	})

	var data []struct {
		Type          string
		TransportId   string
		Timestamp     uint32
		BytesReceived uint32
		BytesSent     uint32
		Tuple         *TransportTuple
		RtcpTuple     *TransportTuple
	}
	transport.GetStats().Unmarshal(&data)

	assert.Len(t, data, 1)
	assert.Equal(t, data[0].Type, "transport")
	assert.NotEmpty(t, data[0].TransportId)
	assert.NotEmpty(t, data[0].Timestamp)
	assert.Zero(t, data[0].BytesReceived)
	assert.Zero(t, data[0].BytesSent)
	assert.Nil(t, data[0].Tuple)
	assert.Nil(t, data[0].RtcpTuple)
}

func TestPlaintRtpTransport_Connect_Succeeds(t *testing.T) {
	router, _ := worker.CreateRouter(testPlainMediaCodecs)

	transport, _ := router.CreatePlainRtpTransport(CreatePlainRtpTransportParams{
		ListenIp: ListenIp{Ip: "127.0.0.1", AnnouncedIp: "4.4.4.4"},
		RtcpMux:  false,
	})

	err := transport.Connect(transportConnectParams{
		Ip:       "1.2.3.4",
		Port:     1234,
		RtcpPort: 1235,
	})
	assert.NoError(t, err)

	err = transport.Connect(transportConnectParams{
		Ip:       "1.2.3.4",
		Port:     1234,
		RtcpPort: 1235,
	})
	assert.Error(t, err)

	tuple, rtcpTuple := transport.Tuple(), transport.RtcpTuple()
	assert.Equal(t, tuple.RemoteIp, "1.2.3.4")
	assert.EqualValues(t, tuple.RemotePort, 1234)
	assert.Equal(t, tuple.Protocol, "udp")
	assert.Equal(t, rtcpTuple.RemoteIp, "1.2.3.4")
	assert.EqualValues(t, rtcpTuple.RemotePort, 1235)
	assert.Equal(t, tuple.Protocol, "udp")
}

func TestPlaintRtpTransport_Connect_TypeError(t *testing.T) {
	router, _ := worker.CreateRouter(testPlainMediaCodecs)

	transport, _ := router.CreatePlainRtpTransport(CreatePlainRtpTransportParams{
		ListenIp: ListenIp{Ip: "127.0.0.1", AnnouncedIp: "4.4.4.4"},
		RtcpMux:  false,
	})
	err := transport.Connect(transportConnectParams{})
	assert.IsType(t, err, NewTypeError(""))

	err = transport.Connect(transportConnectParams{
		Ip: "::::1234",
	})
	assert.IsType(t, err, NewTypeError(""))

	err = transport.Connect(transportConnectParams{
		Ip:   "127.0.0.1",
		Port: 1234,
	})
	assert.IsType(t, err, NewTypeError(""))
}

func TestPlainRtpTransport_Reject_If_Closed(t *testing.T) {
	router, _ := worker.CreateRouter(testPlainMediaCodecs)
	transport, _ := router.CreatePlainRtpTransport(CreatePlainRtpTransportParams{
		ListenIp: ListenIp{Ip: "127.0.0.1", AnnouncedIp: "4.4.4.4"},
		RtcpMux:  false,
	})

	called := 0
	transport.Observer().Once("close", func() {
		called++
	})
	transport.Close()

	assert.Equal(t, called, 1)
	assert.True(t, transport.Closed())

	assert.Error(t, transport.Dump().Err())
	assert.Error(t, transport.GetStats().Err())
	assert.Error(t, transport.Connect(transportConnectParams{}))
}

func TestPlaintRtpTransport_Emits_Routerclose_If_RouterClosed(t *testing.T) {
	router, _ := worker.CreateRouter(testPlainMediaCodecs)
	transport, _ := router.CreatePlainRtpTransport(CreatePlainRtpTransportParams{
		ListenIp: ListenIp{Ip: "127.0.0.1", AnnouncedIp: "4.4.4.4"},
		RtcpMux:  false,
	})

	called := 0
	transport.Observer().Once("close", func() {
		called++
	})

	router.Close()

	assert.Equal(t, called, 1)
	assert.True(t, transport.Closed())
}

func TestPlaintRtpTransport_Emits_Routerclose_If_WorkerClosed(t *testing.T) {
	router, _ := worker.CreateRouter(testPlainMediaCodecs)
	transport, _ := router.CreatePlainRtpTransport(CreatePlainRtpTransportParams{
		ListenIp: ListenIp{Ip: "127.0.0.1", AnnouncedIp: "4.4.4.4"},
		RtcpMux:  false,
	})

	called := 0
	transport.Observer().Once("close", func() {
		called++
	})

	worker.Close()

	assert.Equal(t, called, 1)
	assert.True(t, transport.Closed())
}
