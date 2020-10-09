package mediasoup

type Router struct {
	EventEmitter
}

func (r *Router) workerClosed() {
	return
}
