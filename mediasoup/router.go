package mediasoup

type Router struct {
	EventEmitter
	internal Internal
	data     interface{}
	channel  *Channel
}

func NewRouter(internal Internal, data interface{}, channel *Channel) *Router {
	return &Router{
		EventEmitter: NewEventEmitter(AppLogger()),
		internal:     internal,
		data:         data,
		channel:      channel,
	}
}

func (r *Router) WorkerClosed() {

}
