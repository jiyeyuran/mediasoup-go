# Changelog

### Unreleased

Close the remaining API gaps against the mediasoup Node.js binding and fix the
  correctness issues found while auditing that gap.

- **Breaking change:** every `OnXxx()` method now returns a `removeListener func()`
  that unregisters the listener again. Existing code that ignores the return value
  keeps working; only code that stored an `OnXxx` method value needs updating.
  Without this there was no way to unsubscribe, so registering per-call listeners
  on a long-lived `Router` or `Worker` leaked the listener and everything its
  closure captured
- `Worker`: add `Died()`, `SubprocessClosed()`, `OnDied()` and `OnSubprocessClose()`.
  Previously a crashed worker could only be noticed by polling `Err()`, and a
  worker killed by `Close()` was indistinguishable from one that died on its own
- fix(worker): `Err()` read `w.err` while the process-wait goroutine wrote it,
  and it no longer reports an error when `Close()` had to force kill the process
- fix(transport): the `PLAINTRANSPORT_RTCP_TUPLE` handler notified `OnTuple`
  listeners instead of `OnRtcpTuple` ones, so `OnRtcpTuple` never fired and
  `OnTuple` fired with an RTCP tuple
- `Transport`: add `SetMaxOutgoingBitrate()` and `SetMinOutgoingBitrate()` (return `ErrNotImplemented`
  on a direct transport, matching Node.js)
- `Router`: add the missing `AppData()` getter
- `DataConsumer`: add the missing `Subchannels()` getter, returning a copy of the current subscription
- fix(dataConsumer): `SendText("")` used the empty *binary* payload type (57) instead of the empty
  *string* one (56), so the remote peer decoded an empty string as a binary message.
  `DataProducer.SendText()` was already correct
- fix(transport): `OnNewProducer` / `OnNewConsumer` / `OnNewDataProducer` / `OnNewDataConsumer` held a
  read lock while appending to the listener slice, which is a data race when listeners are registered
  concurrently
- fix(router): `cleanupAfterClosed()` deleted from `transports` while draining `rtpObservers`, leaving
  the observer entries behind

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
