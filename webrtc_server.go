package mediasoup

import (
	"sync"
	"sync/atomic"

	"github.com/go-logr/logr"
)

type WebRtcServerListenInfo struct {
	// Network protocol.
	Protocol TransportProtocol `json:"protocol,omitempty"`

	// Listening IPv4 or IPv6.
	Ip string `json:"ip,omitempty"`

	// Announced IPv4 or IPv6 (useful when running mediasoup behind NAT with private IP).
	AnnouncedIp string `json:"announcedIp,omitempty"`

	// Listening port.
	Port uint16 `json:"port,omitempty"`
}

type WebRtcServerOptions struct {
	// Listen infos.
	ListenInfos []WebRtcServerListenInfo

	// appData
	AppData interface{}
}

type webrtcServerParams struct {
	internal internalData
	channel  *Channel
	appData  interface{}
}

// WebRtcServer brings the ability to listen on a single UDP/TCP port to WebRtcTransports.
// Instead of passing listenIps to router.CreateWebRtcTransport() pass webRtcServer with an
// instance of a WebRtcServer so the new WebRTC transport will not listen on its own IP:port(s)
// but will have its network traffic handled by the WebRTC server instead.
//
// A WebRTC server exists within the context of a Worker, meaning that if your app launches N
// workers it also needs to create N WebRTC servers listening on different ports (to not collide).
//
// The WebRTC transport implementation of mediasoup is ICE Lite, meaning that it does not initiate
// ICE connections but expects ICE Binding Requests from endpoints.
//
// - @emits @close
// - @emits workerclose
type WebRtcServer struct {
	IEventEmitter
	logger           logr.Logger
	internal         internalData
	channel          *Channel
	closed           uint32
	appData          interface{}
	webRtcTransports sync.Map // string:*WebRtcTransport
	observer         IEventEmitter
}

func NewWebRtcServer(params webrtcServerParams) *WebRtcServer {
	logger := NewLogger("WebRtcServer")
	logger.V(1).Info("constructor()", "internal", params.internal)

	return &WebRtcServer{
		IEventEmitter: NewEventEmitter(),
		logger:        logger,
		internal:      params.internal,
		channel:       params.channel,
		appData:       params.appData,
		observer:      NewEventEmitter(),
	}
}

// Id returns router id.
func (s *WebRtcServer) Id() string {
	return s.internal.WebRtcServerId
}

// Closed returns whether the router is closed or not.
func (s *WebRtcServer) Closed() bool {
	return atomic.LoadUint32(&s.closed) > 0
}

// AppData returns App custom data.
func (s *WebRtcServer) AppData() interface{} {
	return s.appData
}

// Observer returns an EventEmitter object.
//
// - @emits close
// - @emits webrtctransporthandled - (transport *WebRtcTransport)
// - @emits webrtctransportunhandled - (transport *WebRtcTransport)
func (s *WebRtcServer) Observer() IEventEmitter {
	return s.observer
}

// webRtcTransportsForTesting is used for testing purposes.
func (s *WebRtcServer) webRtcTransportsForTesting() map[string]*WebRtcTransport {
	transports := make(map[string]*WebRtcTransport)

	s.webRtcTransports.Range(func(key, value interface{}) bool {
		transports[key.(string)] = value.(*WebRtcTransport)
		return true
	})

	return transports
}

// Close the webrtc server.
func (s *WebRtcServer) Close() {
	if !atomic.CompareAndSwapUint32(&s.closed, 0, 1) {
		return
	}
	s.logger.V(1).Info("close()")

	reqData := H{"webRtcServerId": s.internal.WebRtcServerId}

	s.channel.Request("worker.closeWebRtcServer", s.internal, reqData)

	s.webRtcTransports.Range(func(key, value interface{}) bool {
		webRtcTransport := value.(*WebRtcTransport)
		webRtcTransport.listenServerClosed()

		// Emit observer event.
		s.observer.SafeEmit("webrtctransportunhandled", webRtcTransport)
		return true
	})
	s.webRtcTransports = sync.Map{}

	s.Emit("@close")
	s.RemoveAllListeners()

	// Emit observer event.
	s.observer.SafeEmit("close")
	s.observer.RemoveAllListeners()
}

// workerClosed is called when worker was closed.
func (s *WebRtcServer) workerClosed() {
	if !atomic.CompareAndSwapUint32(&s.closed, 0, 1) {
		return
	}
	s.logger.V(1).Info("workerClosed()")

	// NOTE: No need to close WebRtcTransports since they are closed by their
	// respective Router parents.
	s.webRtcTransports = sync.Map{}

	s.Emit("workerclose")

	// Emit observer event.
	s.observer.SafeEmit("close")
}

// Dump returns WebRtcServer information.
func (s *WebRtcServer) Dump() (data WebRtcServerDump, err error) {
	s.logger.V(1).Info("dump()")
	err = s.channel.Request("webRtcServer.dump", s.internal).Unmarshal(&data)
	return
}

func (s *WebRtcServer) handleWebRtcTransport(webRtcTransport *WebRtcTransport) {
	s.webRtcTransports.Store(webRtcTransport.Id(), webRtcTransport)

	s.observer.SafeEmit("webrtctransporthandled", webRtcTransport)

	webRtcTransport.On("@close", func() {
		s.webRtcTransports.Delete(webRtcTransport.Id())
		// Emit observer event.
		s.observer.SafeEmit("webrtctransportunhandled", webRtcTransport)
	})
}
