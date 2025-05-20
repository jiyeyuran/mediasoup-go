package mediasoup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func createDataProducer(transport *Transport) *DataProducer {
	dp, err := transport.ProduceData(&DataProducerOptions{
		SctpStreamParameters: &SctpStreamParameters{
			StreamId:          12345,
			Ordered:           ref(false),
			MaxPacketLifeTime: ref[uint16](5000),
		},
		Label:    "foo",
		Protocol: "bar",
		AppData:  H{"foo": 1, "bar": "2"},
	})
	if err != nil {
		panic(err)
	}
	return dp
}

func TestDataProducerDumpSucceeds(t *testing.T) {
	transport := createWebRtcTransport(nil, func(o *WebRtcTransportOptions) { o.EnableSctp = true })
	dataProducer1, err := transport.ProduceData(&DataProducerOptions{
		SctpStreamParameters: &SctpStreamParameters{
			StreamId: 666,
		},
		Label:    "foo",
		Protocol: "bar",
	})
	assert.NoError(t, err)

	data, err := dataProducer1.Dump()
	assert.NoError(t, err)
	assert.Equal(t, dataProducer1.Id(), data.Id)
	assert.Equal(t, DataProducerSctp, data.Type)
	assert.NotNil(t, dataProducer1.SctpStreamParameters())
	assert.Equal(t, dataProducer1.SctpStreamParameters().StreamId, data.SctpStreamParameters.StreamId)
	assert.True(t, *data.SctpStreamParameters.Ordered)
	assert.Zero(t, data.SctpStreamParameters.MaxPacketLifeTime)
	assert.Zero(t, data.SctpStreamParameters.MaxRetransmits)
	assert.Equal(t, "foo", data.Label)
	assert.Equal(t, "bar", data.Protocol)

	transport = createPlainTransport(nil, func(o *PlainTransportOptions) { o.EnableSctp = true })
	dataProducer2, _ := transport.ProduceData(&DataProducerOptions{
		SctpStreamParameters: &SctpStreamParameters{
			StreamId:       777,
			MaxRetransmits: ref[uint16](3),
		},
		Label:    "foo",
		Protocol: "bar",
		AppData:  H{"foo": 1, "bar": "2"},
	})
	data, err = dataProducer2.Dump()
	assert.NoError(t, err)
	assert.Equal(t, dataProducer2.Id(), data.Id)
	assert.Equal(t, DataProducerSctp, data.Type)
	assert.NotNil(t, dataProducer2.SctpStreamParameters())
	assert.Equal(t, dataProducer2.SctpStreamParameters().StreamId, data.SctpStreamParameters.StreamId)
	assert.False(t, *data.SctpStreamParameters.Ordered)
	assert.Zero(t, data.SctpStreamParameters.MaxPacketLifeTime)
	assert.EqualValues(t, 3, *data.SctpStreamParameters.MaxRetransmits)
	assert.Equal(t, "foo", data.Label)
	assert.Equal(t, "bar", data.Protocol)
}

func TestDataProducerClose(t *testing.T) {
	t.Run("close normally", func(t *testing.T) {
		mymock := new(MockedHandler)
		defer mymock.AssertExpectations(t)

		mymock.On("OnClose", mock.IsType(context.Background())).Once()

		router := createRouter(nil)
		transport := createDirectTransport(router)
		dataProducer, _ := transport.ProduceData(&DataProducerOptions{
			SctpStreamParameters: &SctpStreamParameters{
				StreamId: 666,
			},
		})
		dataProducer.OnClose(mymock.OnClose)
		err := dataProducer.Close()
		assert.NoError(t, err)
		assert.True(t, dataProducer.Closed())

		routerDump, _ := router.Dump()
		expectedDump := RouterDump{
			MapDataProducerIdDataConsumerIds: []KeyValues[string, string]{},
			MapDataConsumerIdDataProducerId:  []KeyValue[string, string]{},
		}

		assert.Equal(t, expectedDump.MapDataProducerIdDataConsumerIds, routerDump.MapDataProducerIdDataConsumerIds)
		assert.Equal(t, expectedDump.MapDataConsumerIdDataProducerId, routerDump.MapDataConsumerIdDataProducerId)

		transportDump, _ := transport.Dump()
		assert.Equal(t, transport.Id(), transportDump.Id)
		assert.Empty(t, transportDump.ProducerIds)
		assert.Empty(t, transportDump.DataProducerIds)
	})

	t.Run("transport closed", func(t *testing.T) {
		mymock := new(MockedHandler)
		defer mymock.AssertExpectations(t)

		mymock.On("OnClose", mock.IsType(context.Background())).Once()

		router := createRouter(nil)
		transport := createDirectTransport(router)
		dataProducer, _ := transport.ProduceData(&DataProducerOptions{
			SctpStreamParameters: &SctpStreamParameters{
				StreamId: 666,
			},
		})
		dataProducer.OnClose(mymock.OnClose)
		transport.Close()
		assert.True(t, dataProducer.Closed())
	})

	t.Run("router closed", func(t *testing.T) {
		mymock := new(MockedHandler)
		defer mymock.AssertExpectations(t)

		mymock.On("OnClose", mock.IsType(context.Background())).Once()

		router := createRouter(nil)
		transport := createDirectTransport(router)
		dataProducer, _ := transport.ProduceData(&DataProducerOptions{
			SctpStreamParameters: &SctpStreamParameters{
				StreamId: 666,
			},
		})
		dataProducer.OnClose(mymock.OnClose)
		router.Close()
		assert.True(t, dataProducer.Closed())
	})
}
