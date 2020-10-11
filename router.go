package mediasoup

type RouterOptions struct {
	/**
	 * Router media codecs.
	 */
	MediaCodecs []RtpCodecCapability `json:"mediaCodecs,omitempty"`

	/**
	 * Custom application data.
	 */
	AppData interface{} `json:"appData,omitempty"`
}

type routerData struct {
	RtpCapabilities RtpCapabilities
}

type routerOptions struct {
	internal       internalData
	data           routerData
	channel        *Channel
	payloadChannel *PayloadChannel
	appData        interface{}
}

type Router struct {
	IEventEmitter
}

func newRouter(options routerOptions) *Router {
	return &Router{
		IEventEmitter: NewEventEmitter(),
	}
}

func (r *Router) workerClosed() {
	return
}
