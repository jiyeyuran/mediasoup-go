package mediasoup

import (
	"github.com/stretchr/testify/mock"
)

type MockedHandler struct {
	mock.Mock
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
