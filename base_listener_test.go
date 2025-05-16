package mediasoup

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloseNotifier(t *testing.T) {
	notifier := baseListener{}
	called := 0
	notifier.OnClose(func() {
		called++
	})

	notifier.notifyClosed()
	notifier.notifyClosed()

	assert.Equal(t, 2, called)

	notifier.OnClose(func() {
		called++
	})

	notifier.notifyClosed()
	assert.Equal(t, 4, called)
}

func TestSync(t *testing.T) {
	m := sync.Map{}
	m.Store("key", "value")
	m.Store("key2", "value")
	m.Store("key3", "value")
	m.Store("aaa", "value")

	m.Range(func(key, value any) bool {
		t.Log(key, value)
		return true
	})

}
