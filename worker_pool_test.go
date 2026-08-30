package mediasoup

import (
	"context"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestWorkerPool(t *testing.T, size int) *WorkerPool {
	t.Helper()

	pool, err := NewWorkerPool(WorkerBinPath, size, func(s *WorkerSettings) {
		s.LogLevel = WorkerLogLevelWarn
	})
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return pool
}

func TestWorkerPoolRoundRobin(t *testing.T) {
	pool := newTestWorkerPool(t, 3)
	workers := pool.Workers()
	require.Len(t, workers, 3)

	// Two full cycles must visit every worker in order.
	for i := 0; i < len(workers)*2; i++ {
		assert.Same(t, workers[i%len(workers)], pool.Next(), "cycle position %d", i)
	}
}

func TestWorkerPoolDefaultSize(t *testing.T) {
	pool := newTestWorkerPool(t, 0)
	assert.Len(t, pool.Workers(), runtime.NumCPU())
}

func TestWorkerPoolCreateRouter(t *testing.T) {
	pool := newTestWorkerPool(t, 2)
	workers := pool.Workers()

	// Consecutive routers land on different workers, which is the whole point of
	// the pool.
	first, err := pool.CreateRouter(&RouterOptions{})
	require.NoError(t, err)
	second, err := pool.CreateRouter(&RouterOptions{})
	require.NoError(t, err)

	firstDump, err := workers[0].Dump()
	require.NoError(t, err)
	secondDump, err := workers[1].Dump()
	require.NoError(t, err)

	assert.Contains(t, firstDump.RouterIds, first.Id())
	assert.Contains(t, secondDump.RouterIds, second.Id())
}

// A dead worker must be skipped rather than handed out, and its routers are gone
// with it.
func TestWorkerPoolSkipsDeadWorkers(t *testing.T) {
	pool := newTestWorkerPool(t, 2)
	workers := pool.Workers()

	died := make(chan struct{})
	// Checked from inside the callback on purpose: a worker reports its death
	// before it finishes closing, so at this point Closed() is still false while
	// the subprocess is already gone. A pool that only consults Closed() hands out
	// the dead worker here.
	workers[0].OnDied(func(ctx context.Context, err error) {
		assert.NotSame(t, workers[0], pool.Next(), "the pool handed out a worker that had just died")
		close(died)
	})

	process, err := os.FindProcess(workers[0].Pid())
	require.NoError(t, err)
	require.NoError(t, process.Kill())

	select {
	case <-died:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for the worker to die")
	}

	// Both the dead worker's turn and the live one's must yield the live one.
	assert.Same(t, workers[1], pool.Next())
	assert.Same(t, workers[1], pool.Next())

	router, err := pool.CreateRouter(&RouterOptions{})
	require.NoError(t, err)

	dump, err := workers[1].Dump()
	require.NoError(t, err)
	assert.Contains(t, dump.RouterIds, router.Id())

	// Workers keeps reporting the dead one, so callers can inspect it.
	assert.Len(t, pool.Workers(), 2)
}

func TestWorkerPoolNoWorkerAvailable(t *testing.T) {
	pool := newTestWorkerPool(t, 1)

	pool.Close()
	assert.True(t, pool.Closed())

	assert.Nil(t, pool.Next())

	_, err := pool.CreateRouter(&RouterOptions{})
	assert.ErrorIs(t, err, ErrNoWorkerAvailable)

	// Closing twice is a no-op.
	pool.Close()
}

// Close is also what NewWorkerPool uses to clean up after a partial start, so
// this covers both.
func TestWorkerPoolCloseClosesEveryWorker(t *testing.T) {
	pool := newTestWorkerPool(t, 3)
	workers := pool.Workers()
	require.Len(t, workers, 3)

	pool.Close()

	for i, worker := range workers {
		assert.True(t, worker.Closed(), "worker %d is still open", i)
		assert.False(t, worker.Died(), "worker %d must count as closed, not died", i)
	}
}

func TestWorkerPoolStartFailure(t *testing.T) {
	before := runtime.NumGoroutine()

	pool, err := NewWorkerPool("/nonexistent/mediasoup-worker", 2)
	require.Error(t, err)
	assert.Nil(t, pool)

	// Polled from this goroutine: assert.Eventually would run the check on a
	// goroutine of its own and count it.
	waitUntil(t, func() bool {
		return runtime.NumGoroutine() <= before
	}, 5*time.Second, "the failed start to leave no goroutines behind")
}
