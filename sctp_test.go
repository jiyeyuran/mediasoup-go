package mediasoup

import (
	"fmt"
	"log"
	"net"
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
	recvBinaryMessages := 0
	recvStringMessages := 0
	var sendData []byte
	var recvData []byte

	dataConsumer.OnMessage(func(payload []byte, ppid SctpPayloadType) {
		recvData = append(recvData, payload...)
		switch ppid {
		case SctpPayloadWebRTCBinary:
			recvBinaryMessages++

		case SctpPayloadWebRTCString:
			recvStringMessages++
		}
	})

	for i := 0; i < numMessages/2; i++ {
		data := []byte(fmt.Sprintf("%d", i))
		_, err := stcpStream.WriteSCTP(data, sctp.PayloadTypeWebRTCBinary)
		require.NoError(t, err)
		sendData = append(sendData, data...)
	}

	for i := 0; i < numMessages/2; i++ {
		data := []byte(fmt.Sprintf("%d", i))
		_, err := stcpStream.WriteSCTP(data, sctp.PayloadTypeWebRTCString)
		require.NoError(t, err)
		sendData = append(sendData, data...)
	}

	// wait all messages are received
	time.Sleep(time.Millisecond * 10)

	assert.Equal(t, numMessages/2, recvBinaryMessages)
	assert.Equal(t, numMessages/2, recvStringMessages)
	assert.Equal(t, len(sendData), len(recvData))
	assert.Equal(t, string(sendData), string(recvData))

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
