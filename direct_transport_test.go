package mediasoup

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

func TestDirectTransportTestingSuite(t *testing.T) {
	suite.Run(t, new(DirectTransportTestingSuite))
}

type DirectTransportTestingSuite struct {
	TestingSuite
	worker    *Worker
	router    *Router
	transport ITransport
}

func (suite *DirectTransportTestingSuite) SetupTest() {
	var err error
	suite.worker = CreateTestWorker()
	suite.router = CreateRouter(suite.worker)
	suite.NoError(err)

	transport, err := suite.router.CreateDirectTransport()
	suite.NoError(err)
	suite.transport = transport
}

func (suite *DirectTransportTestingSuite) TearDownTest() {
	suite.worker.Close()
}

func (suite *DirectTransportTestingSuite) TestRouterCreateDirectTransportSucceeds() {
	dump, _ := suite.router.Dump()
	suite.Equal([]string{suite.transport.Id()}, dump.TransportIds)

	onObserverNewTransport := NewMockFunc(suite.T())
	suite.router.Observer().Once("newtransport", onObserverNewTransport.Fn())

	transport1, _ := suite.router.CreateDirectTransport(DirectTransportOptions{
		MaxMessageSize: 1024,
		AppData:        H{"foo": "bar"},
	})
	onObserverNewTransport.ExpectCalledTimes(1)
	onObserverNewTransport.ExpectCalledWith(transport1)
	suite.False(transport1.Closed())
	suite.Equal(H{"foo": "bar"}, transport1.AppData())

	data1, _ := transport1.Dump()

	suite.Equal(transport1.Id(), data1.Id)
	suite.True(data1.Direct)
	suite.Empty(data1.DataProducerIds)
	suite.Empty(data1.DataConsumerIds)

	transport1.Close()
	suite.True(transport1.Closed())

	_, err := suite.router.CreateDirectTransport()
	suite.NoError(err)
}

func (suite *DirectTransportTestingSuite) TestDirectTransportGetStatsSucceeds() {
	data, _ := suite.transport.GetStats()
	suite.Len(data, 1)
	suite.Equal("direct-transport", data[0].Type)
	suite.NotZero(data[0].TransportId)
	suite.NotZero(data[0].Timestamp)
}

func (suite *DirectTransportTestingSuite) TestDirectTransportConnectSucceeds() {
	err := suite.transport.Connect(TransportConnectOptions{})
	suite.NoError(err)
}

func (suite *DirectTransportTestingSuite) TestDataProducerSendSucceeds() {
	transport2, _ := suite.router.CreateDirectTransport()
	dataProducer, _ := transport2.ProduceData(DataProducerOptions{
		Label:    "foo",
		Protocol: "bar",
		AppData:  H{"foot": "bar"},
	})
	dataConsumer, _ := transport2.ConsumeData(DataConsumerOptions{
		DataProducerId: dataProducer.Id(),
	})
	const numMessages = 200
	var sentMessageBytes = 0
	var recvMessageBytes = 0
	var lastSentMessageId = 0
	var lastRecvMessageId = 0

	done := make(chan struct{})

	go func() {
		for {
			lastSentMessageId++

			data := []byte(fmt.Sprintf("%d", lastSentMessageId))

			if lastSentMessageId < numMessages/2 {
				err := dataProducer.SendText(string(data))
				suite.NoError(err)
			} else {
				err := dataProducer.Send(data)
				suite.NoError(err)
			}

			sentMessageBytes += len(data)

			if lastSentMessageId == numMessages {
				break
			}
		}
	}()

	dataConsumer.On("message", func(payload []byte, ppid int) {
		recvMessageBytes += len(payload)
		id, err := strconv.Atoi(string(payload))
		suite.NoError(err)

		if id == numMessages {
			close(done)
		}

		if id < numMessages/2 {
			suite.EqualValues(PPID_WEBRTC_STRING, ppid)
		} else {
			suite.EqualValues(PPID_WEBRTC_BINARY, ppid)
		}

		lastRecvMessageId++

		suite.Equal(lastRecvMessageId, id)
	})

	select {
	case <-done:
	case <-time.NewTimer(2 * time.Duration(numMessages) * time.Millisecond).C:
		suite.FailNow("timeout")
	}

	suite.EqualValues(lastSentMessageId, numMessages)
	suite.EqualValues(lastRecvMessageId, numMessages)
	suite.EqualValues(sentMessageBytes, recvMessageBytes)

	dataProducerStats, err := dataProducer.GetStats()
	suite.NoError(err)
	suite.Equal(&DataProducerStat{
		Type:             "data-producer",
		Timestamp:        dataProducerStats[0].Timestamp,
		Label:            dataProducer.Label(),
		Protocol:         dataProducer.Protocol(),
		MessagesReceived: int64(numMessages),
		BytesReceived:    int64(sentMessageBytes),
	}, dataProducerStats[0])

	dataConumserStats, err := dataConsumer.GetStats()
	suite.NoError(err)
	suite.Equal(&DataConsumerStat{
		Type:         "data-consumer",
		Timestamp:    dataConumserStats[0].Timestamp,
		Label:        dataConsumer.Label(),
		Protocol:     dataConsumer.Protocol(),
		MessagesSent: int64(numMessages),
		BytesSent:    int64(sentMessageBytes),
	}, dataConumserStats[0])
}

func (suite *DirectTransportTestingSuite) TestDirectTransportMethodRejectIfclosed() {
	onObserverClose := NewMockFunc(suite.T())
	suite.transport.Observer().Once("close", onObserverClose.Fn())
	suite.transport.Close()

	onObserverClose.ExpectCalledTimes(1)
	suite.True(suite.transport.Closed())

	_, err := suite.transport.GetStats()
	suite.Error(err)
}

func (suite *DirectTransportTestingSuite) TestDirectTransportEmitRouterClosedIfRouterIsClosed() {
	onObserverClose := NewMockFunc(suite.T())
	suite.transport.Observer().Once("close", onObserverClose.Fn())
	suite.router.Close()

	onObserverClose.ExpectCalledTimes(1)
	suite.True(suite.transport.Closed())
}

func (suite *DirectTransportTestingSuite) TestDirectTransportEmitRouterClosedIfWorkerIsClosed() {
	onObserverClose := NewMockFunc(suite.T())
	suite.transport.Observer().Once("close", onObserverClose.Fn())
	suite.worker.Close()

	onObserverClose.ExpectCalledTimes(1)
	suite.True(suite.transport.Closed())
}
