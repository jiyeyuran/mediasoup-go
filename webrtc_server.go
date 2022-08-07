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
	data     interface{}
	channel  *Channel
	appData  interface{}
}

type WebRtcServer struct {
	IEventEmitter
	logger           logr.Logger
	internal         internalData
	channel          *Channel
	closed           uint32
	appData          interface{}
	webRtcTransports sync.Map // string:WebRtcTransport
	observer         IEventEmitter
}

func NewWebRtcServer(params webrtcServerParams) *WebRtcServer {
	logger := NewLogger("WebRtcServer")
	logger.V(1).Info("constructor()")

	return &WebRtcServer{
		IEventEmitter: NewEventEmitter(),
		logger:        logger,
		internal:      params.internal,
		channel:       params.channel,
		appData:       params.appData,
		observer:      NewEventEmitter(),
	}
}

// Router id
func (s *WebRtcServer) Id() string {
	return s.internal.WebRtcServerId
}

// Whether the Router is closed.
func (s *WebRtcServer) Closed() bool {
	return atomic.LoadUint32(&s.closed) > 0
}

// AppData returns App custom data.
func (s *WebRtcServer) AppData() interface{} {
	return s.appData
}

func (s *WebRtcServer) Observer() IEventEmitter {
	return s.observer
}

// Just for testing purposes.
func (s *WebRtcServer) webRtcTransportsForTesting() map[string]*WebRtcTransport {
	transports := make(map[string]*WebRtcTransport)

	s.webRtcTransports.Range(func(key, value interface{}) bool {
		transports[key.(string)] = value.(*WebRtcTransport)
		return true
	})

	return transports
}

func (s *WebRtcServer) Close() {
	if !atomic.CompareAndSwapUint32(&s.closed, 0, 1) {
		return
	}
	s.logger.V(1).Info("close()")

	s.channel.Request("webRtcServer.close", s.internal)

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

// Worker was closed.
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

// Dump WebRtcServer.
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
