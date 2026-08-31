# Changelog

### 2.6.1

- `WorkerPool`: add the `Scheduler` interface and `SetScheduler()`, so which worker a
  router lands on is the application's decision. Round-robin stays the default and
  spreads rooms evenly, but it treats every room as equally expensive: `LeastLoaded`
  weighs the workers by a load function of your own, `Random` carries no shared
  state, and `SchedulerFunc` covers any other strategy. A scheduler only ever sees
  live workers, and is called without the pool lock held so it may consult them.
  `LeastLoaded` given no load function of its own weighs producers and consumers,
  which is what a worker's capacity is measured in. `GetResourceUsage()` is a
  round trip to the subprocess and does not belong on the path of every router
  creation
- `Router.PipeToRouter()`: when `KeepId` is left unset, two routers on the same
  worker now get a new producer id instead of failing. A `WorkerPool` does not
  tell the application which worker a router landed on, so the call no longer
  has to know
- `WorkerSettings.WebRtcListenInfos`: if set, a WebRtcServer is created with
  the worker. `Worker.WebRtcServer()` returns it. `CreateWebRtcTransport` with
  neither `ListenInfos` nor `WebRtcServer` uses that default. A `WorkerPool`
  increments a fixed listen port per worker unless `UDPReusePort` is set, and
  recreates the server on a replacement worker. `Router.Worker()` says
  which worker a router sits on
- `WorkerPool`: a worker that dies is replaced with a new empty one so later
  rooms can still use that core; `OnWorkerDied` / `OnWorkerReplaced` are how
  the application hears about it. Rooms on the dead worker are gone.

### 2.6.0

Close the remaining API gaps against the mediasoup Node.js binding, add the
  lifecycle, observability and multi-core pieces that had no equivalent here, and
  fix the eight defects found while auditing all of it. No worker protocol change,
  so mediasoup-worker **v3.26.0** is still what this requires.

- **Breaking change:** every `OnXxx()` method now returns a `removeListener func()`
  that unregisters the listener again. Existing code that ignores the return value
  keeps working; only code that stored an `OnXxx` method value needs updating.
  Without this there was no way to unsubscribe, so registering per-call listeners
  on a long-lived `Router` or `Worker` leaked the listener and everything its
  closure captured
- add `WorkerPool`, which runs a group of workers and hands out one per router,
  round-robin, skipping any that died or were closed. A worker is pinned to one CPU
  core, so using more than one core previously meant hand-rolling this
- add `WorkerSettings.OnChannelRequest` and `Worker.ChannelPendingRequests()`, so
  the cost of talking to the worker subprocess can be exported as metrics. Request
  latency, request errors and the pending request count were previously invisible
- `Worker`: add `Died()`, `SubprocessClosed()`, `OnDied()` and `OnSubprocessClose()`.
  Previously a crashed worker could only be noticed by polling `Err()`, and a
  worker killed by `Close()` was indistinguishable from one that died on its own
- `Transport`: add `SetMaxOutgoingBitrate()` and `SetMinOutgoingBitrate()` (return `ErrNotImplemented`
  on a direct transport, matching Node.js)
- `Router`: add the missing `AppData()` getter
- `DataConsumer`: add the missing `Subchannels()` getter, returning a copy of the current subscription
- fix(channel): `Close()` walked the pending-request map without holding the lock,
  while requests that give up delete their own entries under it. Concurrent map
  iteration and write panics rather than merely racing, so a worker going away with
  requests in flight could take the process down
- fix(dataConsumer): `SendText("")` used the empty *binary* payload type (57) instead of the empty
  *string* one (56), so the remote peer decoded an empty string as a binary message.
  `DataProducer.SendText()` was already correct
- fix(transport): the `PLAINTRANSPORT_RTCP_TUPLE` handler notified `OnTuple`
  listeners instead of `OnRtcpTuple` ones, so `OnRtcpTuple` never fired and
  `OnTuple` fired with an RTCP tuple
- fix(webrtcserver): `Close()` never emitted the close event, so `OnClose`
  listeners never ran and the worker kept a reference to every closed server
- fix(webrtcserver): `Closed()` stayed false after the worker went down
- fix(transport): `OnNewProducer` / `OnNewConsumer` / `OnNewDataProducer` / `OnNewDataConsumer` held a
  read lock while appending to the listener slice, which is a data race when listeners are registered
  concurrently
- fix(worker): `Err()` read `w.err` while the process-wait goroutine wrote it,
  and it no longer reports an error when `Close()` had to force kill the process
- fix(router): `cleanupAfterClosed()` deleted from `transports` while draining `rtpObservers`, leaving
  the observer entries behind
- docs: add package documentation covering the worker binary requirement, the
  object graph, close cascades, the event model and context semantics, plus
  runnable godoc examples
