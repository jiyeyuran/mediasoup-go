package mediasoup

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func createDataConsumer(transport *Transport, dataProucerId string, options ...func(*DataConsumerOptions)) *DataConsumer {
	option := &DataConsumerOptions{
		DataProducerId:    dataProucerId,
		MaxPacketLifeTime: 4000,
		AppData:           H{"baz": "LOL"},
	}
	for _, o := range options {
		o(option)
	}
	dataConsumer, err := transport.ConsumeData(option)
	if err != nil {
		panic(err)
	}
	return dataConsumer
}

func TestDataConsumerDump(t *testing.T) {
	transport := createWebRtcTransport(nil, func(o *WebRtcTransportOptions) { o.EnableSctp = true })
	dataProducer := createDataProducer(transport)
	dataConsumer := createDataConsumer(transport, dataProducer.Id())

	data, err := dataConsumer.Dump()
	assert.NoError(t, err)
	assert.Equal(t, dataConsumer.Id(), data.Id)
	assert.Equal(t, dataConsumer.DataProducerId(), data.DataProducerId)
	assert.Equal(t, DataConsumerSctp, data.Type)
	assert.NotNil(t, dataConsumer.SctpStreamParameters())
	assert.Equal(t, dataConsumer.SctpStreamParameters().StreamId, data.SctpStreamParameters.StreamId)
	assert.False(t, *data.SctpStreamParameters.Ordered)
	assert.EqualValues(t, 4000, *data.SctpStreamParameters.MaxPacketLifeTime)
	assert.Zero(t, data.SctpStreamParameters.MaxRetransmits)
	assert.Equal(t, "foo", data.Label)
	assert.Equal(t, "bar", data.Protocol)
}

func TestDirectTransportConsumeData(t *testing.T) {
	transport := createDirectTransport(nil)
	dataProducer := createDataProducer(transport)
	dataConsumer, err := transport.ConsumeData(&DataConsumerOptions{
		DataProducerId: dataProducer.Id(),
		AppData:        H{"hehe": "HEHE"},
	})
	assert.NoError(t, err)
	assert.Equal(t, dataProducer.Id(), dataConsumer.DataProducerId())
	assert.False(t, dataConsumer.Closed())
	assert.Equal(t, DataConsumerDirect, dataConsumer.Type())
	assert.Nil(t, dataConsumer.SctpStreamParameters())
	assert.Equal(t, "foo", dataConsumer.Label())
	assert.Equal(t, "bar", dataConsumer.Protocol())
	assert.Equal(t, H{"hehe": "HEHE"}, dataConsumer.AppData())

	transportDump, _ := transport.Dump()
	assert.Equal(t, transport.Id(), transportDump.Id)
	assert.Empty(t, transportDump.ProducerIds)
	assert.Equal(t, []string{dataConsumer.Id()}, transportDump.DataConsumerIds)
}

func TestDirectTransportDataConsumerDump(t *testing.T) {
	transport := createDirectTransport(nil)
	dataProducer := createDataProducer(transport)
	dataConsumer, err := transport.ConsumeData(&DataConsumerOptions{
		DataProducerId:    dataProducer.Id(),
		MaxPacketLifeTime: 4000,
		AppData:           H{"hehe": "HEHE"},
	})
	assert.NoError(t, err)

	data, err := dataConsumer.Dump()
	assert.NoError(t, err)
	assert.Equal(t, dataConsumer.Id(), data.Id)
	assert.Equal(t, dataConsumer.DataProducerId(), data.DataProducerId)
	assert.Equal(t, DataConsumerDirect, data.Type)
	assert.Nil(t, data.SctpStreamParameters)
	assert.Equal(t, "foo", data.Label)
	assert.Equal(t, "bar", data.Protocol)
}

func TestDataConsumerGetStats(t *testing.T) {
	transport := createDirectTransport(nil)
	dataProducer := createDataProducer(transport)
	dataConsumer, _ := transport.ConsumeData(&DataConsumerOptions{
		DataProducerId:    dataProducer.Id(),
		MaxPacketLifeTime: 4000,
		AppData:           H{"hehe": "HEHE"},
	})
	stats, err := dataConsumer.GetStats()
	assert.NoError(t, err)
	assert.Equal(t, &DataConsumerStat{
		Type:      "data-consumer",
		Timestamp: stats[0].Timestamp,
		Label:     dataConsumer.Label(),
		Protocol:  dataConsumer.Protocol(),
	}, stats[0])
}

