package mediasoup

import (
	"github.com/stretchr/testify/mock"
)

type MockedHandler struct {
	mock.Mock
}

func (m *MockedHandler) OnNewWebRtcServer(arg1 *WebRtcServer) {
	m.Called(arg1)
}

func (m *MockedHandler) OnNewRouter(arg1 *Router) {
	m.Called(arg1)
}

func (m *MockedHandler) OnNewRtpObserver(arg1 *RtpObserver) {
	m.Called(arg1)
}

func (m *MockedHandler) OnNewTransport(arg1 *Transport) {
	m.Called(arg1)
}

func (m *MockedHandler) OnNewProducer(arg1 *Producer) {
	m.Called(arg1)
}

func (m *MockedHandler) OnNewConsumer(arg1 *Consumer) {
	m.Called(arg1)
}

func (m *MockedHandler) OnNewDataProducer(arg1 *DataProducer) {
	m.Called(arg1)
}

func (m *MockedHandler) OnNewDataConsumer(arg1 *DataConsumer) {
	m.Called(arg1)
}

func (m *MockedHandler) OnClose() {
	m.Called()
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

func (m *MockedHandler) OnProducerClosed() {
	m.Called()
}

func (m *MockedHandler) OnProducerPause() {
	m.Called()
}

func (m *MockedHandler) OnProducerResume() {
	m.Called()
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
