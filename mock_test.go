package mediasoup

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

type MockFunc struct {
	require *require.Assertions
	wg      *sync.WaitGroup
	called  int32
	args    atomic.Value
}

func NewMockFunc(t *testing.T) *MockFunc {
	return &MockFunc{
		require: require.New(t),
		wg:      &sync.WaitGroup{},
	}
}

func (w *MockFunc) Fn() func(args ...interface{}) {
	w.wg.Add(1)

	return func(args ...interface{}) {
		w.args.Store(args)
		atomic.AddInt32(&w.called, 1)
		w.wg.Done()
	}
}

func (w *MockFunc) ExpectCalledWith(args ...interface{}) {
	w.wg.Wait()
	w.require.Equal(w.args.Load(), args)
}

func (w *MockFunc) ExpectCalledTimes(called int32) {
	w.wg.Wait()
	w.require.Equal(called, w.called)
}

func (w *MockFunc) Reset() {
	w.args.Store(nil)
	atomic.StoreInt32(&w.called, 0)
	w.wg = &sync.WaitGroup{}
}
