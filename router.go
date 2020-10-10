package mediasoup

type RouterOptions struct {
	/**
	 * Router media codecs.
	 */
	MediaCodecs []RtpCodecCapability `json:"mediaCodecs,omitempty"`

	/**
	 * Custom application data.
	 */
	AppData H `json:"appData,omitempty"`
}

type routerData struct {
	RtpCapabilities RtpCapabilities
}

type routerOptions struct {
	internal       internalData
	data           routerData
	channel        *Channel
	payloadChannel *PayloadChannel
	appData        H
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
