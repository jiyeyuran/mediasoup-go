# Changelog

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
