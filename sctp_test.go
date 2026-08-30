package mediasoup

import (
	"fmt"
	"log"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/pion/logging"
	"github.com/pion/sctp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSctpMessage(t *testing.T) {
	router := createRouter(nil)
	transport, err := router.CreatePlainTransport(&PlainTransportOptions{
		ListenInfo: TransportListenInfo{
			Protocol:         TransportProtocolUDP,
			Ip:               "0.0.0.0",
			AnnouncedAddress: "127.0.0.1",
		},
		Comedia:    true,
		EnableSctp: true,
	})
	require.NoError(t, err)

	negotiatedCapabilities := make(chan SctpNegotiatedCapabilities, 1)
	transport.OnSctpNegotiatedCapabilities(func(capabilities SctpNegotiatedCapabilities) {
		select {
		case negotiatedCapabilities <- capabilities:
		default:
		}
	})

	plainTransportData := transport.Data().PlainTransportData
	remoteUdpIp := plainTransportData.Tuple.LocalAddress
	remoteUdpPort := plainTransportData.Tuple.LocalPort

	// Resolve the address for UDP (supports both IPv4 and IPv6)
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("[%s]:%d", remoteUdpIp, remoteUdpPort))
	if err != nil {
		log.Fatalf("Failed to resolve address: %v", err)
	}

	// Dial UDP connection
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		log.Fatalf("Failed to dial UDP connection: %v", err)
	}
	defer conn.Close()

	config := sctp.Config{
		NetConn:       conn,
		LoggerFactory: logging.NewDefaultLoggerFactory(),
	}
	association, err := sctp.Client(config)
	require.NoError(t, err)

	select {
	case capabilities := <-negotiatedCapabilities:
		assert.NotZero(t, capabilities.NegotiatedMaxOutboundStreams)
		assert.NotZero(t, capabilities.NegotiatedMaxInboundStreams)
		assert.Equal(t, &capabilities, transport.SctpNegotiatedCapabilities())

	case <-time.After(time.Second):
		t.Fatal("timeout waiting for the SCTP negotiated capabilities")
	}

	// Create an explicit SCTP outgoing stream with id 123 (id 0 is already used
	// by the implicit SCTP outgoing stream built-in the SCTP socket).
	var sctpSendStreamId = uint16(123)

	stcpStream, err := association.OpenStream(sctpSendStreamId, sctp.PayloadTypeWebRTCBinary)
	require.NoError(t, err)

	// Create a DataProducer with the corresponding SCTP stream id.
	dataProducer, err := transport.ProduceData(&DataProducerOptions{
		SctpStreamParameters: &SctpStreamParameters{
			StreamId: sctpSendStreamId,
			Ordered:  ref(true),
		},
		Label:    "go-sctp",
		Protocol: "foo & bar 😀😀😀",
	})
	require.NoError(t, err)

	transport2, err := router.CreateDirectTransport(nil)
	require.NoError(t, err)

	// Create a DataConsumer to receive messages from the DataProducer over the
	// direct transport.
	dataConsumer, err := transport2.ConsumeData(&DataConsumerOptions{
		DataProducerId: dataProducer.Id(),
	})
	require.NoError(t, err)

	numMessages := 20
	var (
		mu                 sync.Mutex
		recvBinaryMessages int
		recvStringMessages int
		sendData           []byte
		recvData           []byte
	)

	dataConsumer.OnMessage(func(payload []byte, ppid SctpPayloadType) {
		mu.Lock()
		defer mu.Unlock()
		recvData = append(recvData, payload...)
		switch ppid {
		case SctpPayloadWebRTCBinary:
			recvBinaryMessages++

		case SctpPayloadWebRTCString:
			recvStringMessages++
		}
	})

	for i := 0; i < numMessages/2; i++ {
		data := fmt.Appendf(nil, "%d", i)
		_, err = stcpStream.WriteSCTP(data, sctp.PayloadTypeWebRTCBinary)
		require.NoError(t, err)
		sendData = append(sendData, data...)
	}

	for i := 0; i < numMessages/2; i++ {
		data := fmt.Appendf(nil, "%d", i)
		_, err = stcpStream.WriteSCTP(data, sctp.PayloadTypeWebRTCString)
		require.NoError(t, err)
		sendData = append(sendData, data...)
	}

	// The messages travel over real SCTP and come back through the worker, so how
	// long they take is not something the test can assume.
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()

		return recvBinaryMessages+recvStringMessages == numMessages
	}, notificationTimeout, time.Millisecond, "not all messages arrived")

	mu.Lock()
	assert.Equal(t, numMessages/2, recvBinaryMessages)
	assert.Equal(t, numMessages/2, recvStringMessages)
	assert.Equal(t, len(sendData), len(recvData))
	assert.Equal(t, string(sendData), string(recvData))
	mu.Unlock()

	dataProducerStats, err := dataProducer.GetStats()
	assert.NoError(t, err)
	assert.Equal(t, &DataProducerStat{
		Type:             "data-producer",
		Timestamp:        dataProducerStats[0].Timestamp,
		Label:            dataProducer.Label(),
		Protocol:         dataProducer.Protocol(),
		MessagesReceived: uint64(numMessages),
		BytesReceived:    uint64(len(sendData)),
	}, dataProducerStats[0])

	// A SCTP DataConsumer reports its buffered amount when sending messages.
	sctpDataConsumer, err := transport.ConsumeData(&DataConsumerOptions{
		DataProducerId: dataProducer.Id(),
	})
	require.NoError(t, err)

	bufferedAmount, err := sctpDataConsumer.SendText("hello")
	assert.NoError(t, err)
	assert.LessOrEqual(t, bufferedAmount, uint32(len("hello")))

	// An empty string must travel as WebRTCStringEmpty, not WebRTCBinaryEmpty,
	// otherwise the remote peer decodes it as a binary message.
	_, err = sctpDataConsumer.SendText("")
	assert.NoError(t, err)

	recvStream, err := association.AcceptStream()
	require.NoError(t, err)
	require.NoError(t, recvStream.SetReadDeadline(time.Now().Add(time.Second)))

	buf := make([]byte, 128)
	n, ppid, err := recvStream.ReadSCTP(buf)
	require.NoError(t, err)
	assert.Equal(t, sctp.PayloadTypeWebRTCString, ppid)
	assert.Equal(t, "hello", string(buf[:n]))

	_, ppid, err = recvStream.ReadSCTP(buf)
	require.NoError(t, err)
	assert.Equal(t, sctp.PayloadTypeWebRTCStringEmpty, ppid)

	dataConumserStats, err := dataConsumer.GetStats()
	assert.NoError(t, err)
	assert.Equal(t, &DataConsumerStat{
		Type:         "data-consumer",
		Timestamp:    dataConumserStats[0].Timestamp,
		Label:        dataConsumer.Label(),
		Protocol:     dataConsumer.Protocol(),
		MessagesSent: uint64(numMessages),
		BytesSent:    uint64(len(sendData)),
	}, dataConumserStats[0])
}
