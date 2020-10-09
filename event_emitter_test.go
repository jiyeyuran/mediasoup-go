package mediasoup

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEventEmitter_AddListener(t *testing.T) {
	evName := "test"
	emitter := EventEmitter{}

	emitter.AddListener(evName, func() {})
	emitter.AddListener(evName, func() {})
	assert.Equal(t, 2, emitter.ListenerCount(evName))
}

func TestEventEmitter_Once(t *testing.T) {
	evName := "test"
	emitter := EventEmitter{}

	onceObserver := NewWaitFunc(t)
	emitter.Once(evName, onceObserver.Fn())

	wg := sync.WaitGroup{}

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go (func() {
			defer wg.Done()
			emitter.Emit(evName)
		})()
	}

	wg.Wait()

	onceObserver.Wait()

	assert.Equal(t, 1, onceObserver.CalledTimes())
	assert.Equal(t, 0, emitter.ListenerCount(evName))
}

func TestEventEmitter_Emit(t *testing.T) {
	evName := "test"
	emitter := EventEmitter{}

	onObserver := NewMockFunc(t)
	emitter.On(evName, onObserver.Fn())
	emitter.On(evName, func(i, j int) {})
	emitter.Emit(evName)
	emitter.Emit(evName, 1)
	emitter.Emit(evName, 1, 2)
	emitter.Emit(evName, 1, 2, 3)

	time.Sleep(time.Millisecond)

	assert.Equal(t, 4, onObserver.CalledTimes())
}

func TestEventEmitter_SafeEmit(t *testing.T) {
	evName := "test"
	emitter := EventEmitter{}

	called := false
	emitter.On(evName, func(int) { called = true })
	emitter.SafeEmit(evName, "1") // invalid argument, panic

	assert.False(t, called)
}

func TestEventEmitter_RemoveListener(t *testing.T) {
	evName := "test"
	emitter := EventEmitter{}

	onObserver := NewMockFunc(t)
	fn := onObserver.Fn()

	emitter.On(evName, fn)
	emitter.Off(evName, fn)

	assert.Equal(t, 0, onObserver.CalledTimes())
	assert.Equal(t, 0, emitter.ListenerCount(evName))
}

func TestEventEmitter_RemoveAllListeners(t *testing.T) {
	evName := "test"
	emitter := EventEmitter{}

	onObserver := NewMockFunc(t)
	fn := onObserver.Fn()

	emitter.On(evName, fn)
	emitter.RemoveAllListeners(evName)
	emitter.Emit(evName)

	assert.Equal(t, 0, onObserver.CalledTimes())
	assert.Equal(t, 0, emitter.ListenerCount(evName))
}
