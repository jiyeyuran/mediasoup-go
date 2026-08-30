package mediasoup

import (
	"context"
	"slices"
	"sync"
)

// listenerList stores event listeners in registration order and lets individual
// listeners be removed again.
//
// Mutations never edit the published slices in place, they replace them. That
// lets a notifier take list() under a read lock, release the lock, and then
// invoke the listeners without blocking registration and without racing against
// a concurrent add or remove.
//
// A listenerList is not safe for concurrent use on its own; callers must hold
// the owning entity's lock.
type listenerList[T any] struct {
	nextID uint64
	ids    []uint64
	funcs  []T
}

// add registers fn and returns the id needed to remove it again.
func (l *listenerList[T]) add(fn T) uint64 {
	l.nextID++
	// Clip forces append to allocate, leaving any slice already handed out by
	// list() untouched.
	l.ids = append(slices.Clip(l.ids), l.nextID)
	l.funcs = append(slices.Clip(l.funcs), fn)
	return l.nextID
}

// remove drops the listener registered under id. Unknown ids are ignored, so
// removing twice or removing after the owner is closed is harmless.
func (l *listenerList[T]) remove(id uint64) {
	i := slices.Index(l.ids, id)
	if i < 0 {
		return
	}
	l.ids = slices.Delete(slices.Clone(l.ids), i, i+1)
	l.funcs = slices.Delete(slices.Clone(l.funcs), i, i+1)
}

// list returns the registered listeners in registration order. The result
// aliases internal storage and must not be modified.
func (l *listenerList[T]) list() []T {
	return l.funcs
}

// addListener registers fn on l and returns a function removing it again. The
// returned function is safe to call concurrently, more than once, and after the
// owning entity is closed.
func addListener[T any](mu *sync.RWMutex, l *listenerList[T], fn T) func() {
	mu.Lock()
	defer mu.Unlock()

	id := l.add(fn)

	return func() {
		mu.Lock()
		defer mu.Unlock()

		l.remove(id)
	}
}

type baseListener struct {
	mu             sync.RWMutex
	closeListeners listenerList[func(ctx context.Context)]
}

// OnClose adds a listener on the "close" event. Call the returned function to
// remove the listener again.
func (l *baseListener) OnClose(listener func(ctx context.Context)) (removeListener func()) {
	return addListener(&l.mu, &l.closeListeners, listener)
}

func (l *baseListener) notifyClosed(ctx context.Context) {
	l.mu.RLock()
	closeListeners := l.closeListeners.list()
	l.mu.RUnlock()

	for _, listener := range closeListeners {
		listener(ctx)
	}
}
