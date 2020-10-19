package mediasoup

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEventEmitter_AddListener(t *testing.T) {
	evName := "test"
	emitter := NewEventEmitter()

	emitter.AddListener(evName, func() {})
	emitter.AddListener(evName, func() {})
	assert.Equal(t, 2, emitter.ListenerCount(evName))
}

func TestEventEmitter_Once(t *testing.T) {
	evName := "test"
	emitter := NewEventEmitter()

	onceObserver := NewMockFunc(t)
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

	onceObserver.ExpectCalledTimes(1)
	assert.Equal(t, 0, emitter.ListenerCount(evName))
}

func TestEventEmitter_Emit1(t *testing.T) {
	evName := "test"
	emitter := NewEventEmitter()

	onObserver := NewMockFunc(t)
	emitter.On(evName, onObserver.Fn())
	emitter.On(evName, func(i, j int) {})
	emitter.Emit(evName)
	emitter.Emit(evName, 1)
	emitter.Emit(evName, 1, 2)
	emitter.Emit(evName, 1, 2, 3)

	time.Sleep(time.Millisecond)

	onObserver.ExpectCalledTimes(4)
}

func TestEventEmitter_Emit2(t *testing.T) {
	evName := "test"
	emitter := NewEventEmitter()

	type struct1 struct {
		A int
	}

	s := struct1{A: 1}
	data, _ := json.Marshal(s)

	called := 0

	emitter.On(evName, func(s struct1) {
		called++
	})
	emitter.On(evName, func(s *struct1) {
		called++
	})
	emitter.On(evName, func(s []byte) {
		called++
	})
	emitter.On(evName, func(s json.RawMessage) {
		called++
	})
	emitter.Emit(evName, data)
	emitter.Emit(evName, json.RawMessage(data))

	time.Sleep(time.Millisecond)

	assert.Equal(t, 4*2, called)

	called = 0
	evName2 := "test2"
	ss := []struct1{{A: 1}}
	data, _ = json.Marshal(ss)

	emitter.On(evName2, func(s []struct1) {
		called++
	})
	emitter.On(evName2, func(s []byte) {
		called++
	})
	emitter.On(evName2, func(s json.RawMessage) {
		called++
	})
	emitter.Emit(evName2, data)
	emitter.Emit(evName2, json.RawMessage(data))

	time.Sleep(time.Millisecond)
	assert.Equal(t, 3*2, called)
}

func TestEventEmitter_Emit3(t *testing.T) {
	evName := "test"
	emitter := NewEventEmitter()
	observer := NewMockFunc(t)

	emitter.On(evName, func(args ...int) {
		assert.Equal(t, 1, args[0])
		assert.Equal(t, 2, args[1])
	})
	emitter.On(evName, observer.Fn())
	emitter.Emit(evName, 1, 2)

	observer.ExpectCalledWith(1, 2)
}

func TestEventEmitter_SafeEmit(t *testing.T) {
	evName := "test"
	emitter := NewEventEmitter()

	called := false
	emitter.On(evName, func(int) { called = true })
	emitter.SafeEmit(evName, "1") // invalid argument, panic

	assert.False(t, called)
}

func TestEventEmitter_RemoveListener(t *testing.T) {
	evName := "test"
	emitter := NewEventEmitter()

	onObserver := NewMockFunc(t)
	fn := onObserver.Fn()

	emitter.On(evName, fn)
	emitter.Off(evName, fn)

	onObserver.ExpectCalledTimes(0)

	assert.Equal(t, 0, emitter.ListenerCount(evName))
}

func TestEventEmitter_RemoveAllListeners(t *testing.T) {
	evName := "test"
	emitter := NewEventEmitter()

	onObserver := NewMockFunc(t)
	fn := onObserver.Fn()

	emitter.On(evName, fn)
	emitter.RemoveAllListeners(evName)
	emitter.Emit(evName)

	onObserver.ExpectCalledTimes(0)

	assert.Equal(t, 0, emitter.ListenerCount(evName))
}
