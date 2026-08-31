package mediasoup

import (
	"context"
	"os"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestWorkerPool(t *testing.T, size int, options ...Option) *WorkerPool {
	t.Helper()

	defaults := []Option{func(s *WorkerSettings) {
		s.LogLevel = WorkerLogLevelWarn
	}}
	pool, err := NewWorkerPool(WorkerBinPath, size, append(defaults, options...)...)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return pool
}

func withTestWebRtcListenInfos(port uint16) Option {
	return func(s *WorkerSettings) {
		s.WebRtcListenInfos = []*TransportListenInfo{
			{Protocol: TransportProtocolUDP, Ip: "127.0.0.1", Port: port},
		}
	}
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

func TestWorkerPoolSetScheduler(t *testing.T) {
	pool := newTestWorkerPool(t, 3)
	workers := pool.Workers()

	// Always the last worker, so a router landing anywhere else means the pool
	// ignored the scheduler.
	pool.SetScheduler(SchedulerFunc(func(candidates []*Worker) *Worker {
		return candidates[len(candidates)-1]
	}))

	for i := 0; i < 3; i++ {
		assert.Same(t, workers[2], pool.Next(), "call %d", i)
	}

	router, err := pool.CreateRouter(&RouterOptions{})
	require.NoError(t, err)

	dump, err := workers[2].Dump()
	require.NoError(t, err)
	assert.Contains(t, dump.RouterIds, router.Id())
}

func TestWorkerPoolSetSchedulerNilRestoresRoundRobin(t *testing.T) {
	pool := newTestWorkerPool(t, 2)
	workers := pool.Workers()

	pool.SetScheduler(SchedulerFunc(func(candidates []*Worker) *Worker {
		return candidates[len(candidates)-1]
	}))
	require.Same(t, workers[1], pool.Next())

	pool.SetScheduler(nil)
	assert.Same(t, workers[0], pool.Next())
	assert.Same(t, workers[1], pool.Next())
}

// A scheduler must never have to decide whether a worker is still usable.
func TestWorkerPoolSchedulerSeesOnlyLiveWorkers(t *testing.T) {
	pool := newTestWorkerPool(t, 2)
	workers := pool.Workers()

	var candidates []*Worker
	pool.SetScheduler(SchedulerFunc(func(live []*Worker) *Worker {
		candidates = slices.Clone(live)
		return live[0]
	}))

	died := make(chan struct{})
	workers[0].OnDied(func(context.Context, error) {
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

	assert.Same(t, workers[1], pool.Next())
	assert.Equal(t, []*Worker{workers[1]}, candidates)
}

// Pick runs with no pool lock held, so a scheduler may consult the pool it belongs
// to. Moving the call back inside the lock would deadlock here.
func TestWorkerPoolSchedulerMayConsultThePool(t *testing.T) {
	pool := newTestWorkerPool(t, 2)

	pool.SetScheduler(SchedulerFunc(func(candidates []*Worker) *Worker {
		if pool.Closed() || len(pool.Workers()) == 0 {
			return nil
		}

		return candidates[0]
	}))

	assert.Same(t, pool.Workers()[0], pool.Next())
}

func TestWorkerPoolSchedulerDeclining(t *testing.T) {
	pool := newTestWorkerPool(t, 1)

	pool.SetScheduler(SchedulerFunc(func([]*Worker) *Worker {
		return nil
	}))

	assert.Nil(t, pool.Next())

	_, err := pool.CreateRouter(&RouterOptions{})
	assert.ErrorIs(t, err, ErrNoWorkerAvailable)
}

func TestWorkerPoolLeastLoadedDefaultsToRtpStreams(t *testing.T) {
	pool := newTestWorkerPool(t, 2)
	workers := pool.Workers()
	pool.SetScheduler(LeastLoaded(nil))

	// Three empty rooms on the first worker. Counting rooms would call that
	// loaded; producers and consumers would not, and both workers are still idle.
	for i := 0; i < 3; i++ {
		_, err := workers[0].CreateRouter(&RouterOptions{})
		require.NoError(t, err)
	}
	assert.Same(t, workers[0], pool.Next())

	// Data channels on the first worker still do not count: SCTP is cheap next to
	// forwarding RTP, and counting it here would pack rooms onto the other worker.
	sctpRouter := createRouter(workers[0])
	sctpTransport := createWebRtcTransport(sctpRouter, func(o *WebRtcTransportOptions) {
		o.EnableSctp = true
	})
	createDataProducer(sctpTransport)
	assert.Same(t, workers[0], pool.Next())

	// A real stream on the first worker is what finally sends the next room elsewhere.
	mediaRouter := createRouter(workers[0])
	createAudioProducer(createPlainTransport(mediaRouter))
	assert.Same(t, workers[1], pool.Next())

	router, err := pool.CreateRouter(&RouterOptions{})
	require.NoError(t, err)

	dump, err := workers[1].Dump()
	require.NoError(t, err)
	assert.Contains(t, dump.RouterIds, router.Id())
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

	assert.Same(t, workers[0], first.Worker())
	assert.Same(t, workers[1], second.Worker())
}

func TestWorkerPoolCreateWebRtcServer(t *testing.T) {
	pool := newTestWorkerPool(t, 2, withTestWebRtcListenInfos(0))
	workers := pool.Workers()

	require.NotNil(t, workers[0].WebRtcServer())
	require.NotNil(t, workers[1].WebRtcServer())

	firstDump, err := workers[0].Dump()
	require.NoError(t, err)
	secondDump, err := workers[1].Dump()
	require.NoError(t, err)
	require.Len(t, firstDump.WebRtcServerIds, 1)
	require.Len(t, secondDump.WebRtcServerIds, 1)

	first, err := pool.CreateRouter(&RouterOptions{})
	require.NoError(t, err)
	second, err := pool.CreateRouter(&RouterOptions{})
	require.NoError(t, err)

	assert.Equal(t, firstDump.WebRtcServerIds[0], first.Worker().WebRtcServer().Id())
	assert.Equal(t, secondDump.WebRtcServerIds[0], second.Worker().WebRtcServer().Id())
	assert.NotSame(t, first.Worker().WebRtcServer(), second.Worker().WebRtcServer())

	transport, err := first.CreateWebRtcTransport(&WebRtcTransportOptions{
		WebRtcServer: first.Worker().WebRtcServer(),
	})
	require.NoError(t, err)
	assert.False(t, transport.Closed())
}

func TestWorkerPoolCreateWebRtcServerIncrementsPortWithoutReuse(t *testing.T) {
	port := pickUdpPort()
	pool := newTestWorkerPool(t, 2, withTestWebRtcListenInfos(port))

	first, err := pool.CreateRouter(&RouterOptions{})
	require.NoError(t, err)
	second, err := pool.CreateRouter(&RouterOptions{})
	require.NoError(t, err)

	firstDump, err := first.Worker().WebRtcServer().Dump()
	require.NoError(t, err)
	secondDump, err := second.Worker().WebRtcServer().Dump()
	require.NoError(t, err)
	require.Len(t, firstDump.UdpSockets, 1)
	require.Len(t, secondDump.UdpSockets, 1)
	assert.Equal(t, port, firstDump.UdpSockets[0].Port)
	assert.Equal(t, port+1, secondDump.UdpSockets[0].Port)
}

func TestListenInfosForWorker(t *testing.T) {
	infos := []*TransportListenInfo{
		{Protocol: TransportProtocolUDP, Ip: "127.0.0.1", Port: 44444},
		{Protocol: TransportProtocolTCP, Ip: "127.0.0.1", Port: 44444},
	}

	worker0, err := listenInfosForWorker(infos, 0)
	require.NoError(t, err)
	assert.Equal(t, uint16(44444), worker0[0].Port)
	assert.Equal(t, uint16(44444), infos[0].Port, "the caller's ListenInfos must not be modified")

	worker1, err := listenInfosForWorker(infos, 1)
	require.NoError(t, err)
	assert.Equal(t, uint16(44445), worker1[0].Port)
	assert.Equal(t, uint16(44445), worker1[1].Port)

	reuse := []*TransportListenInfo{{
		Protocol: TransportProtocolUDP,
		Ip:       "127.0.0.1",
		Port:     44444,
		Flags:    TransportSocketFlags{UDPReusePort: true},
	}}
	kept, err := listenInfosForWorker(reuse, 3)
	require.NoError(t, err)
	assert.Equal(t, uint16(44444), kept[0].Port)

	ranged := []*TransportListenInfo{{
		Protocol:  TransportProtocolUDP,
		Ip:        "127.0.0.1",
		Port:      44444,
		PortRange: TransportPortRange{Min: 40000, Max: 40010},
	}}
	left, err := listenInfosForWorker(ranged, 2)
	require.NoError(t, err)
	assert.Equal(t, uint16(44444), left[0].Port)

	_, err = listenInfosForWorker([]*TransportListenInfo{{
		Protocol: TransportProtocolUDP,
		Ip:       "127.0.0.1",
		Port:     65535,
	}}, 1)
	require.Error(t, err)
}

// A dead worker must be skipped rather than handed out. Its routers are gone
// with it; a replacement is started after it finishes closing so later rooms
// can still use that core.
func TestWorkerPoolSkipsDeadWorkers(t *testing.T) {
	pool := newTestWorkerPool(t, 2)
	workers := pool.Workers()

	died := make(chan struct{})
	// Checked from inside the callback on purpose: a worker reports its death
	// before it finishes closing, so at this point Closed() is still false while
	// the subprocess is already gone. A pool that only consults Closed() hands out
	// the dead worker here. The replacement has not been swapped in yet.
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

	// Until the replacement is in, both turns yield the worker that is still up.
	assert.Same(t, workers[1], pool.Next())
	assert.Same(t, workers[1], pool.Next())

	router, err := pool.CreateRouter(&RouterOptions{})
	require.NoError(t, err)

	dump, err := workers[1].Dump()
	require.NoError(t, err)
	assert.Contains(t, dump.RouterIds, router.Id())
}

func TestWorkerPoolReplacesDeadWorker(t *testing.T) {
	pool := newTestWorkerPool(t, 2)
	workers := pool.Workers()
	dead := workers[0]

	replaced := make(chan *Worker, 1)
	pool.OnWorkerReplaced(func(_ context.Context, old, next *Worker) {
		assert.Same(t, dead, old)
		replaced <- next
	})

	process, err := os.FindProcess(dead.Pid())
	require.NoError(t, err)
	require.NoError(t, process.Kill())

	var next *Worker
	select {
	case next = <-replaced:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for the replacement worker")
	}

	require.False(t, next.Closed())
	require.False(t, next.Died())
	assert.NotSame(t, dead, next)
	assert.Contains(t, pool.Workers(), next)
	assert.Len(t, pool.Workers(), 2)

	// The replacement is empty and must accept a new room.
	router, err := pool.CreateRouter(&RouterOptions{})
	require.NoError(t, err)
	assert.False(t, router.Closed())
}

func TestWorkerPoolReplacesWebRtcServerOnDeadWorker(t *testing.T) {
	pool := newTestWorkerPool(t, 2, withTestWebRtcListenInfos(pickUdpPort()))
	workers := pool.Workers()

	replaced := make(chan *Worker, 1)
	pool.OnWorkerReplaced(func(_ context.Context, _, next *Worker) {
		replaced <- next
	})

	process, err := os.FindProcess(workers[0].Pid())
	require.NoError(t, err)
	require.NoError(t, process.Kill())

	var next *Worker
	select {
	case next = <-replaced:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for the replacement worker")
	}

	router, err := next.CreateRouter(&RouterOptions{})
	require.NoError(t, err)
	server := router.Worker().WebRtcServer()
	require.NotNil(t, server)
	assert.False(t, server.Closed())

	transport, err := router.CreateWebRtcTransport(&WebRtcTransportOptions{
		WebRtcServer: server,
	})
	require.NoError(t, err)
	assert.False(t, transport.Closed())
}

func TestWorkerPoolCloseDoesNotReplace(t *testing.T) {
	pool := newTestWorkerPool(t, 2)
	workers := pool.Workers()

	replaced := make(chan struct{}, 1)
	pool.OnWorkerReplaced(func(context.Context, *Worker, *Worker) {
		replaced <- struct{}{}
	})

	pool.Close()

	select {
	case <-replaced:
		t.Fatal("Close replaced a worker")
	case <-time.After(200 * time.Millisecond):
	}

	for i, worker := range workers {
		assert.True(t, worker.Closed(), "worker %d is still open", i)
		assert.False(t, worker.Died(), "worker %d must count as closed, not died", i)
		assert.Contains(t, pool.Workers(), worker)
	}
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