- test: wait for worker notifications instead of sleeping. Notifications are
  dispatched on a goroutine of their own, so the sleeps were a source of random CI
  failures

### 2.5.0

Sync with mediasoup v3.20.0~v3.26.0 changelog. Requires mediasoup-worker **v3.26.0**
  (earlier 3.20.x/3.23.x workers are not wire-compatible: FBS field ids in
  `Transport.Options` and `Transport.Dump` shifted).

- **Breaking change:** `WorkerSettings`: remove `UseBuiltInSctpStack` and `DisableLiburing`, the worker
  always uses the built-in SCTP stack and `io_uring` support was dropped
- **Breaking change:** `WorkerDump`: remove `Liburing`
- **Breaking change:** remove `SctpCapabilities` and `NumSctpStreams`, no longer needed
- **Breaking change:** `SctpParameters` changes from `{ Port, OS, MIS, MaxMessageSize }` to
  `{ Port, MaxSendMessageSize, MaxReceiveMessageSize, SendBufferSize, PerStreamSendQueueLimit,
  MaxReceiverWindowBufferSize, IsDataChannel }`
- **Breaking change:** `WebRtcTransportOptions`, `PlainTransportOptions` and `PipeTransportOptions`:
  remove `NumSctpStreams`, `MaxSctpMessageSize` and `SctpSendBufferSize` in favour of the embedded
  `SctpOptions`
- **Breaking change:** `DirectTransportOptions`: remove `MaxMessageSize`, add `MaxSendMessageSize` and
  `MaxReceiveMessageSize`
- **Breaking change:** `DataConsumer.Send()` and `DataConsumer.SendText()` return the current buffered
  amount
- **Breaking change:** `WebRtcServerDump`: remove `TupleHashes`
- `Transport`: add `SctpNegotiatedCapabilities()` getter and `OnSctpNegotiatedCapabilities()` listener
- `DataProducer.Send()`: add `DataProducerSendWithIgnoredSubchannel()` option
- `SctpOptions`: add `SctpPerStreamSendQueueLimit`, `SctpMaxReceiverWindowBufferSize` and
  `SctpDefaultStreamBufferedAmountLowThreshold`
- Add `ErrNotFound`, returned when the entity referenced by a request doesn't exist in the worker
- fix(ortc): don't reuse the given `RtpCapabilities` storage when filtering RTCP feedback, which
  corrupted them when consuming from several goroutines
- fix(router): data race on the error of the two pipe transports created by `PipeToRouter()`
- fix(worker): don't read `cmd.ProcessState` while `Wait()` is running; wait on a `waitDone` channel instead ([#83](https://github.com/jiyeyuran/mediasoup-go/pull/83))
- fix(transport): lock `t.mu` when `Connect` / `RestartIce` update transport data ([#83](https://github.com/jiyeyuran/mediasoup-go/pull/83))

### 2.4.1

- Worker: Add `UseBuiltInSctpStack` setting (defaults to `false`) to enable mediasoup built-in SCTP stack

### 2.4.0

- Convert WORKER_CLOSE into a notification

### 2.3.3

- fix: correct json field name from listenIps to listenInfos

### 2.3.2

- refactor: change ID prefix separators from hyphen to underscore
- fix(channel): add size check for received messages
- feat: add cross-component event listeners for cleanup

### 2.3.1

- feat: export ID prefix for all IDs
- fix: id prefix of Producer

### 2.3.0

- Add `jitter` in `Consumer` 'outbound-rtp' stats
- Fix RTCP packets lost in stats
- RtpParameters: add msid optional field
- AV1: Set DependencyDescriptor Header Extension to 'recvonly' but forward it between pipe transports
- Add custom 'urn:mediasoup:params:rtp-hdrext:packet-id' (mediasoup-packet-id) header extension
- router.PipeToRouter() can now connect two Routers in the same Worker if KeepId is set to false

### 2.2.0

- ListenInfo: Add ExposeInternalIp field

### 2.1.0

- Remove H265 codec and deprecated frame-marking RTP extension
- Remove H264-SVC codec
- `Router`: Add `UpdateMediaCodecs()` method to dynamically change Router's RTP capabilities
- add version support for mediasoup C++ subprocess

### 2.0.3

- feat: Add initial AV1 codec support

### 2.0.2

- feat: DataConsumer and DataProducer to use options for sending data

### 2.0.1

- feat: enhance DataProducer with send options for subchannels
- feat: add WorkerLogger to log information from worker process
- fix: synchronize pause and resume states for DataProducer and Producer

### 2.0.0

- FlatBuffers protocol support — Now compatible with mediasoup v3.14.0+.
- Context-aware APIs — Added context support for better traceability and logging.
- Type-safe callbacks — Callback functions are now strictly typed for safer and clearer code.
- Optimized message handling — Improved internal messaging performance, reduced goroutine usage, and ensured that event callbacks for the same object are processed sequentially.
