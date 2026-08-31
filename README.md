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
- Multi-core via `WorkerPool`, with pluggable scheduling and `PipeTransport` to bridge routers across workers
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
<summary>Single worker</summary>

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

<details>
<summary>WorkerPool (multi-core)</summary>

A worker is pinned to one CPU core. `WorkerPool` starts one worker per core (or
as many as you ask for) and puts each new router on the next live worker.

Routers on different workers cannot forward media to each other. Put peers that
talk to each other on the same router; use `Router.PipeToRouter` when they cannot
share one. You do not have to know which worker each router is on: `PipeToRouter`
keeps the producer id across workers and generates a new one when they share one.

```go
package main

import (
    "context"
    "log"

    "github.com/jiyeyuran/mediasoup-go/v2"
)

func main() {
    // 0 means runtime.NumCPU(). WebRtcServer is created with each worker;
    // without UDPReusePort the port is incremented per worker (44444, 44445, …).
    pool, err := mediasoup.NewWorkerPool("/path/to/mediasoup-worker", 0, func(s *mediasoup.WorkerSettings) {
        s.WebRtcListenInfos = []*mediasoup.TransportListenInfo{
            {
                Protocol:         mediasoup.TransportProtocolUDP,
                Ip:               "0.0.0.0",
                AnnouncedAddress: "your.public.ip",
                Port:             44444,
            },
        }
    })
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()

    // A worker that dies (C++ abort) is replaced with an empty one so later
    // rooms can still use that core. The rooms it hosted are gone.
    pool.OnWorkerDied(func(ctx context.Context, worker *mediasoup.Worker, err error) {
        log.Printf("worker %d died: %v; tell its clients to renegotiate", worker.Pid(), err)
    })

    // Default is round-robin. LeastLoaded(nil) picks the worker carrying the
    // fewest producers and consumers; pass your own function to weigh rooms
    // by something the application already tracks.
    pool.SetScheduler(mediasoup.LeastLoaded(nil))

    router, err := pool.CreateRouter(&mediasoup.RouterOptions{
        // Configure media codecs
    })
    if err != nil {
        log.Fatal(err)
    }

    // ListenInfos and WebRtcServer can both be omitted: the worker's default
    // WebRtcServer is used.
    transport, err := router.CreateWebRtcTransport(&mediasoup.WebRtcTransportOptions{})
    if err != nil {
        log.Fatal(err)
    }

    _ = transport
}
```

</details>

## License

[ISC](/LICENSE)
