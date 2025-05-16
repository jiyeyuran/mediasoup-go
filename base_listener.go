package mediasoup

import "sync"

type baseListener struct {
	mu       sync.RWMutex
	handlers []func()
}

func (l *baseListener) OnClose(handler func()) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.handlers = append(l.handlers, handler)
}

func (l *baseListener) notifyClosed() {
	l.mu.RLock()
	handlers := l.handlers
	l.mu.RUnlock()

	for _, handler := range handlers {
		handler()
	}
}
