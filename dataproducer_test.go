package mediasoup

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestDataProducerTestingSuite(t *testing.T) {
	suite.Run(t, new(DataProducerTestingSuite))
}

type DataProducerTestingSuite struct {
	TestingSuite
	worker     *Worker
	router     *Router
	transport1 ITransport
	transport2 ITransport
}

func (suite *DataProducerTestingSuite) SetupTest() {
	suite.worker = CreateTestWorker()
	suite.router = CreateRouter(suite.worker)
	suite.transport1, _ = suite.router.CreateWebRtcTransport(WebRtcTransportOptions{
		ListenIps: []TransportListenIp{
			{Ip: "127.0.0.1"},
		},
		EnableSctp: true,
	})
	suite.transport2, _ = suite.router.CreatePlainTransport(PlainTransportOptions{
		ListenIp: TransportListenIp{
			Ip: "127.0.0.1",
		},
		EnableSctp: true,
	})
}

func (suite *DataProducerTestingSuite) TearDownTest() {
	suite.worker.Close()
}

func (suite *DataProducerTestingSuite) TestTransport1ProduceDataSucceeds() {
	onObserverNewDataProducer := NewMockFunc(suite.T())
	suite.transport1.Observer().Once("newdataproducer", onObserverNewDataProducer.Fn())

	dataProducer1, err := suite.transport1.ProduceData(DataProducerOptions{
		SctpStreamParameters: &SctpStreamParameters{StreamId: 666},
		Label:                "foo",
		Protocol:             "bar",
		AppData:              H{"foo": 1, "bar": "2"},
	})
	suite.NoError(err)
	onObserverNewDataProducer.ExpectCalledTimes(1)
	onObserverNewDataProducer.ExpectCalledWith(dataProducer1)
	suite.False(dataProducer1.Closed())
	suite.Equal(DataProducerType_Sctp, dataProducer1.Type())
	suite.NotNil(dataProducer1.SctpStreamParameters())
	suite.EqualValues(666, dataProducer1.SctpStreamParameters().StreamId)
	suite.True(*dataProducer1.SctpStreamParameters().Ordered)
	suite.Zero(dataProducer1.SctpStreamParameters().MaxPacketLifeTime)
	suite.Zero(dataProducer1.SctpStreamParameters().MaxRetransmits)
	suite.Equal("foo", dataProducer1.Label())
	suite.Equal("bar", dataProducer1.Protocol())
	suite.Equal(H{"foo": 1, "bar": "2"}, dataProducer1.AppData())

	routerDump, _ := suite.router.Dump()
	expectedDump := RouterDump{
		MapDataProducerIdDataConsumerIds: map[string][]string{
			dataProducer1.Id(): {},
		},
		MapDataConsumerIdDataProducerId: map[string]string{},
	}

	suite.Equal(expectedDump.MapDataProducerIdDataConsumerIds, routerDump.MapDataProducerIdDataConsumerIds)
	suite.Equal(expectedDump.MapDataConsumerIdDataProducerId, routerDump.MapDataConsumerIdDataProducerId)

	transportDump, _ := suite.transport1.Dump()
	suite.Equal(suite.transport1.Id(), transportDump.Id)
	suite.Equal([]string{dataProducer1.Id()}, transportDump.DataProducerIds)
	suite.Empty(transportDump.DataConsumerIds)
}

func (suite *DataProducerTestingSuite) TestTransport2ProduceDataSucceeds() {
	onObserverNewDataProducer := NewMockFunc(suite.T())
	suite.transport2.Observer().Once("newdataproducer", onObserverNewDataProducer.Fn())

	dataProducer2, err := suite.transport2.ProduceData(DataProducerOptions{
		SctpStreamParameters: &SctpStreamParameters{StreamId: 777, MaxRetransmits: 3},
		Label:                "foo",
		Protocol:             "bar",
		AppData:              H{"foo": 1, "bar": "2"},
	})
	suite.NoError(err)
	onObserverNewDataProducer.ExpectCalledTimes(1)
	onObserverNewDataProducer.ExpectCalledWith(dataProducer2)
	suite.False(dataProducer2.Closed())
	suite.Equal(DataProducerType_Sctp, dataProducer2.Type())
	suite.NotNil(dataProducer2.SctpStreamParameters())
	suite.EqualValues(777, dataProducer2.SctpStreamParameters().StreamId)
	suite.False(*dataProducer2.SctpStreamParameters().Ordered)
	suite.Zero(dataProducer2.SctpStreamParameters().MaxPacketLifeTime)
	suite.EqualValues(3, dataProducer2.SctpStreamParameters().MaxRetransmits)
	suite.Equal("foo", dataProducer2.Label())
	suite.Equal("bar", dataProducer2.Protocol())
	suite.Equal(H{"foo": 1, "bar": "2"}, dataProducer2.AppData())

	routerDump, _ := suite.router.Dump()
	expectedDump := RouterDump{
		MapDataProducerIdDataConsumerIds: map[string][]string{
			dataProducer2.Id(): {},
		},
		MapDataConsumerIdDataProducerId: map[string]string{},
	}

	suite.Equal(expectedDump.MapDataProducerIdDataConsumerIds, routerDump.MapDataProducerIdDataConsumerIds)
	suite.Equal(expectedDump.MapDataConsumerIdDataProducerId, routerDump.MapDataConsumerIdDataProducerId)

	transportDump, _ := suite.transport2.Dump()
	suite.Equal(suite.transport2.Id(), transportDump.Id)
	suite.Equal([]string{dataProducer2.Id()}, transportDump.DataProducerIds)
	suite.Empty(transportDump.DataConsumerIds)
}

