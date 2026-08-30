package mediasoup

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListenerListAddRemove(t *testing.T) {
	var l listenerList[string]

	assert.Empty(t, l.list())

	a := l.add("a")
	b := l.add("b")
	c := l.add("c")
	assert.Equal(t, []string{"a", "b", "c"}, l.list())

	l.remove(b)
	assert.Equal(t, []string{"a", "c"}, l.list())

	// Removing the same id again is a no-op, as is removing an unknown id.
	l.remove(b)
	l.remove(12345)
	assert.Equal(t, []string{"a", "c"}, l.list())

	// Ids are not reused, so a listener added after a removal gets its own id.
	d := l.add("d")
	assert.NotEqual(t, b, d)
	assert.Equal(t, []string{"a", "c", "d"}, l.list())

	l.remove(a)
	l.remove(c)
	l.remove(d)
	assert.Empty(t, l.list())
}

// A notifier takes list() under a read lock and then invokes the listeners with
// the lock released, so a concurrent add or remove must not modify the slice it
// is iterating over.
func TestListenerListDoesNotMutatePublishedSlice(t *testing.T) {
	var l listenerList[string]

	first := l.add("a")
	l.add("b")

	published := l.list()

	l.add("c")
	l.remove(first)

	assert.Equal(t, []string{"a", "b"}, published)
	assert.Equal(t, []string{"b", "c"}, l.list())
}

func TestAddListenerConcurrent(t *testing.T) {
	var (
		mu sync.RWMutex
		l  listenerList[func()]
	)

	const goroutines = 50

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			remove := addListener(&mu, &l, func() {})
			// Read the list the way a notifier would, concurrently with the
			// other goroutines registering and removing.
			mu.RLock()
			for _, listener := range l.list() {
				listener()
			}
			mu.RUnlock()
			remove()
			// Removing twice must stay harmless under concurrency.
			remove()
		}()
	}
	wg.Wait()

	mu.RLock()
	defer mu.RUnlock()
	assert.Empty(t, l.list())
}

func TestOnCloseRemoveListener(t *testing.T) {
	worker := newTestWorker()
	router, err := worker.CreateRouter(&RouterOptions{})
	assert.NoError(t, err)

	var calls int
	removeListener := router.OnClose(func(ctx context.Context) { calls++ })
	router.OnClose(func(ctx context.Context) { calls += 10 })

	removeListener()
	assert.NoError(t, router.Close())

	// Only the listener that was not removed must have run.
	assert.Equal(t, 10, calls)

	// Removing after close must not panic.
	removeListener()

	worker.Close()
}
