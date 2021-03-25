package netstring

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNetstring(t *testing.T) {
	testBytes := []byte("we are test string")
	n := BUFFER_SIZE

	wg := &sync.WaitGroup{}
	wg.Add(n)

	decoder := NewDecoder()

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			select {
			case result := <-decoder.Result():
				assert.Equal(t, string(testBytes), string(result))
			case <-time.NewTimer(time.Millisecond * 100).C:
				assert.Fail(t, "timeout")
			}
		}()
	}

	go func() {
		for i := 0; i < n; i++ {
			rawBytes := Encode(testBytes)
			decoder.Feed(rawBytes)
		}
	}()

	doneCh := make(chan struct{})

	go func() {
		select {
		case <-doneCh:
		case <-time.NewTimer(time.Second).C:
			assert.Fail(t, "wait group timeout")
		}
	}()

	wg.Wait()
	close(doneCh)
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
