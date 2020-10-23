package mediasoup

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ishidawataru/sctp"
	"github.com/stretchr/testify/suite"
)

var sctpSendStreamId int

func TestSctpTestingSuite(t *testing.T) {
	suite.Run(t, new(SctpTestingSuite))
}

type SctpTestingSuite struct {
	TestingSuite
	worker       *Worker
	router       *Router
	dataProducer *DataProducer
	dataConsumer *DataConsumer
	stcpSocket   *sctp.SCTPConn
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
		o.EnableSctp = true
		o.Comedia = true
		o.NumSctpStreams = NumSctpStreams{OS: 256, MIS: 256}
	})

	remoteUdpIp := transport.Tuple().LocalIp
	remoteUdpPort := transport.Tuple().LocalPort
	sctpParameters := transport.SctpParameters()
	OS, MIS := sctpParameters.OS, sctpParameters.MIS

	laddr := &sctp.SCTPAddr{
		IPAddrs: []net.IPAddr{
			{IP: net.ParseIP("127.0.0.1")},
		},
		Port: 12345,
	}
	raddr := &sctp.SCTPAddr{
		IPAddrs: []net.IPAddr{
			{IP: net.ParseIP(remoteUdpIp)},
		},
		Port: int(remoteUdpPort),
	}

	conn, err := sctp.DialSCTPExt("sctp", laddr, raddr, sctp.InitMsg{
		NumOstreams:  uint16(OS),
		MaxInstreams: uint16(MIS),
	})
	suite.NoError(err)

	suite.stcpSocket = conn

	// Create an explicit SCTP outgoing stream with id 123 (id 0 is already used
	// by the implicit SCTP outgoing stream built-in the SCTP socket).
	sctpSendStreamId = 123

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

	// Create a DataConsumer to receive messages from the DataProducer over the
	// same transport.
	dataConsumer, err := transport.ConsumeData(DataConsumerOptions{
		DataProducerId: dataProducer.Id(),
	})
	suite.NoError(err)

	suite.dataConsumer = dataConsumer
}

func (suite *SctpTestingSuite) TearDownTest() {
	suite.stcpSocket.Close()
	suite.worker.Close()
}

func (suite *SctpTestingSuite) TestOrderedDataProducerDeliversAllSCTPMessagesToTheDataConsumer() {
	onStream := NewMockFunc(suite.T())

	numMessages := 200
	sentMessageBytes := 0
	recvMessageBytes := 0
	lastSentMessageId := 0
	lastRecvMessageId := 0

	// It must be zero because it's the first DataConsumer on the transport.
	suite.Zero(suite.dataConsumer.SctpStreamParameters().StreamId)

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		for {
			time.Sleep(10 * time.Millisecond)

			lastSentMessageId++
			ppid := PPID_WEBRTC_BINARY

			if lastSentMessageId < numMessages/2 {
				ppid = PPID_WEBRTC_STRING
			}

			data := fmt.Sprintf("%d", lastSentMessageId)
			info := &sctp.SndRcvInfo{
				Stream: uint16(sctpSendStreamId),
				PPID:   uint32(ppid),
			}

			suite.stcpSocket.SCTPWrite([]byte(data), info)
			sentMessageBytes += len(data)

			if lastSentMessageId == numMessages {
				break
			}
		}
	}()

	go func() {
		defer wg.Done()
		buf := make([]byte, 512)
		for {
			n, info, err := suite.stcpSocket.SCTPRead(buf)
			suite.NoError(err)

			recvMessageBytes += n

			// It must be zero because it's the first SCTP incoming stream (so first
			// DataConsumer).
			suite.Equal(0, info.Stream)

			data := buf[0:n]
			id, err := strconv.Atoi(string(data))
			suite.NoError(err)

			if id == numMessages {
				break
			}

			if id < numMessages/2 {
				suite.EqualValues(PPID_WEBRTC_STRING, info.PPID)
			} else {
				suite.EqualValues(PPID_WEBRTC_BINARY, info.PPID)
			}

			lastRecvMessageId++

			suite.Equal(lastRecvMessageId, id)
		}
	}()

	wg.Wait()

	onStream.ExpectCalledTimes(1)
	suite.EqualValues(lastSentMessageId, numMessages)
	suite.EqualValues(lastRecvMessageId, numMessages)
	suite.EqualValues(sentMessageBytes, recvMessageBytes)

	dataProducerStats, err := suite.dataProducer.GetStats()
	suite.NoError(err)
	suite.Equal(DataProducerStat{
		Type:             "data-producer",
		Timestamp:        dataProducerStats[0].Timestamp,
		Label:            suite.dataProducer.Label(),
		Protocol:         suite.dataProducer.Protocol(),
		MessagesReceived: int64(numMessages),
		BytesReceived:    int64(sentMessageBytes),
	}, dataProducerStats[0])

	dataConumserStats, err := suite.dataConsumer.GetStats()
	suite.NoError(err)
	suite.Equal(DataConsumerStat{
		Type:         "data-consumer",
		Timestamp:    dataConumserStats[0].Timestamp,
		Label:        suite.dataConsumer.Label(),
		Protocol:     suite.dataConsumer.Protocol(),
		MessagesSent: int64(numMessages),
		BytesSent:    int64(recvMessageBytes),
	}, dataConumserStats[0])
}
