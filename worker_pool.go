package mediasoup

import (
	"context"
	"fmt"
	"runtime"
	"slices"
	"sync"
	"time"
)

// WorkerPool runs a group of workers. A worker is a subprocess pinned to a single
// CPU core, so spreading rooms across a pool is how an application uses more than
// one core.
//
// Which worker a new router goes on comes from the pool's Scheduler, RoundRobin
// unless SetScheduler says otherwise.
//
// Routers on different workers cannot forward media to each other directly. Put
// everything that talks to each other on one router, and use Router.PipeToRouter
// when that is not possible. PipeToRouter picks KeepId itself when the two
// routers share a worker, so the caller does not have to know which worker
// CreateRouter landed on.
//
// A worker that dies (the C++ subprocess aborting) is replaced with a new empty
// one so later CreateRouter calls can still use that core. The rooms that were
// on the dead worker are gone; register OnWorkerDied to renegotiate those
// clients. Close does not replace: that shutdown is ours.
type WorkerPool struct {
	mu                      sync.RWMutex
	workerPath              string
	options                 []Option
	workers                 []*Worker
	scheduler               Scheduler
	webRtcServerOptions     *WebRtcServerOptions
	webRtcServers           []*WebRtcServer
	workerDiedListeners     listenerList[func(context.Context, *Worker, error)]
	workerReplacedListeners listenerList[func(context.Context, *Worker, *Worker)]
	closed                  bool
}

// NewWorkerPool starts size workers, or runtime.NumCPU() of them when size is not
// positive. The options apply to every worker, including ones started later to
// replace a worker that died.
//
// If any worker fails to start, the ones already started are closed, so a failed
// call leaves no subprocesses behind.
func NewWorkerPool(workerBinaryPath string, size int, options ...Option) (*WorkerPool, error) {
	if size <= 0 {
		size = runtime.NumCPU()
	}

	pool := &WorkerPool{
		workerPath: workerBinaryPath,
		options:    options,
		workers:    make([]*Worker, 0, size),
		scheduler:  RoundRobin(),
	}

	for i := 0; i < size; i++ {
		worker, err := NewWorker(workerBinaryPath, options...)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("starting worker %d of %d: %w", i+1, size, err)
		}
		pool.watch(worker)
		pool.workers = append(pool.workers, worker)
	}

	return pool, nil
}

// Next returns a worker to put the next router on, skipping workers that have
// died and not yet been replaced. It returns nil once no worker is left alive.
//
// Which of the live workers it is comes from the pool's Scheduler, RoundRobin
// unless SetScheduler says otherwise.
func (p *WorkerPool) Next() *Worker {
	p.mu.RLock()
	scheduler := p.scheduler
	candidates := make([]*Worker, 0, len(p.workers))
	for _, worker := range p.workers {
		// Died has to be consulted too. A worker that dies reports it before it
		// finishes closing, so between those two points Closed is still false while
		// the subprocess is already gone.
		if !worker.Closed() && !worker.Died() {
			candidates = append(candidates, worker)
		}
	}
	p.mu.RUnlock()

	if len(candidates) == 0 {
		return nil
	}

	// Picking outside the lock: a scheduler reads the workers it is handed, and an
	// application's own strategy must not be able to stall the rest of the pool.
	return scheduler.Pick(candidates)
}

// SetScheduler replaces the strategy Next uses to choose among the live workers.
// A nil scheduler restores the default, RoundRobin.
//
// It may be called at any time. A scheduler that carries state, as RoundRobin
// does, starts from whatever it is handed first.
func (p *WorkerPool) SetScheduler(scheduler Scheduler) {
	if scheduler == nil {
		scheduler = RoundRobin()
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.scheduler = scheduler
}

// Workers returns the workers currently in the pool, including a replacement
// once one has been swapped in for a worker that died.
func (p *WorkerPool) Workers() []*Worker {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return slices.Clone(p.workers)
}

// OnWorkerDied is notified when a worker process exits on its own. The rooms it
// hosted are gone; the listener is where those clients get told to renegotiate.
// A replacement is started after the dead worker finishes closing. Call the
// returned function to remove the listener again.
func (p *WorkerPool) OnWorkerDied(listener func(context.Context, *Worker, error)) (removeListener func()) {
	return addListener(&p.mu, &p.workerDiedListeners, listener)
}

// OnWorkerReplaced is notified once a died worker has been swapped for a new
// empty one. Call the returned function to remove the listener again.
func (p *WorkerPool) OnWorkerReplaced(listener func(context.Context, *Worker, *Worker)) (removeListener func()) {
	return addListener(&p.mu, &p.workerReplacedListeners, listener)
}

// CreateRouter creates a router on the worker returned by Next.
func (p *WorkerPool) CreateRouter(options *RouterOptions) (*Router, error) {
	return p.CreateRouterContext(context.Background(), options)
}

func (p *WorkerPool) CreateRouterContext(ctx context.Context, options *RouterOptions) (*Router, error) {
	worker := p.Next()
	if worker == nil {
		return nil, ErrNoWorkerAvailable
	}

	return worker.CreateRouterContext(ctx, options)
}

// CreateWebRtcServer creates a WebRtcServer on every live worker. A server
// belongs to one worker; a transport can only use the server that shares its
// router's worker. WebRtcServerFor picks that one. A worker started later to
// replace one that died gets a server with the same options.
//
// A fixed port without Flags.UDPReusePort is incremented per worker so they
// do not collide (44444, 44445, …). UDPReusePort, Port 0 and PortRange leave
// the port as given. The caller's ListenInfos are not modified.
func (p *WorkerPool) CreateWebRtcServer(options *WebRtcServerOptions) error {
	return p.CreateWebRtcServerContext(context.Background(), options)
}

func (p *WorkerPool) CreateWebRtcServerContext(ctx context.Context, options *WebRtcServerOptions) error {
	p.mu.RLock()
	workers := slices.Clone(p.workers)
	p.mu.RUnlock()

	if len(workers) == 0 {
		return ErrNoWorkerAvailable
	}

	servers := make([]*WebRtcServer, len(workers))
	created := 0
	for i, worker := range workers {
		if worker.Closed() || worker.Died() {
			continue
		}
		server, err := p.createWebRtcServerOn(ctx, worker, i, options)
		if err != nil {
			for _, existing := range servers {
				if existing != nil {
					existing.CloseContext(ctx)
				}
			}
			return fmt.Errorf("creating WebRtcServer on worker %d of %d: %w", i+1, len(workers), err)
		}
		servers[i] = server
		created++
	}
	if created == 0 {
		return ErrNoWorkerAvailable
	}

	p.mu.Lock()
	p.webRtcServerOptions = cloneWebRtcServerOptions(options)
	p.webRtcServers = servers
	p.mu.Unlock()

	return nil
}

// WebRtcServerFor returns the WebRtcServer that shares router’s worker, or nil
// if CreateWebRtcServer has not been called or that worker has no server.
func (p *WorkerPool) WebRtcServerFor(router *Router) *WebRtcServer {
	if router == nil {
		return nil
	}

	p.mu.RLock()
	servers := p.webRtcServers
	p.mu.RUnlock()

	for _, server := range servers {
		if server != nil && server.channel == router.channel && !server.Closed() {
			return server
		}
	}

	return nil
}

// Closed reports whether the pool has been closed. It says nothing about the
// individual workers, which can die on their own.
func (p *WorkerPool) Closed() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.closed
}

