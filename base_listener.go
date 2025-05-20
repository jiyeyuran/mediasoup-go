package mediasoup

import (
	"context"
	"sync"
)

type baseListener struct {
	mu             sync.RWMutex
	closeListeners []func(ctx context.Context)
}

func (l *baseListener) OnClose(listener func(ctx context.Context)) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.closeListeners = append(l.closeListeners, listener)
}

func (l *baseListener) notifyClosed(ctx context.Context) {
	l.mu.RLock()
	closeListeners := l.closeListeners
	l.mu.RUnlock()

	for _, listener := range closeListeners {
		listener(ctx)
	}
}
