package mediasoup

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type MockFunc struct {
	require *require.Assertions
	called  int32
	args    interface{}
	waited  bool
}

func NewMockFunc(t *testing.T) *MockFunc {
	return &MockFunc{
		require: require.New(t),
	}
}

func (w *MockFunc) Fn() func(...interface{}) {
	w.Reset()

	return func(args ...interface{}) {
		w.args = args
		w.called++
	}
}

func (w *MockFunc) ExpectCalledWith(args ...interface{}) {
	w.wait()
	w.require.Equal(w.args, args)
}

func (w *MockFunc) ExpectCalled() {
	w.wait()
	w.require.NotZero(w.called)
}

func (w *MockFunc) ExpectCalledTimes(called int32) {
	w.wait()
	w.require.Equal(called, w.called)
}

func (w *MockFunc) Reset() {
	w.args = nil
	w.called = 0
	w.waited = false
}

func (w *MockFunc) wait() {
	if !w.waited {
		wait(time.Millisecond)
		w.waited = true
	}
}

func wait(d time.Duration) {
	time.Sleep(d)
}
