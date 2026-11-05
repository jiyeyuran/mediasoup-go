package mediasoup

import "sync"

// pipeTransportPairPromise is the in-flight or finished creation of the
// PipeTransport pair shared by two Routers. As in the Node implementation, it is
// registered in the maps of both Routers before the transports are created, so a
// concurrent reverse PipeToRouter call waits for it instead of creating a second
// pair, and a later serial reverse call finds the finished pair.
type pipeTransportPairPromise struct {
	done chan struct{}
	// creator is the Router that created the pair.
	creator *Router
	// pair is oriented for creator: pair[0] is the creator's local transport.
	pair [2]*Transport
	err  error
}

// pairFor returns the pair oriented for the given Router: [local, remote].
func (p *pipeTransportPairPromise) pairFor(r *Router) [2]*Transport {
	if p.creator == r {
		return p.pair
	}
	return [2]*Transport{p.pair[1], p.pair[0]}
}

var (
	// routerPipeMu guards mapRouterPipeTransports. A single mutex is used because
	// registering a pair always touches the maps of both Routers, and per-router
	// mutexes would allow ABBA deadlocks.
	routerPipeMu sync.Mutex
	// mapRouterPipeTransports maps a Router to its peers and the shared pair.
	mapRouterPipeTransports = map[*Router]map[*Router]*pipeTransportPairPromise{}
)

// getOrCreatePipeTransportPair returns the promise for the given pair of
// Routers, registering it on both of them when it does not exist yet. The second
// return value reports whether the caller must create the transports.
func getOrCreatePipeTransportPair(a, b *Router) (*pipeTransportPairPromise, bool) {
	routerPipeMu.Lock()
	defer routerPipeMu.Unlock()

	if peers := mapRouterPipeTransports[a]; peers != nil {
		if promise, ok := peers[b]; ok {
			return promise, false
		}
	}

	promise := &pipeTransportPairPromise{
		done:    make(chan struct{}),
		creator: a,
	}
	setPipeTransportPairLocked(a, b, promise)
	setPipeTransportPairLocked(b, a, promise)

	return promise, true
}

func setPipeTransportPairLocked(from, to *Router, promise *pipeTransportPairPromise) {
	peers := mapRouterPipeTransports[from]
	if peers == nil {
		peers = make(map[*Router]*pipeTransportPairPromise)
		mapRouterPipeTransports[from] = peers
	}
	peers[to] = promise
}

// deletePipeTransportPair forgets the pair on both Routers.
func deletePipeTransportPair(a, b *Router) {
	routerPipeMu.Lock()
	defer routerPipeMu.Unlock()

	deletePipeTransportPairLocked(a, b)
	deletePipeTransportPairLocked(b, a)
}

func deletePipeTransportPairLocked(from, to *Router) {
	peers := mapRouterPipeTransports[from]
	if peers == nil {
		return
	}
	delete(peers, to)
	if len(peers) == 0 {
		delete(mapRouterPipeTransports, from)
	}
}
