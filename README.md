# Mediasoup-Go

A Go library for [mediasoup](https://github.com/versatica/mediasoup) that enables WebRTC Selective Forwarding Unit (SFU) functionality without Node.js dependencies.

## Version compatibility

The following table shows which mediasoup versions are supported by each mediasoup-go release:

| mediasoup-go version | Supported mediasoup version |
| -------------------- | --------------------------- |
| v2.6.x               | v3.26.0                     |
| v2.5.x               | v3.26.0                     |
| v2.4.x               | v3.19.18~v3.19.22           |
| v2.3.x               | v3.19.14~v3.19.17           |
| v2.2.0               | v3.17.0                     |
| v2.0.0~v2.2.0        | v3.14.0~v3.17.0             |
| not supported        | v3.13.0~v3.14.0             |
| v1.x.x               | < v3.13.0                   |

Note: Make sure to download the prebuilt mediasoup worker that matches the version you are using. Be aware that future mediasoup releases may change the FlatBuffers (fbs) protocol, which can break compatibility with mediasoup-go — always use a worker version that matches the mediasoup-go release you are running or rebuild the worker accordingly.

## Features

- Full mediasoup v3 API support in Go
- Consistent API design with the original Node.js version
- Typed event listeners instead of string event names, each removable again
- Multi-core via `WorkerPool`, with `PipeTransport` to bridge routers across workers
- Worker channel request latency and pending-request count exposed for metrics
- Uses `Cmd.ExtraFiles` for worker communication (not compatible with Windows)

## Prerequisites

- Download the prebuilt mediasoup worker from [mediasoup releases](https://github.com/versatica/mediasoup/releases)
- Linux or macOS (Windows not supported)

## Installation

```go
import "github.com/jiyeyuran/mediasoup-go/v2"
```

## Documentation

- [Go API Documentation](https://pkg.go.dev/github.com/jiyeyuran/mediasoup-go/v2) — the package
  overview covers the worker binary requirement, the object graph, close cascades and the event
  model, and the examples there cover the usual SFU shape, worker death, multi-core and metrics
- [Official mediasoup Documentation](https://mediasoup.org/documentation/v3/mediasoup/api/)

## Example Usage

See [mediasoup-go-demo](https://github.com/jiyeyuran/mediasoup-go-demo) for a complete example application.

<details>
<summary>Click to see code example</summary>

```go
package main

import (
    "github.com/jiyeyuran/mediasoup-go/v2"
    // ... other imports
)

func main() {
    // Create worker
    worker, err := mediasoup.NewWorker("path/to/mediasoup-worker")
    if err != nil {
        panic(err)
    }

    // Create router
    router, err := worker.CreateRouter(&mediasoup.RouterOptions{
        // Configure media codecs
    })

    // Create WebRTC transport
    transport, err := router.CreateWebRtcTransport(&mediasoup.WebRtcTransportOptions{
        ListenInfos: []mediasoup.TransportListenInfo{
            {Ip: "0.0.0.0", AnnouncedAddress: "your.public.ip"},
        },
    })

    // Use the transport to produce/consume media
    // ...
}
```

</details>

## License

[ISC](/LICENSE)
