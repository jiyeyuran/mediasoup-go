package netstring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNetstring(t *testing.T) {
	testBytes := []byte("we are test string")
	rawBytes := Encode(testBytes)

	decoder := NewDecoder()
	decoder.Feed(rawBytes)

	select {
	case result := <-decoder.Result():
		assert.Equal(t, testBytes, result)
	default:
		assert.Fail(t, "no result")
	}
}

func BenchmarkEncode(b *testing.B) {
	testBytes := []byte("we are test string")
	for i := 0; i < b.N; i++ {
		Encode(testBytes)
	}
}

func BenchmarkDecode(b *testing.B) {
	testBytes := []byte("we are test string")
	rawBytes := Encode(testBytes)

	decoder := NewDecoder()

	for i := 0; i < b.N; i++ {
		decoder.Feed(rawBytes)
		<-decoder.Result()
	}
}
