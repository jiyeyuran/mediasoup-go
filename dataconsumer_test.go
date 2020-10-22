package mediasoup

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestDataConsumerTestingSuite(t *testing.T) {
	suite.Run(t, new(DataConsumerTestingSuite))
}

type DataConsumerTestingSuite struct {
	TestingSuite
	worker       *Worker
	router       *Router
	transport1   ITransport
	transport2   ITransport
	transport3   ITransport
	dataProducer *DataProducer
}

func (suite *DataConsumerTestingSuite) SetupTest() {
	dataProducerParameters := DataProducerOptions{
		SctpStreamParameters: &SctpStreamParameters{
			StreamId:          12345,
			Ordered:           Bool(false),
			MaxPacketLifeTime: 5000,
		},
		Label:    "foo",
		Protocol: "bar",
	}
	var err error
	suite.worker = CreateTestWorker()
	suite.router, err = suite.worker.CreateRouter(RouterOptions{})
	suite.NoError(err)
	suite.transport1, err = suite.router.CreateWebRtcTransport(func(o *WebRtcTransportOptions) {
		o.ListenIps = []TransportListenIp{
			{Ip: "127.0.0.1"},
		}
		o.EnableSctp = true
	})
	suite.NoError(err)
	suite.transport2, err = suite.router.CreatePlainTransport(func(o *PlainTransportOptions) {
		o.ListenIp = TransportListenIp{
			Ip: "127.0.0.1",
		}
		o.EnableSctp = true
	})
	suite.NoError(err)
	suite.transport3, err = suite.router.CreateDirectTransport()
	suite.NoError(err)

	observer := NewMockFunc(suite.T())
	suite.transport1.Observer().Once("newdataproducer", observer.Fn())

	suite.dataProducer, err = suite.transport1.ProduceData(dataProducerParameters)
	suite.NoError(err)
	observer.ExpectCalledTimes(1)
	observer.ExpectCalledWith(suite.dataProducer)
}

func (suite *DataConsumerTestingSuite) TearDownTest() {
	suite.worker.Close()
}

func (suite *DataConsumerTestingSuite) TestTransportConsumeDataSucceeds() {
	onObserverNewDataConsumer := NewMockFunc(suite.T())

	suite.transport2.Observer().Once("newdataconsumer", onObserverNewDataConsumer.Fn())

	dataConsumer1, err := suite.transport2.ConsumeData(DataConsumerOptions{
		DataProducerId:    suite.dataProducer.Id(),
		MaxPacketLifeTime: 4000,
		AppData:           H{"baz": "LOL"},
	})

	suite.NoError(err)
	onObserverNewDataConsumer.ExpectCalledTimes(1)
	onObserverNewDataConsumer.ExpectCalledWith(dataConsumer1)
	suite.Equal(suite.dataProducer.Id(), dataConsumer1.DataProducerId())
	suite.False(dataConsumer1.Closed())
	suite.Equal(DataConsumerType_Sctp, dataConsumer1.Type())
	suite.False(*dataConsumer1.SctpStreamParameters().Ordered)
	suite.Equal(4000, dataConsumer1.SctpStreamParameters().MaxPacketLifeTime)
	suite.Zero(dataConsumer1.SctpStreamParameters().MaxRetransmits)
	suite.Equal("foo", dataConsumer1.Label())
	suite.Equal("bar", dataConsumer1.Protocol())
	suite.Equal(H{"baz": "LOL"}, dataConsumer1.AppData())

	routerDump, _ := suite.router.Dump()
	expectedDump := RouterDump{
		MapDataProducerIdDataConsumerIds: map[string][]string{
			suite.dataProducer.Id(): {dataConsumer1.Id()},
		},
		MapDataConsumerIdDataProducerId: map[string]string{
			dataConsumer1.Id(): suite.dataProducer.Id(),
		},
	}

	suite.Equal(expectedDump.MapDataProducerIdDataConsumerIds, routerDump.MapDataProducerIdDataConsumerIds)
	suite.Equal(expectedDump.MapDataConsumerIdDataProducerId, routerDump.MapDataConsumerIdDataProducerId)

	transportDump, _ := suite.transport2.Dump()
	suite.Equal(suite.transport2.Id(), transportDump.Id)
	suite.Empty(transportDump.ProducerIds)
	suite.Equal([]string{dataConsumer1.Id()}, transportDump.DataConsumerIds)
}

