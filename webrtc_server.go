package mediasoup

import (
	"context"
	"log/slog"
	"sync"

	FbsRequest "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Request"
	FbsWebRtcServer "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/WebRtcServer"
	FbsWorker "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Worker"
	"github.com/jiyeyuran/mediasoup-go/v2/internal/channel"
)

// WebRtcServer brings the ability to listen on a single UDP/TCP port to WebRtcTransports.
// Instead of passing listenIps to router.CreateWebRtcTransport() pass webRtcServer with an
// instance of a WebRtcServer so the new WebRTC transport will not listen on its own Ip:port(s)
// but will have its network traffic handled by the WebRTC server instead.
//
// A WebRTC server exists within the context of a Worker, meaning that if your app launches N
// workers it also needs to create N WebRTC servers listening on different ports (to not collide).
//
// The WebRTC transport implementation of mediasoup is ICE Lite, meaning that it does not initiate
// ICE connections but expects ICE Binding Requests from endpoints.
type WebRtcServer struct {
	baseListener
	id               string
	channel          *channel.Channel
	appData          H
	closed           bool
	webRtcTransports sync.Map
	logger           *slog.Logger
}

func NewWebRtcServer(worker *Worker, id string, appData H) *WebRtcServer {
	return &WebRtcServer{
		id:      id,
		channel: worker.channel,
		appData: appData,
		logger:  worker.logger.With("webRtcServerId", id),
	}
}

// Id returns webrtc server's id.
func (s *WebRtcServer) Id() string {
	return s.id
}

// AppData returns App custom data.
func (s *WebRtcServer) AppData() H {
	return s.appData
}

// Closed returns whether the webrtc server is closed.
func (s *WebRtcServer) Closed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.closed
}

// Close the webrtc server.
func (s *WebRtcServer) Close() error {
	return s.CloseContext(context.Background())
}

func (s *WebRtcServer) CloseContext(ctx context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.logger.DebugContext(ctx, "Close()")

	_, err := s.channel.Request(ctx, &FbsRequest.RequestT{
		Method: FbsRequest.MethodWORKER_WEBRTCSERVER_CLOSE,
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyWorker_CloseWebRtcServerRequest,
			Value: &FbsWorker.CloseWebRtcServerRequestT{
				WebRtcServerId: s.Id(),
			},
		},
	})
	if err != nil {
		s.mu.Unlock()
		return err
	}
	s.closed = true
	s.mu.Unlock()

	transports := []*Transport{}
	s.webRtcTransports.Range(func(key, value any) bool {
		s.webRtcTransports.Delete(key)
		transports = append(transports, value.(*Transport))
		return true
	})

	for _, transport := range transports {
		transport.listenServerClosed(ctx)
	}

	return err
}

// workerClosed is called when worker was closed.
func (s *WebRtcServer) workerClosed(ctx context.Context) {
	s.logger.DebugContext(ctx, "workerClosed()")
	// NOTE: No need to close WebRtcTransports since they are closed by their
	// respective Router parents.
	clearSyncMap(&s.webRtcTransports)
	s.notifyClosed(ctx)
}

// Dump returns WebRtcServer information.
func (s *WebRtcServer) Dump() (*WebRtcServerDump, error) {
	return s.DumpContext(context.Background())
}

func (s *WebRtcServer) DumpContext(ctx context.Context) (*WebRtcServerDump, error) {
	s.logger.DebugContext(ctx, "Dump()")

	val, err := s.channel.Request(ctx, &FbsRequest.RequestT{
		HandlerId: s.Id(),
		Method:    FbsRequest.MethodWEBRTCSERVER_DUMP,
	})
	if err != nil {
		return nil, err
	}
	resp := val.(*FbsWebRtcServer.DumpResponseT)

	return &WebRtcServerDump{
		Id: resp.Id,
		UdpSockets: collect(resp.UdpSockets, func(v *FbsWebRtcServer.IpPortT) IpPort {
			return IpPort{Ip: v.Ip, Port: v.Port}
		}),
		TcpServers: collect(resp.TcpServers, func(v *FbsWebRtcServer.IpPortT) IpPort {
			return IpPort{Ip: v.Ip, Port: v.Port}
		}),
		WebRtcTransportIds: resp.WebRtcTransportIds,
		LocalIceUsernameFragments: collect(resp.LocalIceUsernameFragments,
			func(v *FbsWebRtcServer.IceUserNameFragmentT) IceUserNameFragment {
				return IceUserNameFragment{
					LocalIceUsernameFragment: v.LocalIceUsernameFragment,
					WebRtcTransportId:        v.WebRtcTransportId,
				}
			}),
		TupleHashes: collect(resp.TupleHashes, func(v *FbsWebRtcServer.TupleHashT) TupleHash {
			return TupleHash{TupleHash: v.TupleHash,
				WebRtcTransportId: v.WebRtcTransportId,
			}
		}),
	}, nil
}

func (s *WebRtcServer) handleWebRtcTransport(transport *Transport) {
	s.webRtcTransports.Store(transport.Id(), transport)
	transport.OnClose(func(ctx context.Context) {
		s.webRtcTransports.Delete(transport.Id())
	})
}
