package mediasoup

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockedHandler struct {
	mock.Mock
}

func (m *MockedHandler) OnNewWebRtcServer(ctx context.Context, arg1 *WebRtcServer) {
	m.Called(ctx, arg1)
}

func (m *MockedHandler) OnNewRouter(ctx context.Context, arg1 *Router) {
	m.Called(ctx, arg1)
}

func (m *MockedHandler) OnNewRtpObserver(ctx context.Context, arg1 *RtpObserver) {
	m.Called(ctx, arg1)
}

func (m *MockedHandler) OnNewTransport(ctx context.Context, arg1 *Transport) {
	m.Called(ctx, arg1)
}

func (m *MockedHandler) OnNewProducer(ctx context.Context, arg1 *Producer) {
	m.Called(ctx, arg1)
}

func (m *MockedHandler) OnNewConsumer(ctx context.Context, arg1 *Consumer) {
	m.Called(ctx, arg1)
}

func (m *MockedHandler) OnNewDataProducer(ctx context.Context, arg1 *DataProducer) {
	m.Called(ctx, arg1)
}

func (m *MockedHandler) OnNewDataConsumer(ctx context.Context, arg1 *DataConsumer) {
	m.Called(ctx, arg1)
}

func (m *MockedHandler) OnClose(ctx context.Context) {
	m.Called(ctx)
}

func (m *MockedHandler) OnProducerScore(arg1 []ProducerScore) {
	m.Called(arg1)
}

func (m *MockedHandler) OnProducerVideoOrientation(arg1 ProducerVideoOrientation) {
	m.Called(arg1)
}

func (m *MockedHandler) OnProducerEventTrace(arg1 ProducerTraceEventData) {
	m.Called(arg1)
}

func (m *MockedHandler) OnProducerClose(ctx context.Context) {
	m.Called(ctx)
}

func (m *MockedHandler) OnProducerPause(ctx context.Context) {
	m.Called(ctx)
}

func (m *MockedHandler) OnProducerResume(ctx context.Context) {
	m.Called(ctx)
}

func (m *MockedHandler) OnDataProducerClose(ctx context.Context) {
	m.Called(ctx)
}

func (m *MockedHandler) OnConsumeScore(arg1 ConsumerScore) {
	m.Called(arg1)
}

func (m *MockedHandler) OnDominantSpeaker(arg1 AudioLevelObserverDominantSpeaker) {
	m.Called(arg1)
}

func (m *MockedHandler) OnVolume(arg1 []AudioLevelObserverVolume) {
	m.Called(arg1)
}

func (m *MockedHandler) OnSilence() {
	m.Called()
}
