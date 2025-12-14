# Changelog

### 2.3.0
- router.PipeToRouter() can now connect two Routers in the same Worker if KeepId is set to false
- Add custom 'urn:mediasoup:params:rtp-hdrext:packet-id' (mediasoup-packet-id) header extension

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
