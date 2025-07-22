# Changelog

### NEXT
- Remove H265 codec and deprecated frame-marking RTP extension

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
