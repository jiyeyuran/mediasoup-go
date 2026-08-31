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
supported.

# Using more than one core

A worker is a subprocess pinned to a single core, so a single worker caps out at
one core no matter how many rooms it hosts. WorkerPool starts a group of them and
hands out a worker per room, round-robin, skipping any that have died:

	pool, err := mediasoup.NewWorkerPool(binPath, 0) // 0 means runtime.NumCPU()
	if err != nil {
		return err
	}
	defer pool.Close()

	router, err := pool.CreateRouter(&mediasoup.RouterOptions{MediaCodecs: codecs})

Round-robin spreads rooms evenly but treats them all as equally expensive. When
they are not, WorkerPool.SetScheduler takes a Scheduler that decides which worker
a new router goes on: Random, LeastLoaded weighing the workers by a load function
of your own, or any strategy of your own through SchedulerFunc.

	pool.SetScheduler(mediasoup.LeastLoaded(func(worker *mediasoup.Worker) float64 {
		return float64(consumerCount(worker))
	}))

A scheduler is consulted on every CreateRouter call, so it has to be cheap. Count
what the application already tracks rather than asking the subprocess through
Worker.GetResourceUsage. LeastLoaded with no load function of its own weighs the
producers and consumers already on each worker.

Routers on different workers cannot forward media to each other directly, so put
endpoints that talk to each other on one router where possible, and bridge with
Router.PipeToRouter where not. PipeToRouter does not need to be told whether the
two routers share a worker: it keeps the producer id across workers and generates
a new one when they do not.

Set WorkerSettings.WebRtcListenInfos to create a WebRtcServer with each worker.
Worker.WebRtcServer() returns it. CreateWebRtcTransport with neither
ListenInfos nor WebRtcServer uses that default. Router.Worker says which
worker a router sits on. A fixed listen port without
UDPReusePort is incremented per worker so the binds do not collide.

A worker that dies is replaced with a new empty one so later CreateRouter calls
can still use that core. The rooms it hosted are gone; OnWorkerDied is where
those clients get told to renegotiate. Close does not replace.

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

# Observability

Everything this package does costs a round trip to the subprocess, and that cost
is otherwise invisible. WorkerSettings.OnChannelRequest reports each completed
request with its method, duration and error, which is enough for a latency
histogram and an error counter. Worker.ChannelPendingRequests is the matching
gauge: it climbing means the worker is falling behind, which turns into
ErrChannelRequestTimeout a few seconds later.

The loggers in WorkerSettings take a *slog.Logger, and every request is logged
with the context it was issued under, so a Context variant carrying trace
identifiers ties worker activity back to the request that caused it.

# Contexts

Every method that talks to the worker has a Context variant carrying a deadline
for that request: Produce and ProduceContext, Close and CloseContext, and so
on. The plain form uses context.Background.

Cancelling a context abandons the response, it does not undo the request. A
Produce whose context expires may still have created a producer inside the
worker, which then has to be cleaned up by closing the transport.
*/
package mediasoup
