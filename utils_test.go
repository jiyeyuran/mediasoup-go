package mediasoup

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClearSyncMap(t *testing.T) {
	m := sync.Map{}
	for i := 0; i < 100000; i++ {
		m.Store(i, i)
	}
	clearSyncMap(&m)

	size := 0
	m.Range(func(key, value any) bool {
		size++
		return true
	})
	assert.Equal(t, 0, size)
}

func TestClone(t *testing.T) {
	assert.Equal(t, supportedRtpCapabilities, clone(supportedRtpCapabilities))
}

func BenchmarkClone(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = clone(supportedRtpCapabilities)
	}
}

func TestRef(t *testing.T) {
	val := 1
	assert.Same(t, &val, &(*(&val))) //nolint
	assert.NotSame(t, &val, ref(val))
}
