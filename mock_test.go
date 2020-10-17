package mediasoup

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type MockFunc struct {
	assertions *assert.Assertions
	times      int
	argv       interface{}
}

func NewMockFunc(t *testing.T) *MockFunc {
	return &MockFunc{
		assertions: assert.New(t),
	}
}

func (w *MockFunc) Fn() func(argv interface{}) {
	// reset
	w.times, w.argv = 0, nil

	return func(argv interface{}) {
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

func (w *MockFunc) Wait() {
	Wait(time.Millisecond)
	return
}