// Close closes every worker in the pool and stops replacements.
func (p *WorkerPool) Close() {
	p.CloseContext(context.Background())
}

func (p *WorkerPool) CloseContext(ctx context.Context) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	workers := slices.Clone(p.workers)
	p.mu.Unlock()

	for _, worker := range workers {
		worker.CloseContext(ctx)
	}
}

func (p *WorkerPool) watch(worker *Worker) {
	worker.OnDied(func(ctx context.Context, err error) {
		p.mu.RLock()
		listeners := p.workerDiedListeners.list()
		p.mu.RUnlock()
		for _, listener := range listeners {
			listener(ctx, worker, err)
		}
	})
	// OnClose runs after the worker has released its sockets, so a replacement
	// can rebind the same listen port. OnDied is too early: the dead server is
	// still holding it.
	worker.OnClose(func(ctx context.Context) {
		if !worker.Died() {
			return
		}
		go p.replace(worker)
	})
}

func (p *WorkerPool) replace(dead *Worker) {
	backoff := 100 * time.Millisecond

	for {
		p.mu.RLock()
		closed := p.closed
		path := p.workerPath
		options := p.options
		p.mu.RUnlock()
		if closed {
			return
		}

		worker, err := NewWorker(path, options...)
		if err != nil {
			time.Sleep(backoff)
			if backoff < 5*time.Second {
				backoff *= 2
			}
			continue
		}

		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			worker.Close()
			return
		}
		slot := slices.Index(p.workers, dead)
		if slot < 0 {
			p.mu.Unlock()
			worker.Close()
			return
		}
		p.workers[slot] = worker
		p.watch(worker)
		serverOpts := p.webRtcServerOptions
		p.mu.Unlock()

		if serverOpts != nil {
			// Failure leaves the worker in the pool without a server; the next
			// CreateWebRtcServer call will cover it.
			_, _ = p.createWebRtcServerOn(context.Background(), worker, slot, serverOpts)
		}

		p.mu.RLock()
		listeners := p.workerReplacedListeners.list()
		p.mu.RUnlock()
		for _, listener := range listeners {
			listener(context.Background(), dead, worker)
		}
		return
	}
}

func (p *WorkerPool) createWebRtcServerOn(ctx context.Context, worker *Worker, slot int, options *WebRtcServerOptions) (*WebRtcServer, error) {
	listenInfos, err := listenInfosForWorker(options.ListenInfos, slot)
	if err != nil {
		return nil, err
	}
	perWorker := *options
	perWorker.ListenInfos = listenInfos
	server, err := worker.CreateWebRtcServerContext(ctx, &perWorker)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	if slot >= len(p.webRtcServers) {
		grown := make([]*WebRtcServer, slot+1)
		copy(grown, p.webRtcServers)
		p.webRtcServers = grown
	}
	p.webRtcServers[slot] = server
	p.mu.Unlock()

	return server, nil
}

func cloneWebRtcServerOptions(options *WebRtcServerOptions) *WebRtcServerOptions {
	if options == nil {
		return nil
	}
	cloned := *options
	cloned.ListenInfos = make([]*TransportListenInfo, len(options.ListenInfos))
	for i, info := range options.ListenInfos {
		copied := *info
		cloned.ListenInfos[i] = &copied
	}
	return &cloned
}

// listenInfosForWorker copies infos and, when a worker would otherwise collide
// on a fixed port, bumps that port by index.
func listenInfosForWorker(infos []*TransportListenInfo, index int) ([]*TransportListenInfo, error) {
	out := make([]*TransportListenInfo, len(infos))
	for i, info := range infos {
		copied := *info
		if index > 0 && copied.Port != 0 &&
			copied.PortRange == (TransportPortRange{}) &&
			!copied.Flags.UDPReusePort {
			next := uint32(copied.Port) + uint32(index)
			if next > 65535 {
				return nil, fmt.Errorf("listen port %d + worker index %d overflows", copied.Port, index)
			}
			copied.Port = uint16(next)
		}
		out[i] = &copied
	}
	return out, nil
}