func TestDataConsumerClose(t *testing.T) {
	t.Run("close normally", func(t *testing.T) {
		mymock := new(MockedHandler)
		defer mymock.AssertExpectations(t)

		mymock.On("OnClose", mock.IsType(context.Background())).Once()

		router := createRouter(nil)
		transport := createDirectTransport(router)
		dataProducer := createDataProducer(transport)
		dataConsumer, _ := transport.ConsumeData(&DataConsumerOptions{
			DataProducerId:    dataProducer.Id(),
			MaxPacketLifeTime: 4000,
			AppData:           H{"baz": "LOL"},
		})
		dataConsumer.OnClose(mymock.OnClose)

		err := dataConsumer.Close()
		assert.NoError(t, err)
		assert.True(t, dataConsumer.Closed())

		routerDump, _ := router.Dump()

		assert.False(t, slices.ContainsFunc(
			routerDump.MapDataProducerIdDataConsumerIds,
			func(kvs KeyValues[string, string]) bool {
				return kvs.Key == dataProducer.Id() && slices.Contains(kvs.Values, dataConsumer.Id())
			}),
		)
		assert.False(t, slices.ContainsFunc(
			routerDump.MapDataConsumerIdDataProducerId,
			func(kvs KeyValue[string, string]) bool {
				return kvs.Key == dataConsumer.Id()
			}),
		)

		transportDump, _ := transport.Dump()

		assert.Equal(t, transport.Id(), transportDump.Id)
		assert.Empty(t, transportDump.ProducerIds)
		assert.Empty(t, transportDump.DataConsumerIds)
	})

	t.Run("dataProducer closed", func(t *testing.T) {
		mymock := new(MockedHandler)
		defer mymock.AssertExpectations(t)

		ctx := context.TODO()

		mymock.On("OnClose", ctx).Once()
		mymock.On("OnDataProducerClose", ctx).Once()

		router := createRouter(nil)
		transport := createDirectTransport(router)
		dataProducer := createDataProducer(transport)
		dataConsumer, _ := transport.ConsumeData(&DataConsumerOptions{
			DataProducerId:    dataProducer.Id(),
			MaxPacketLifeTime: 4000,
			AppData:           H{"baz": "LOL"},
		})
		dataConsumer.OnClose(mymock.OnClose)
		dataConsumer.OnDataProducerClose(mymock.OnDataProducerClose)
		dataProducer.CloseContext(ctx)
		time.Sleep(time.Millisecond)
		assert.True(t, dataConsumer.Closed())
	})

	t.Run("transport closed", func(t *testing.T) {
		mymock := new(MockedHandler)
		defer mymock.AssertExpectations(t)

		mymock.On("OnClose", mock.IsType(context.Background())).Once()

		router := createRouter(nil)
		transport := createDirectTransport(router)
		dataProducer := createDataProducer(transport)
		dataConsumer, _ := transport.ConsumeData(&DataConsumerOptions{
			DataProducerId:    dataProducer.Id(),
			MaxPacketLifeTime: 4000,
			AppData:           H{"baz": "LOL"},
		})
		dataConsumer.OnClose(mymock.OnClose)
		transport.Close()
		assert.True(t, dataConsumer.Closed())
		time.Sleep(time.Millisecond)
	})

	t.Run("router closed", func(t *testing.T) {
		mymock := new(MockedHandler)
		defer mymock.AssertExpectations(t)

		mymock.On("OnClose", mock.IsType(context.Background())).Once()

		router := createRouter(nil)
		transport := createDirectTransport(router)
		dataProducer := createDataProducer(transport)
		dataConsumer, _ := transport.ConsumeData(&DataConsumerOptions{
			DataProducerId:    dataProducer.Id(),
			MaxPacketLifeTime: 4000,
			AppData:           H{"baz": "LOL"},
		})
		dataConsumer.OnClose(mymock.OnClose)
		router.Close()
		assert.True(t, dataConsumer.Closed())
	})
}
