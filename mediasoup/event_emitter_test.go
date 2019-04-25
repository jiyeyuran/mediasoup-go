package mediasoup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventEmitter_AddListener(t *testing.T) {
	evName := "test"
	logger := TypeLogger("eventEmitter")
	emitter := NewEventEmitter(logger)

	emitter.AddListener(evName, 1, "a", func() {}, func() {})
	assert.Equal(t, 2, emitter.ListenerCount(evName))
}

func TestEventEmitter_Once(t *testing.T) {
	evName := "test"
	logger := TypeLogger("eventEmitter")
	emitter := NewEventEmitter(logger)

	onceObserver := NewMockFunc(t)
	emitter.Once(evName, onceObserver.Fn())
	emitter.Emit(evName)
	emitter.Emit(evName)

	assert.Equal(t, 1, onceObserver.CalledTimes())
}

func TestEventEmitter_Emit(t *testing.T) {
	evName := "test"
	logger := TypeLogger("eventEmitter")
	emitter := NewEventEmitter(logger)

	onObserver := NewMockFunc(t)
	emitter.On(evName, onObserver.Fn())
	emitter.Emit(evName)
	emitter.Emit(evName)

	assert.Equal(t, 2, onObserver.CalledTimes())
}

func TestEventEmitter_SafeEmit(t *testing.T) {
	evName := "test"
	logger := TypeLogger("eventEmitter")
	emitter := NewEventEmitter(logger)

	called := false
	emitter.On(evName, func(int) { called = true })
	emitter.SafeEmit(evName, "1")

	assert.False(t, called)
}

func TestEventEmitter_RemoveListener(t *testing.T) {
	evName := "test"
	logger := TypeLogger("eventEmitter")
	emitter := NewEventEmitter(logger)

	onObserver := NewMockFunc(t)
	fn := onObserver.Fn()

	emitter.On(evName, fn)
	emitter.RemoveListener(evName, fn)
	emitter.Emit(evName)

	assert.Equal(t, 0, onObserver.CalledTimes())
	assert.Equal(t, 0, emitter.ListenerCount(evName))
}

func TestEventEmitter_RemoveAllListeners(t *testing.T) {
	evName := "test"
	logger := TypeLogger("eventEmitter")
	emitter := NewEventEmitter(logger)

	onObserver := NewMockFunc(t)
	fn := onObserver.Fn()

	emitter.On(evName, fn)
	emitter.RemoveAllListeners(evName)
	emitter.Emit(evName)

	assert.Equal(t, 0, onObserver.CalledTimes())
	assert.Equal(t, 0, emitter.ListenerCount(evName))
}
