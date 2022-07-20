package mediasoup

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type MockFunc struct {
	require    *require.Assertions
	notifyChan chan []interface{}
	results    [][]interface{}
	timeout    time.Duration
}

func NewMockFunc(t *testing.T) *MockFunc {
	return &MockFunc{
		require:    require.New(t),
		notifyChan: make(chan []interface{}, 100),
		timeout:    50 * time.Millisecond,
	}
}

func (w *MockFunc) WithTimeout(timeout time.Duration) *MockFunc {
	w.timeout = timeout
	return w
}

func (w *MockFunc) Fn() func(...interface{}) {
	w.Reset()

	return func(args ...interface{}) {
		w.notifyChan <- args
	}
}

func (w *MockFunc) ExpectCalledWith(args ...interface{}) {
	w.wait()

	if len(w.results) == 0 {
		w.require.FailNow("fn is not called")
		return
	}

	last := w.results[len(w.results)-1]

	if len(args) != len(last) {
		w.require.FailNow("fn is called, but the number of arguments is not the same")
		return
	}
	for i, arg := range args {
		w.require.EqualValues(arg, last[i])
	}
}

func (w *MockFunc) ExpectCalled(msgAndArgs ...interface{}) {
	w.require.NotZero(w.CalledTimes(), msgAndArgs...)
}

func (w *MockFunc) ExpectCalledTimes(called int, msgAndArgs ...interface{}) {
	w.require.Equal(called, w.CalledTimes(), msgAndArgs...)
}

func (w *MockFunc) CalledTimes() int {
	w.wait()
	return len(w.results)
}

func (w *MockFunc) Reset() {
	w.notifyChan = make(chan []interface{}, 100)
	w.results = nil
}

func (w *MockFunc) wait() {
	// results are already collected
	if len(w.results) > 0 {
		return
	}

	// collect results within timeout
	timer := time.NewTimer(w.timeout)
	defer timer.Stop()

	for {
		select {
		case result := <-w.notifyChan:
			w.results = append(w.results, result)
		case <-timer.C:
			return
		}
	}
}
