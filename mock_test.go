package mediasoup

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
)

// notificationTimeout is how long to wait for an event that the worker delivers
// asynchronously. Generous on purpose: it is only ever reached when something is
// actually broken, so it costs nothing on a healthy run.
const notificationTimeout = 5 * time.Second

// waitFor makes an expectation signal when it runs, returning a function that
// blocks until it has run times times.
//
// Worker notifications are dispatched on a goroutine of their own, so an event
// has not necessarily reached its listeners by the time the call that triggers it
// returns. Sleeping instead of waiting makes a test that passes on an idle laptop
// fail on a loaded CI runner under -race. Object state is no substitute either:
// close listeners run *after* the object is marked closed, so an object can read
// as closed while its listeners have yet to run.
func waitFor(call *mock.Call, times int) (wait func(t *testing.T, what string)) {
	var (
		remaining = int64(times)
		done      = make(chan struct{})
	)

	call.Run(func(mock.Arguments) {
		if atomic.AddInt64(&remaining, -1) == 0 {
			close(done)
		}
	})

	return func(t *testing.T, what string) {
		t.Helper()

		select {
		case <-done:
		case <-time.After(notificationTimeout):
			t.Fatalf("timed out waiting for %s", what)
		}
	}
}

// waitUntil polls condition from the caller's goroutine, unlike
// assert.Eventually, which runs it on one of its own. Use it when the condition
// observes goroutine counts, which that extra goroutine would distort.
func waitUntil(t *testing.T, condition func() bool, timeout time.Duration, what string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

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
