package mediasoup

import (
	"reflect"
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

type wrapOption func(*asyncResult)

func withWaitTimeout(d time.Duration) wrapOption {
	return func(w *asyncResult) {
		w.waitTimeout = d
	}
}

func asyncRun(fn interface{}, options ...wrapOption) *asyncResult {
	w := &asyncResult{
		doneCh:      make(chan []reflect.Value, 1),
		waitTimeout: 10 * time.Millisecond,
	}
	for _, o := range options {
		o(w)
	}

	go func() {
		fnVal := reflect.ValueOf(fn)
		w.doneCh <- fnVal.Call(nil)
	}()

	return w
}

type asyncResult struct {
	done        bool
	doneCh      chan []reflect.Value
	outs        []interface{}
	waitTimeout time.Duration
	waited      bool
}

func (w *asyncResult) Finished() bool {
	w.mayWait()
	return w.done
}

func (w *asyncResult) Out(i int) interface{} {
	w.mayWait()
	if len(w.outs)-1 < i {
		return nil
	}
	return w.outs[i]
}

func (w *asyncResult) mayWait() {
	if w.waited {
		return
	}

	var waitCh <-chan time.Time

	if w.waitTimeout > 0 {
		waitTimer := time.NewTimer(w.waitTimeout)
		waitCh = waitTimer.C

		defer waitTimer.Stop()
	}

	select {
	case outs := <-w.doneCh:
		w.done = true
		for _, out := range outs {
			w.outs = append(w.outs, out.Interface())
		}

	case <-waitCh:

	}

	w.waited = true
}
