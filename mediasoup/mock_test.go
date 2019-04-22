package mediasoup

import (
	"testing"
	"time"
)

type MockFunc struct {
	times int
}

func NewMockFunc() *MockFunc {
	return &MockFunc{}
}

func (w *MockFunc) Fn() func(argv ...interface{}) {
	return func(argv ...interface{}) {
		w.times++
	}
}

func (w *MockFunc) CalledTimes() int {
	return w.times
}

type WaitFunc struct {
	*MockFunc
	t      *testing.T
	argsCh chan []interface{}
}

func NewWaitFunc(t *testing.T) *WaitFunc {
	return &WaitFunc{
		MockFunc: NewMockFunc(),
		t:        t,
		argsCh:   make(chan []interface{}),
	}
}

func (w *WaitFunc) Fn() func(argv ...interface{}) {
	return func(argv ...interface{}) {
		w.MockFunc.Fn()
		w.argsCh <- argv
	}
}

func (w *WaitFunc) Wait() []interface{} {
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()

	select {
	case argv := <-w.argsCh:
		return argv
	case <-timer.C:
		w.t.Fatal("timeout")
		return nil
	}
}
