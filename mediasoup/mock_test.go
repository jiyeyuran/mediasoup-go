package mediasoup

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type MockFunc struct {
	assertions *assert.Assertions
	times      int
	argv       []interface{}
}

func NewMockFunc(t *testing.T) *MockFunc {
	return &MockFunc{
		assertions: assert.New(t),
	}
}

func (w *MockFunc) Fn() func(argv ...interface{}) {
	// reset
	w.times, w.argv = 0, nil

	return func(argv ...interface{}) {
		w.argv = argv
		w.times++
	}
}

func (w *MockFunc) ExpectCalledWith(argv ...interface{}) {
	w.assertions.Equal(argv, w.argv)
}

func (w *MockFunc) ExpectCalledTimes(times int) {
	w.assertions.Equal(times, w.times)
}

type WaitFunc struct {
	*MockFunc
	t       *testing.T
	closeCh chan struct{}
}

func NewWaitFunc(t *testing.T) *WaitFunc {
	return &WaitFunc{
		t:        t,
		MockFunc: NewMockFunc(t),
		closeCh:  make(chan struct{}),
	}
}

func (w *WaitFunc) Fn() func(argv ...interface{}) {
	// reset
	w.closeCh = make(chan struct{})
	w.MockFunc = NewMockFunc(w.t)

	return func(argv ...interface{}) {
		w.MockFunc.Fn()(argv)
		close(w.closeCh)
	}
}

func (w *WaitFunc) Wait() {
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()

	select {
	case <-w.closeCh:
	case <-timer.C:
		w.t.Fatal("timeout")
	}
	return
}
