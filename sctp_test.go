package mediasoup

import (
	"fmt"
	"net"
	"strconv"
	"sync"
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
	stcpAssoci   *sctp.Association
	stcpStream   *sctp.Stream
}

func (suite *SctpTestingSuite) SetupTest() {
	var err error
	suite.worker = CreateTestWorker()
	suite.router, err = suite.worker.CreateRouter(RouterOptions{})
	suite.NoError(err)
	transport, err := suite.router.CreatePlainTransport(func(o *PlainTransportOptions) {
		o.ListenIp = TransportListenIp{
			Ip:          "0.0.0.0",
			AnnouncedIp: "127.0.0.1",
		}
		o.Comedia = true
		o.EnableSctp = true
		o.NumSctpStreams = NumSctpStreams{OS: 256, MIS: 256}
	})
	suite.NoError(err)

	remoteUdpIp := transport.Tuple().LocalIp
	remoteUdpPort := transport.Tuple().LocalPort

	conn, err := net.Dial("udp", fmt.Sprintf("%s:%d", remoteUdpIp, remoteUdpPort))
	suite.NoError(err)

	net.ListenUDP("udp", conn.LocalAddr().(*net.UDPAddr))

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

	suite.stcpAssoci = association
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
	numMessages := 200
	sentMessageBytes := 0
	recvMessageBytes := 0
	lastSentMessageId := 0
	lastRecvMessageId := 0

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		for {
			time.Sleep(time.Millisecond)

			lastSentMessageId++

			data := fmt.Sprintf("%d", lastSentMessageId)
			payloadType := sctp.PayloadTypeWebRTCBinary

			if lastSentMessageId < numMessages/2 {
				payloadType = sctp.PayloadTypeWebRTCString
			}

			n, err := suite.stcpStream.WriteSCTP([]byte(data), payloadType)
			suite.NoError(err)

			sentMessageBytes += n

			if lastSentMessageId == numMessages {
				break
			}
		}
	}()

	suite.dataConsumer.On("message", func(payload []byte, ppid int) {
		suite.T().Logf("payload: %s", payload)
		recvMessageBytes += len(payload)
		id, err := strconv.Atoi(string(payload))
		suite.NoError(err)

		if id == numMessages {
			wg.Done()
		}

		if id < numMessages/2 {
			suite.EqualValues(sctp.PayloadTypeWebRTCString, ppid)
		} else {
			suite.EqualValues(sctp.PayloadTypeWebRTCBinary, ppid)
		}

		lastRecvMessageId++

		suite.Equal(lastRecvMessageId, id)
	})

	wg.Wait()

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
