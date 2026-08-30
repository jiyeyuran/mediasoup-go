package mediasoup

import (
	"context"
	"fmt"
	"runtime"
	"slices"
	"sync"
)

// WorkerPool runs a group of workers. A worker is a subprocess pinned to a single
// CPU core, so spreading rooms across a pool is how an application uses more than
// one core.
//
// Routers on different workers cannot forward media to each other directly. Put
// everything that talks to each other on one router, and use Router.PipeToRouter
// when that is not possible.
//
// The pool intentionally does not restart a worker that dies. Its routers,
// transports and producers are gone with the subprocess, and only the application
// knows whether the affected clients should renegotiate elsewhere or be dropped.
// Register Worker.OnDied on the workers to find out.
type WorkerPool struct {
	mu      sync.Mutex
	workers []*Worker
	next    int
	closed  bool
}

// NewWorkerPool starts size workers, or runtime.NumCPU() of them when size is not
// positive. The options apply to every worker.
//
// If any worker fails to start, the ones already started are closed, so a failed
// call leaves no subprocesses behind.
func NewWorkerPool(workerBinaryPath string, size int, options ...Option) (*WorkerPool, error) {
	if size <= 0 {
		size = runtime.NumCPU()
	}

	pool := &WorkerPool{workers: make([]*Worker, 0, size)}

	for i := 0; i < size; i++ {
		worker, err := NewWorker(workerBinaryPath, options...)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("starting worker %d of %d: %w", i+1, size, err)
		}
		pool.workers = append(pool.workers, worker)
	}

	return pool, nil
}

// Next returns a worker to put the next router on, cycling through the pool and
// skipping workers that have died. It returns nil once no worker is left alive.
//
// Round-robin spreads rooms evenly but knows nothing about how expensive each one
// is. To weigh workers by actual load, range over Workers instead and pick using
// Worker.ResourceUsage or your own accounting.
func (p *WorkerPool) Next() *Worker {
	p.mu.Lock()
	defer p.mu.Unlock()

	// One full cycle: if no worker is usable there is nothing to hand out.
	for range p.workers {
		worker := p.workers[p.next]
		p.next = (p.next + 1) % len(p.workers)

		// Died has to be consulted too. A worker that dies reports it before it
		// finishes closing, so between those two points Closed is still false while
		// the subprocess is already gone.
		if !worker.Closed() && !worker.Died() {
			return worker
		}
	}

	return nil
}

// Workers returns the workers of the pool, including any that have died.
func (p *WorkerPool) Workers() []*Worker {
	p.mu.Lock()
	defer p.mu.Unlock()

	return slices.Clone(p.workers)
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

// Closed reports whether the pool has been closed. It says nothing about the
// individual workers, which can die on their own.
func (p *WorkerPool) Closed() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.closed
}

// Close closes every worker in the pool.
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
