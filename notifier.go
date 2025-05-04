package mediasoup

import "sync"

type baseNotifier struct {
	mu       sync.RWMutex
	handlers []func()
}

func (n *baseNotifier) OnClose(handler func()) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.handlers = append(n.handlers, handler)
}

func (n *baseNotifier) notifyClosed() {
	n.mu.RLock()
	handlers := n.handlers
	n.mu.RUnlock()

	for _, handler := range handlers {
		handler()
	}
}
