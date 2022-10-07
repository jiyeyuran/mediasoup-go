package mediasoup

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventEmitter(t *testing.T) {
	emitter := NewEventEmitter()
	called := 0

	callback1 := func(i int) {
		called++
	}

	// test "On" and "Emit"
	emitter.On("event1", callback1)
	emitter.Emit("event1", 1)
	emitter.Emit("event1", 1, 2) // more arguments than expected
	emitter.Emit("event1")       // ignore argument

	assert.Equal(t, 3, called)

	called = 0

	// test "Off"
	emitter.Off("event1", callback1)
	emitter.Emit("event1")
	assert.Equal(t, 0, called)

	// test "Once"
	emitter.Once("event1", callback1)
	emitter.Emit("event1")
	emitter.Emit("event1")
	assert.Equal(t, 1, called)

	called = 0

	type Struct struct {
		A int
	}
	val := Struct{A: 1}

	emitter.On("event2", func(s *Struct) {
		called += s.A
	})
	emitter.On("event2", func(s Struct) {
		called += s.A
	})
	emitter.On("event2", func(s Struct) {
		called += s.A
		panic("ignored panic")
	})
	// emit struct, failed on pointer argument
	emitter.SafeEmit("event2", val)

	data, _ := json.Marshal(val)
	// emit json data, success on pointer argument
	emitter.SafeEmit("event2", data)

	assert.Equal(t, 2*3-1, called)

	emitter.RemoveAllListeners("event2")
	assert.Equal(t, 0, emitter.ListenerCount("event2"))

	emitter.On("event2", func() {})

	emitter.RemoveAllListeners()
	assert.Equal(t, 0, emitter.ListenerCount())
}