func (suite *DataProducerTestingSuite) TestProduceDataTypeError() {
	_, err := suite.transport1.ProduceData(DataProducerOptions{})
	suite.Error(err)
}

func (suite *DataProducerTestingSuite) TestProduceDataWithAlreadyUsedStreamIdWithError() {
	suite.transport1.ProduceData(DataProducerOptions{
		SctpStreamParameters: &SctpStreamParameters{
			StreamId: 666,
		},
	})
	_, err := suite.transport1.ProduceData(DataProducerOptions{
		SctpStreamParameters: &SctpStreamParameters{
			StreamId: 666,
		},
	})
	suite.Error(err)
}

func (suite *DataProducerTestingSuite) TestProduceDataWithOrderedAndMaxPacketLifeTimeRejectsWithTypeErrorError() {
	_, err := suite.transport1.ProduceData(DataProducerOptions{
		SctpStreamParameters: &SctpStreamParameters{
			StreamId:          999,
			Ordered:           Bool(true),
			MaxPacketLifeTime: 4000,
		},
	})
	suite.Error(err)
}

func (suite *DataProducerTestingSuite) TestDataProducerDumpSucceeds() {
	dataProducer1, err := suite.transport1.ProduceData(DataProducerOptions{
		SctpStreamParameters: &SctpStreamParameters{
			StreamId: 666,
		},
		Label:    "foo",
		Protocol: "bar",
	})
	suite.NoError(err)

	data, err := dataProducer1.Dump()
	suite.NoError(err)
	suite.Equal(dataProducer1.Id(), data.Id)
	suite.Equal("sctp", data.Type)
	suite.NotNil(dataProducer1.SctpStreamParameters())
	suite.Equal(dataProducer1.SctpStreamParameters().StreamId, data.SctpStreamParameters.StreamId)
	suite.True(*data.SctpStreamParameters.Ordered)
	suite.Zero(data.SctpStreamParameters.MaxPacketLifeTime)
	suite.Zero(data.SctpStreamParameters.MaxRetransmits)
	suite.Equal("foo", data.Label)
	suite.Equal("bar", data.Protocol)

	dataProducer2, _ := suite.transport2.ProduceData(DataProducerOptions{
		SctpStreamParameters: &SctpStreamParameters{
			StreamId:       777,
			MaxRetransmits: 3,
		},
		Label:    "foo",
		Protocol: "bar",
		AppData:  H{"foo": 1, "bar": "2"},
	})
	data, err = dataProducer2.Dump()
	suite.NoError(err)
	suite.Equal(dataProducer2.Id(), data.Id)
	suite.Equal("sctp", data.Type)
	suite.NotNil(dataProducer2.SctpStreamParameters())
	suite.Equal(dataProducer2.SctpStreamParameters().StreamId, data.SctpStreamParameters.StreamId)
	suite.False(*data.SctpStreamParameters.Ordered)
	suite.Zero(data.SctpStreamParameters.MaxPacketLifeTime)
	suite.EqualValues(3, data.SctpStreamParameters.MaxRetransmits)
	suite.Equal("foo", data.Label)
	suite.Equal("bar", data.Protocol)
}

func (suite *DataProducerTestingSuite) TestDataProducerCloseSucceeds() {
	onObserverClose := NewMockFunc(suite.T())
	dataProducer1, _ := suite.transport1.ProduceData(DataProducerOptions{
		SctpStreamParameters: &SctpStreamParameters{
			StreamId: 666,
		},
	})
	dataProducer1.Observer().Once("close", onObserverClose.Fn())
	dataProducer1.Close()

	onObserverClose.ExpectCalledTimes(1)
	suite.True(dataProducer1.Closed())

	routerDump, _ := suite.router.Dump()
	expectedDump := RouterDump{
		MapDataProducerIdDataConsumerIds: map[string][]string{},
		MapDataConsumerIdDataProducerId:  map[string]string{},
	}

	suite.Equal(expectedDump.MapDataProducerIdDataConsumerIds, routerDump.MapDataProducerIdDataConsumerIds)
	suite.Equal(expectedDump.MapDataConsumerIdDataProducerId, routerDump.MapDataConsumerIdDataProducerId)

	transportDump, _ := suite.transport1.Dump()
	suite.Equal(suite.transport1.Id(), transportDump.Id)
	suite.Empty(transportDump.ProducerIds)
	suite.Empty(transportDump.DataProducerIds)
}

func (suite *DataProducerTestingSuite) TestProducerMethodsRejectIfClosed() {
	dataProducer1, _ := suite.transport2.ProduceData(DataProducerOptions{
		SctpStreamParameters: &SctpStreamParameters{
			StreamId: 666,
		},
	})
	dataProducer1.Close()

	_, err := dataProducer1.Dump()
	suite.Error(err)

	_, err = dataProducer1.GetStats()
	suite.Error(err)
}

func (suite *DataProducerTestingSuite) TestDataProducerEmitsTransportcloseIfTransportIsClosed() {
	dataProducer1, _ := suite.transport2.ProduceData(DataProducerOptions{
		SctpStreamParameters: &SctpStreamParameters{
			StreamId: 666,
		},
	})

	onObserverClose := NewMockFunc(suite.T())
	dataProducer1.Observer().Once("close", onObserverClose.Fn())
	dataProducer1.Close()

	onObserverClose.ExpectCalledTimes(1)
	suite.True(dataProducer1.Closed())
}
