package mediasoup

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestEventEmitter(t *testing.T) {
	called := 0
	fn := func(i int) {
		called++
	}
	evName := "test"
	logger := logrus.WithField("type", "eventEmitter")
	emiter := NewEventEmitter(logger)

	emiter.AddListener(evName, fn)
	emiter.SafeEmit(evName)
	assert.Equal(t, called, 1)

	emiter.SafeEmit(evName, "argument type invalid")
	assert.Equal(t, called, 1)

	emiter.RemoveListener(evName, fn)
	emiter.SafeEmit(evName)
	assert.Equal(t, called, 1)
	assert.Equal(t, emiter.ListenerCount(evName), 0)
}
