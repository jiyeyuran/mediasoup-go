package mediasoup

import (
	"fmt"
	"net"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pion/logging"
	"github.com/pion/sctp"
	"github.com/stretchr/testify/suite"
)

var sctpSendStreamId uint16

func TestSctpTestingSuite(t *testing.T) {
	suite.Run(t, new(SctpTestingSuite))
}

type SctpTestingSuite struct {
	TestingSuite
	worker       *Worker
	router       *Router
	dataProducer *DataProducer
	dataConsumer *DataConsumer
	stcpStream   *sctp.Stream
}

func (suite *SctpTestingSuite) SetupTest() {
	var err error
	suite.worker = CreateTestWorker()
	suite.router = CreateRouter(suite.worker)
	suite.NoError(err)
	transport, err := suite.router.CreatePlainTransport(PlainTransportOptions{
		ListenIp: TransportListenIp{
			Ip:          "0.0.0.0",
			AnnouncedIp: "127.0.0.1",
		},
		Comedia:        true,
		EnableSctp:     true,
		NumSctpStreams: NumSctpStreams{OS: 256, MIS: 256},
	})
	suite.NoError(err)

	remoteUdpIp := transport.Tuple().LocalIp
	remoteUdpPort := transport.Tuple().LocalPort

	conn, err := net.Dial("udp", fmt.Sprintf("%s:%d", remoteUdpIp, remoteUdpPort))
	suite.NoError(err)

	config := sctp.Config{
		NetConn:       conn,
		LoggerFactory: logging.NewDefaultLoggerFactory(),
	}
	association, err := sctp.Client(config)
	suite.NoError(err)

	// Create an explicit SCTP outgoing stream with id 123 (id 0 is already used
	// by the implicit SCTP outgoing stream built-in the SCTP socket).
	sctpSendStreamId = uint16(123)

	stream, err := association.OpenStream(sctpSendStreamId, sctp.PayloadTypeWebRTCBinary)
	suite.NoError(err)

	suite.stcpStream = stream

	// Create a DataProducer with the corresponding SCTP stream id.
	dataProducer, err := transport.ProduceData(DataProducerOptions{
		SctpStreamParameters: &SctpStreamParameters{
			StreamId: sctpSendStreamId,
			Ordered:  Bool(true),
		},
		Label:    "go-sctp",
		Protocol: "foo & bar 😀😀😀",
	})
	suite.NoError(err)

	suite.dataProducer = dataProducer

	transport2, err := suite.router.CreateDirectTransport()
	suite.NoError(err)

	// Create a DataConsumer to receive messages from the DataProducer over the
	// direct transport.
	dataConsumer, err := transport2.ConsumeData(DataConsumerOptions{
		DataProducerId: dataProducer.Id(),
	})
	suite.NoError(err)

	suite.dataConsumer = dataConsumer
}

func (suite *SctpTestingSuite) TearDownTest() {
	suite.stcpStream.Close()
	suite.worker.Close()
}

func (suite *SctpTestingSuite) TestOrderedDataProducerDeliversAllSCTPMessagesToTheDataConsumer() {
	numMessages := uint32(200)
	sentMessageBytes := uint32(0)
	recvMessageBytes := uint32(0)
	lastSentMessageId := uint32(0)
	lastRecvMessageId := uint32(0)

	done := make(chan struct{})

	suite.dataConsumer.On("message", func(payload []byte, ppid int) {
		recvMessageBytes += uint32(len(payload))

		id64, err := strconv.ParseUint(string(payload), 10, 32)
		suite.NoError(err)

		id := uint32(id64)

		if id == numMessages {
			close(done)
		}

		if id < numMessages/2 {
			suite.EqualValues(sctp.PayloadTypeWebRTCString, ppid)
		} else {
			suite.EqualValues(sctp.PayloadTypeWebRTCBinary, ppid)
		}

		suite.Equal(atomic.AddUint32(&lastRecvMessageId, 1), id)
	})

	for {
		time.Sleep(time.Millisecond)

		lastSentMessageId++

		data := []byte(fmt.Sprintf("%d", lastSentMessageId))
		payloadType := sctp.PayloadTypeWebRTCBinary

		if lastSentMessageId < numMessages/2 {
			payloadType = sctp.PayloadTypeWebRTCString
		}

		n, err := suite.stcpStream.WriteSCTP(data, payloadType)
		suite.NoError(err)

		sentMessageBytes += uint32(n)

		if lastSentMessageId == numMessages {
			break
		}
	}

	select {
	case <-done:
	case <-time.NewTimer(2 * time.Duration(numMessages) * time.Millisecond).C:
		suite.FailNow("timeout")
	}

	suite.EqualValues(lastSentMessageId, numMessages)
	suite.EqualValues(lastRecvMessageId, numMessages)
	suite.EqualValues(sentMessageBytes, recvMessageBytes)

	dataProducerStats, err := suite.dataProducer.GetStats()
	suite.NoError(err)
	suite.Equal(&DataProducerStat{
		Type:             "data-producer",
		Timestamp:        dataProducerStats[0].Timestamp,
		Label:            suite.dataProducer.Label(),
		Protocol:         suite.dataProducer.Protocol(),
		MessagesReceived: int64(numMessages),
		BytesReceived:    int64(sentMessageBytes),
	}, dataProducerStats[0])

	dataConumserStats, err := suite.dataConsumer.GetStats()
	suite.NoError(err)
	suite.Equal(&DataConsumerStat{
		Type:         "data-consumer",
		Timestamp:    dataConumserStats[0].Timestamp,
		Label:        suite.dataConsumer.Label(),
		Protocol:     suite.dataConsumer.Protocol(),
		MessagesSent: int64(numMessages),
		BytesSent:    int64(sentMessageBytes),
	}, dataConumserStats[0])
}
