/*
Package mediasoup is a Go port of the mediasoup server-side SFU library.

Media is never handled in Go. Every call in this package is marshalled to a
mediasoup-worker subprocess over a pair of pipes, and that C++ worker does the
RTP forwarding. This package owns the subprocess, the FlatBuffers protocol on
the wire, and the object graph mirroring it.

# The worker binary

NewWorker takes a path to a prebuilt mediasoup-worker binary, which is not
shipped with this module. Download the one matching your mediasoup-go release
from https://github.com/versatica/mediasoup/releases; the README has the
compatibility table. A mismatched worker will typically fail while parsing a
FlatBuffers message rather than at startup, because the protocol changes between
mediasoup releases.

The worker communicates over extra file descriptors, so Windows is not
supported. One worker uses one CPU core; to use more cores, run several workers
and connect their routers with Router.PipeToRouter.

# Object graph

	Worker                      one subprocess
	 ├── WebRtcServer           optional shared ICE/DTLS port
	 └── Router                 one conference or room
	      ├── Transport         a connection to one endpoint
	      │    ├── Producer     inbound media
	      │    ├── Consumer     outbound media
	      │    ├── DataProducer inbound SCTP
	      │    └── DataConsumer outbound SCTP
	      └── RtpObserver       audio level or active speaker detection

Unlike the Node.js and Rust ports, all transport flavours share one Transport
type. Router.CreateWebRtcTransport, CreatePlainTransport, CreatePipeTransport
and CreateDirectTransport all return *Transport, and Transport.Type reports
which one it is. Methods that do not apply to the type at hand return
ErrNotImplemented, and the flavour-specific state lives behind
Transport.Data.

# Lifetime

Closing an object closes everything below it. Closing a Worker closes its
routers and WebRTC servers, closing a Router closes its transports and
observers, and closing a Transport closes its producers and consumers. Each
level also reports why it went away: a Router closed because its worker went
down notifies OnWorkerClosed before OnClose, a Transport closed with its router
notifies OnRouterClosed, a Consumer whose Producer disappeared notifies
OnProducerClose.

Close is idempotent and safe to call on an already closed object. All types are
safe for concurrent use.

If the worker subprocess exits on its own, Worker.Died becomes true, Worker.Err
holds the reason, and OnDied fires before the routers are torn down. Close
returns as soon as shutdown has been requested, so wait for
Worker.SubprocessClosed or OnSubprocessClose if you need the process itself to
be gone.

# Events

Every event is an OnXxx method taking a callback, and each returns a function
that unregisters it again:

	removeListener := transport.OnIceStateChange(func(state mediasoup.IceState) {
		log.Println("ICE state:", state)
	})
	defer removeListener()

Listeners run synchronously, in registration order, on whichever goroutine
produced the event. For worker notifications such as OnIceStateChange or
OnMessage that is the goroutine reading the worker channel, and a listener
blocking there stalls every notification behind it, so hand off work that can
block. For events raised by your own call, such as the OnPause that Pause
triggers, it is the calling goroutine.

Ignoring the returned function is fine for a listener that should live as long
as the object it is attached to. It matters for listeners attached to a
long-lived Worker or Router from a short-lived request, which would otherwise
accumulate for the lifetime of the process.

# Contexts

Every method that talks to the worker has a Context variant carrying a deadline
for that request: Produce and ProduceContext, Close and CloseContext, and so
on. The plain form uses context.Background.

Cancelling a context abandons the response, it does not undo the request. A
Produce whose context expires may still have created a producer inside the
worker, which then has to be cleaned up by closing the transport.
*/
package mediasoup