func (suite *DataConsumerTestingSuite) TestDataConsumerDumpSucceeds() {
	dataConsumer1, err := suite.transport2.ConsumeData(DataConsumerOptions{
		DataProducerId:    suite.dataProducer.Id(),
		MaxPacketLifeTime: 4000,
		AppData:           H{"baz": "LOL"},
	})
	suite.NoError(err)

	data, err := dataConsumer1.Dump()
	suite.NoError(err)
	suite.Equal(dataConsumer1.Id(), data.Id)
	suite.Equal(dataConsumer1.DataProducerId(), data.DataProducerId)
	suite.Equal("sctp", data.Type)
	suite.NotNil(dataConsumer1.SctpStreamParameters())
	suite.Equal(dataConsumer1.SctpStreamParameters().StreamId, data.SctpStreamParameters.StreamId)
	suite.False(*data.SctpStreamParameters.Ordered)
	suite.Equal(4000, data.SctpStreamParameters.MaxPacketLifeTime)
	suite.Zero(data.SctpStreamParameters.MaxRetransmits)
	suite.Equal("foo", data.Label)
	suite.Equal("bar", data.Protocol)
}

func (suite *DataConsumerTestingSuite) TestTransportConsumeDataOnADirectTransportSucceeds() {
	onObserverNewDataConsumer := NewMockFunc(suite.T())

	suite.transport3.Observer().Once("newdataconsumer", onObserverNewDataConsumer.Fn())

	dataConsumer2, err := suite.transport3.ConsumeData(DataConsumerOptions{
		DataProducerId: suite.dataProducer.Id(),
		AppData:        H{"hehe": "HEHE"},
	})

	suite.NoError(err)
	onObserverNewDataConsumer.ExpectCalledTimes(1)
	onObserverNewDataConsumer.ExpectCalledWith(dataConsumer2)
	suite.Equal(suite.dataProducer.Id(), dataConsumer2.DataProducerId())
	suite.False(dataConsumer2.Closed())
	suite.EqualValues(DataConsumerType_Direct, dataConsumer2.Type())
	suite.Nil(dataConsumer2.SctpStreamParameters())
	suite.Equal("foo", dataConsumer2.Label())
	suite.Equal("bar", dataConsumer2.Protocol())
	suite.Equal(H{"hehe": "HEHE"}, dataConsumer2.AppData())

	transportDump, _ := suite.transport3.Dump()
	suite.Equal(suite.transport3.Id(), transportDump.Id)
	suite.Empty(transportDump.ProducerIds)
	suite.Equal([]string{dataConsumer2.Id()}, transportDump.DataConsumerIds)
}

func (suite *DataConsumerTestingSuite) TestDataConsumerDumpOnADirectTransportSucceeds() {
	dataConsumer2, err := suite.transport3.ConsumeData(DataConsumerOptions{
		DataProducerId:    suite.dataProducer.Id(),
		MaxPacketLifeTime: 4000,
		AppData:           H{"hehe": "HEHE"},
	})
	suite.NoError(err)

	data, err := dataConsumer2.Dump()
	suite.NoError(err)
	suite.Equal(dataConsumer2.Id(), data.Id)
	suite.Equal(dataConsumer2.DataProducerId(), data.DataProducerId)
	suite.Equal("direct", data.Type)
	suite.Nil(data.SctpStreamParameters)
	suite.Equal("foo", data.Label)
	suite.Equal("bar", data.Protocol)
}

func (suite *DataConsumerTestingSuite) TestdataConsumerGetStatsOnADirectTransportSucceeds() {
	dataConsumer2, _ := suite.transport3.ConsumeData(DataConsumerOptions{
		DataProducerId:    suite.dataProducer.Id(),
		MaxPacketLifeTime: 4000,
		AppData:           H{"hehe": "HEHE"},
	})
	stats, err := dataConsumer2.GetStats()
	suite.NoError(err)

	suite.Equal(&DataConsumerStat{
		Type:      "data-consumer",
		Timestamp: stats[0].Timestamp,
		Label:     dataConsumer2.Label(),
		Protocol:  dataConsumer2.Protocol(),
	}, stats[0])
}

