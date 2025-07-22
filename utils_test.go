package mediasoup

import (
	"sync"
	"testing"

	"github.com/Masterminds/semver/v3"
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
	assert.NotSame(t, &val, ref(val))
}

func TestVersionSatisfies(t *testing.T) {
	t.Run("returns true if version satisfies constraint", func(t *testing.T) {
		v := semver.MustParse("3.10.0")
		assert.True(t, versionSatisfies(v, ">= 3.9.0"))
	})

	t.Run("returns false if version does not satisfy constraint", func(t *testing.T) {
		v := semver.MustParse("3.8.0")
		assert.False(t, versionSatisfies(v, ">= 3.9.0"))
	})

	t.Run("returns false for invalid constraint", func(t *testing.T) {
		v := semver.MustParse("3.10.0")
		assert.False(t, versionSatisfies(v, "invalid"))
	})

	t.Run("returns true for exact match", func(t *testing.T) {
		v := semver.MustParse("3.10.0")
		assert.True(t, versionSatisfies(v, "3.10.0"))
	})

	t.Run("returns true for tilde match", func(t *testing.T) {
		v := semver.MustParse("3.10.5")
		assert.True(t, versionSatisfies(v, "~3.10.0"))
	})

	t.Run("returns false for tilde mismatch", func(t *testing.T) {
		v := semver.MustParse("3.11.0")
		assert.False(t, versionSatisfies(v, "~3.10.0"))
	})

	t.Run("returns true for caret match", func(t *testing.T) {
		v := semver.MustParse("3.11.0")
		assert.True(t, versionSatisfies(v, "^3.10.0"))
	})

	t.Run("returns false for caret mismatch", func(t *testing.T) {
		v := semver.MustParse("4.0.0")
		assert.False(t, versionSatisfies(v, "^3.10.0"))
	})
}