func (suite *DataConsumerTestingSuite) TestDataConsumerCloseSucceeds() {
	dataConsumer1, _ := suite.transport2.ConsumeData(DataConsumerOptions{
		DataProducerId:    suite.dataProducer.Id(),
		MaxPacketLifeTime: 4000,
		AppData:           H{"baz": "LOL"},
	})
	dataConsumer2, _ := suite.transport3.ConsumeData(DataConsumerOptions{
		DataProducerId:    suite.dataProducer.Id(),
		MaxPacketLifeTime: 4000,
		AppData:           H{"hehe": "HEHE"},
	})
	onObserverNewDataConsumer := NewMockFunc(suite.T())

	dataConsumer1.Observer().Once("close", onObserverNewDataConsumer.Fn())
	dataConsumer1.Close()

	onObserverNewDataConsumer.ExpectCalledTimes(1)
	suite.True(dataConsumer1.Closed())

	routerDump, _ := suite.router.Dump()
	expectedDump := RouterDump{
		MapDataProducerIdDataConsumerIds: map[string][]string{
			suite.dataProducer.Id(): {dataConsumer2.Id()},
		},
		MapDataConsumerIdDataProducerId: map[string]string{
			dataConsumer2.Id(): suite.dataProducer.Id(),
		},
	}

	suite.Equal(expectedDump.MapDataProducerIdDataConsumerIds, routerDump.MapDataProducerIdDataConsumerIds)
	suite.Equal(expectedDump.MapDataConsumerIdDataProducerId, routerDump.MapDataConsumerIdDataProducerId)

	transportDump, _ := suite.transport2.Dump()
	suite.Equal(suite.transport2.Id(), transportDump.Id)
	suite.Empty(transportDump.ProducerIds)
	suite.Empty(transportDump.DataConsumerIds)
}

func (suite *DataConsumerTestingSuite) TestConsumerMethodsRejectIfClosed() {
	dataConsumer1, _ := suite.transport2.ConsumeData(DataConsumerOptions{
		DataProducerId:    suite.dataProducer.Id(),
		MaxPacketLifeTime: 4000,
		AppData:           H{"baz": "LOL"},
	})
	dataConsumer1.Close()

	_, err := dataConsumer1.Dump()
	suite.Error(err)

	_, err = dataConsumer1.GetStats()
	suite.Error(err)
}

func (suite *DataConsumerTestingSuite) TestDataConsumerEmitsDataproducercloseIfDataProducerIsClosed() {
	dataConsumer1, _ := suite.transport2.ConsumeData(DataConsumerOptions{
		DataProducerId:    suite.dataProducer.Id(),
		MaxPacketLifeTime: 4000,
		AppData:           H{"baz": "LOL"},
	})

	onObserverClose := NewMockFunc(suite.T())
	dataConsumer1.Observer().Once("close", onObserverClose.Fn())
	suite.dataProducer.Close()
	onObserverClose.ExpectCalledTimes(1)
	suite.True(dataConsumer1.Closed())
}

func (suite *DataConsumerTestingSuite) TestDataConsumerEmitsDataproducercloseIfTransportIsClosed() {
	dataConsumer1, _ := suite.transport2.ConsumeData(DataConsumerOptions{
		DataProducerId: suite.dataProducer.Id(),
	})

	onObserverClose := NewMockFunc(suite.T())
	fn := onObserverClose.Fn()

	dataConsumer1.Observer().Once("close", fn)
	dataConsumer1.On("transportclose", fn)

	suite.transport2.Close()

	onObserverClose.ExpectCalledTimes(2)
	suite.True(dataConsumer1.Closed())

	routerDump, _ := suite.router.Dump()
	suite.Empty(routerDump.MapDataConsumerIdDataProducerId)
	suite.Equal(map[string][]string{
		suite.dataProducer.Id(): {},
	}, routerDump.MapDataProducerIdDataConsumerIds)
}
