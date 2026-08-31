package mediasoup

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var WorkerBinPath = os.Getenv("MEDIASOUP_WORKER_BIN")

func init() {
	if len(WorkerBinPath) == 0 {
		WorkerBinPath = "../mediasoup/worker/out/Release/mediasoup-worker"
	}
}

func newTestWorker(options ...Option) *Worker {
	defaultOptions := []Option{
		func(o *WorkerSettings) {
			o.LogLevel = WorkerLogLevelDebug
		},
	}
	worker, err := NewWorker(WorkerBinPath, append(defaultOptions, options...)...)
	if err != nil {
		panic(err)
	}
	return worker
}

// pickUdpPort get a free udp port of localhost
func pickUdpPort() uint16 {
	a, _ := net.ResolveUDPAddr("udp", "localhost:0")
	l, _ := net.ListenUDP("udp", a)
	defer l.Close()
	return uint16(l.LocalAddr().(*net.UDPAddr).Port)
}

// pickTcpPort get a free tcp port of localhost
func pickTcpPort() uint16 {
	a, _ := net.ResolveTCPAddr("tcp", "localhost:0")
	l, _ := net.ListenTCP("tcp", a)
	defer l.Close()
	return uint16(l.Addr().(*net.TCPAddr).Port)
}

func TestWorkerDump(t *testing.T) {
	worker := newTestWorker()
	dump, err := worker.Dump()
	require.NoError(t, err)
	assert.EqualValues(t, worker.Pid(), dump.Pid)
}

func TestWorkerGetResourceUsage(t *testing.T) {
	worker := newTestWorker()
	usage, err := worker.GetResourceUsage()
	require.NoError(t, err)
	assert.NotZero(t, usage)
}

func TestWorkerUpdateSettings(t *testing.T) {
	worker := newTestWorker()

	// Test updating settings with valid log level and log tags
	settings := &WorkerUpdatableSettings{
		LogLevel: WorkerLogLevelWarn,
		LogTags:  []WorkerLogTag{WorkerLogTagInfo, WorkerLogTagIce},
	}
	err := worker.UpdateSettings(settings)
	require.NoError(t, err)

	// Test updating settings with empty log tags
	settings = &WorkerUpdatableSettings{
		LogLevel: WorkerLogLevelError,
		LogTags:  []WorkerLogTag{},
	}
	err = worker.UpdateSettings(settings)
	require.NoError(t, err)

	// Test updating settings with invalid log level
	settings = &WorkerUpdatableSettings{
		LogLevel: "invalid_log_level",
		LogTags:  []WorkerLogTag{WorkerLogTagRtp},
	}
	err = worker.UpdateSettings(settings)
	assert.Error(t, err)
}

func TestWorkerSettingsWebRtcServer(t *testing.T) {
	worker := newTestWorker(func(s *WorkerSettings) {
		s.WebRtcListenInfos = []*TransportListenInfo{
			{Protocol: TransportProtocolUDP, Ip: "127.0.0.1"},
		}
	})
	defer worker.Close()

	require.NotNil(t, worker.WebRtcServer())
	dump, err := worker.Dump()
	require.NoError(t, err)
	assert.Contains(t, dump.WebRtcServerIds, worker.WebRtcServer().Id())
}

func TestWorkerCreateWebRtcServer(t *testing.T) {
	mymock := new(MockedHandler)
	defer mymock.AssertExpectations(t)

	mymock.On("OnNewWebRtcServer", mock.IsType(context.Background()), mock.IsType(&WebRtcServer{})).Once()

	worker := newTestWorker()
	worker.OnNewWebRtcServer(mymock.OnNewWebRtcServer)
	server, err := worker.CreateWebRtcServer(&WebRtcServerOptions{
		ListenInfos: []*TransportListenInfo{
			{Protocol: TransportProtocolUDP, Ip: "127.0.0.1", Port: pickUdpPort()},
			{Protocol: TransportProtocolTCP, Ip: "127.0.0.1", AnnouncedAddress: "foo.bar.org", Port: pickTcpPort()},
		},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, server)
	dump, _ := worker.Dump()
	assert.Contains(t, dump.WebRtcServerIds, server.Id())
}

func TestWorkerCreateRouter(t *testing.T) {
	mymock := new(MockedHandler)
	defer mymock.AssertExpectations(t)

	mymock.On("OnNewRouter", mock.IsType(context.Background()), mock.IsType(&Router{})).Times(2)

	worker := newTestWorker()
	worker.OnNewRouter(mymock.OnNewRouter)
	router, err := worker.CreateRouter(&RouterOptions{
		MediaCodecs: []*RtpCodecCapability{
			{
				Kind:      "audio",
				MimeType:  "audio/opus",
				ClockRate: 48000,
				Channels:  2,
				Parameters: RtpCodecSpecificParameters{
					Useinbandfec: 1,
				},
			},
			{
				Kind:      "video",
				MimeType:  "video/VP8",
				ClockRate: 90000,
			},
			{
				Kind:      "video",
				MimeType:  "video/H264",
				ClockRate: 90000,
				Parameters: RtpCodecSpecificParameters{
					LevelAsymmetryAllowed: 1,
					PacketizationMode:     1,
					ProfileLevelId:        "4d0032",
				},
			},
		},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, router.Id())
	dump, _ := worker.Dump()
	assert.Contains(t, dump.RouterIds, router.Id())

	router, _ = worker.CreateRouter(&RouterOptions{})
	dump, _ = worker.Dump()
	assert.Contains(t, dump.RouterIds, router.Id())
}

func TestWorkerClose(t *testing.T) {
	t.Run("close normally", func(t *testing.T) {
		worker := newTestWorker()

		var diedCalls atomic.Int32
		worker.OnDied(func(context.Context, error) { diedCalls.Add(1) })

		subprocessClosed := make(chan struct{})
		worker.OnSubprocessClose(func(ctx context.Context) { close(subprocessClosed) })

		worker.Close()
		assert.True(t, worker.Closed())

		select {
		case <-subprocessClosed:
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for the subprocessclose event")
		}

		assert.True(t, worker.SubprocessClosed())
		// Being shut down on request is not dying.
		assert.False(t, worker.Died())
		assert.NoError(t, worker.Err())
		assert.Zero(t, diedCalls.Load())
	})

	t.Run("process killed", func(t *testing.T) {
		worker := newTestWorker()

		assert.False(t, worker.Died())
		assert.False(t, worker.SubprocessClosed())
		assert.NoError(t, worker.Err())

		died := make(chan error, 1)
		worker.OnDied(func(ctx context.Context, err error) { died <- err })

		closed := make(chan struct{})
		worker.OnClose(func(ctx context.Context) { close(closed) })

		process, err := os.FindProcess(worker.Pid())
		require.NoError(t, err)
		require.NoError(t, process.Kill())

		select {
		case err := <-died:
			assert.Error(t, err)
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for the died event")
		}

		select {
		case <-closed:
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for the close event")
		}

		assert.True(t, worker.Died())
		assert.True(t, worker.SubprocessClosed())
		assert.Error(t, worker.Err())
		assert.True(t, worker.Closed())
	})

	t.Run("router is closed when the process dies", func(t *testing.T) {
		worker := newTestWorker()
		router, err := worker.CreateRouter(&RouterOptions{})
		require.NoError(t, err)

		workerClosed := make(chan struct{})
		router.OnWorkerClosed(func(ctx context.Context) { close(workerClosed) })

		process, err := os.FindProcess(worker.Pid())
		require.NoError(t, err)
		require.NoError(t, process.Kill())

		select {
		case <-workerClosed:
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for the router workerclosed event")
		}

		assert.True(t, router.Closed())
	})
}

func TestWorkerObjectCounts(t *testing.T) {
	worker := newTestWorker()
	defer worker.Close()

	assert.Equal(t, objectCounts{}, worker.objectCounts())

	router := createRouter(worker)
	transport := createPlainTransport(router)
	sctpTransport := createWebRtcTransport(router, func(o *WebRtcTransportOptions) {
		o.EnableSctp = true
	})

	producer := createAudioProducer(transport)
	consumer := createConsumer(transport, producer.Id())
	dataProducer := createDataProducer(sctpTransport)
	dataConsumer := createDataConsumer(sctpTransport, dataProducer.Id())

	assert.Equal(t, objectCounts{
		producers:     1,
		consumers:     1,
		dataProducers: 1,
		dataConsumers: 1,
	}, worker.objectCounts())
	assert.Equal(t, router.objectCounts(), worker.objectCounts())
	assert.Equal(t, 2, worker.objectCounts().rtpStreams())

	consumer.Close()
	assert.Equal(t, objectCounts{
		producers:     1,
		dataProducers: 1,
		dataConsumers: 1,
	}, worker.objectCounts())

	producer.Close()
	dataConsumer.Close()
	dataProducer.Close()
	assert.Equal(t, objectCounts{}, worker.objectCounts())
}

func TestWorkerChannelRequestObserver(t *testing.T) {
	var (
		mu    sync.Mutex
		calls []ChannelRequestStats
	)
	observed := func() []ChannelRequestStats {
		mu.Lock()
		defer mu.Unlock()

		return slices.Clone(calls)
	}

	worker := newTestWorker(func(s *WorkerSettings) {
		s.OnChannelRequest = func(stats ChannelRequestStats) {
			mu.Lock()
			defer mu.Unlock()

			calls = append(calls, stats)
		}
	})
	defer worker.Close()

	assert.Zero(t, worker.ChannelPendingRequests())

	router, err := worker.CreateRouter(&RouterOptions{})
	require.NoError(t, err)

	createRouter := findRequestStats(observed(), "WORKER_CREATE_ROUTER")
	require.NotNil(t, createRouter, "the create router request was not reported")
	assert.NoError(t, createRouter.Err)
	assert.Positive(t, createRouter.Duration)
	assert.Equal(t, "worker", createRouter.HandlerID)
	// Nothing else was in flight, and the request itself is already accounted for.
	assert.Zero(t, createRouter.Pending)

	transport, err := router.CreateDirectTransport(&DirectTransportOptions{})
	require.NoError(t, err)

	dump, err := transport.Dump()
	require.NoError(t, err)

	transportDump := findRequestStats(observed(), "TRANSPORT_DUMP")
	require.NotNil(t, transportDump, "the transport dump request was not reported")
	assert.Equal(t, dump.Id, transportDump.HandlerID, "requests are addressed to the object")

	// A request the worker rejects must be reported with its error rather than
	// dropped.
	require.Error(t, transport.SetMaxIncomingBitrate(100))

	rejected := findRequestStats(observed(), "TRANSPORT_SET_MAX_INCOMING_BITRATE")
	assert.Nil(t, rejected, "a direct transport rejects this before it reaches the worker")

	plainTransport, err := router.CreatePlainTransport(&PlainTransportOptions{
		ListenInfo: TransportListenInfo{Protocol: TransportProtocolUDP, Ip: "127.0.0.1"},
	})
	require.NoError(t, err)
	require.NoError(t, plainTransport.SetMinOutgoingBitrate(600000))
	require.Error(t, plainTransport.SetMaxOutgoingBitrate(100), "max below min must be rejected")

	rejected = findRequestStats(observed(), "TRANSPORT_SET_MAX_OUTGOING_BITRATE")
	require.NotNil(t, rejected, "the rejected request was not reported")
	assert.Error(t, rejected.Err)
	assert.Positive(t, rejected.Duration)
}

func findRequestStats(calls []ChannelRequestStats, method string) *ChannelRequestStats {
	for _, stats := range calls {
		if stats.Method == method {
			return &stats
		}
	}
	return nil
}

func TestWorkerNoGoroutineLeaks(t *testing.T) {
	numOfGoroutines := runtime.NumGoroutine()

	worker := newTestWorker(func(s *WorkerSettings) {
		s.LogLevel = WorkerLogLevelWarn
	})

	server, _ := worker.CreateWebRtcServer(&WebRtcServerOptions{
		ListenInfos: []*TransportListenInfo{
			{Protocol: TransportProtocolUDP, Ip: "127.0.0.1", Port: 0},
			{Protocol: TransportProtocolTCP, Ip: "127.0.0.1", AnnouncedAddress: "foo.bar.org", Port: 0},
		},
	})

	n := 10
	var mu sync.Mutex
	var wg sync.WaitGroup

	var routers []*Router

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			router := createRouter(worker)
			mu.Lock()
			routers = append(routers, router)
			mu.Unlock()
		}()
	}

	wg.Wait()

	var transports []*Transport

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, router := range routers {
				transport1 := createPlainTransport(router, func(o *PlainTransportOptions) {
					o.EnableSctp = true
				})
				transport2 := createWebRtcTransport(router, func(o *WebRtcTransportOptions) {
					o.EnableSctp = true
				})
				transport3 := createWebRtcTransport(router, func(o *WebRtcTransportOptions) {
					o.WebRtcServer = server
					o.EnableSctp = true
				})
				transport4 := createDirectTransport(router)
				mu.Lock()
				transports = append(transports, transport1, transport2, transport3, transport4)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	var producers []*Producer
	var dataProducers []*DataProducer
	var consumers []*Consumer
	var dataConsumers []*DataConsumer

	for _, transport := range transports {
		wg.Add(1)
		go func(transport *Transport) {
			defer wg.Done()
			audioProducer := createAudioProducer(transport)
			videoProducer := createVideoProducer(transport)
			dataProducer := createDataProducer(transport)
			mu.Lock()
			producers = append(producers, audioProducer, videoProducer)
			dataProducers = append(dataProducers, dataProducer)
			mu.Unlock()
			for i := 0; i < n; i++ {
				wg.Add(1)
				go func(audioProducer, videoProducer, dataProducer interface{ Id() string }) {
					defer wg.Done()
					consumer1 := createConsumer(transport, audioProducer.Id())
					consumer2 := createConsumer(transport, videoProducer.Id())
					dataConsumer := createDataConsumer(transport, dataProducer.Id())
					mu.Lock()
					consumers = append(consumers, consumer1, consumer2)
					dataConsumers = append(dataConsumers, dataConsumer)
					mu.Unlock()
				}(audioProducer, videoProducer, dataProducer)
			}
		}(transport)
	}

	wg.Wait()

	for _, producer := range producers {
		wg.Add(1)
		go func(producer *Producer) {
			defer wg.Done()
			producer.Pause()
		}(producer)
	}
	wg.Wait()

	// Each pause reaches its consumers as a worker notification, so there is no
	// bound on how long it takes.
	require.Eventually(t, func() bool {
		for _, consumer := range consumers {
			if !consumer.ProducerPaused() {
				return false
			}
		}
		return true
	}, notificationTimeout, 5*time.Millisecond, "not every consumer saw its producer pause")

	messagesReceived := map[string][]string{}
	for _, dataConsumer := range dataConsumers {
		dataProducerId := dataConsumer.DataProducerId()
		dataConsumer.OnMessage(func(payload []byte, ppid SctpPayloadType) {
			mu.Lock()
			messagesReceived[dataProducerId] = append(messagesReceived[dataProducerId], string(payload))
			mu.Unlock()
		})
	}

	messagesSent := map[string]string{}
	for i, dataProducer := range dataProducers {
		wg.Add(1)
		go func(index int, dataProducer *DataProducer) {
			defer wg.Done()
			msg := fmt.Sprintf("hello world %d", index)
			dataProducer.SendText(msg)
			mu.Lock()
			messagesSent[dataProducer.Id()] = msg
			mu.Unlock()
		}(i, dataProducer)
	}

	wg.Wait()

	// Only the direct transports deliver: the others carry data over SCTP, and
	// these transports were never connected to a remote peer, so nothing sent on
	// them arrives. Each delivering dataProducer reaches its n dataConsumers.
	deliveringTransports := 0
	for _, transport := range transports {
		if transport.Type() == TransportDirect {
			deliveringTransports++
		}
	}

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()

		if len(messagesReceived) != deliveringTransports {
			return false
		}
		for _, messages := range messagesReceived {
			if len(messages) != n {
				return false
			}
		}
		return true
	}, notificationTimeout, 5*time.Millisecond, "not every data message came back")

	worker.Close()

	for dataProducerId, messages := range messagesReceived {
		require.Len(t, messages, n, dataProducerId)
		require.Equal(t, messagesSent[dataProducerId], messages[rand.Intn(len(messages))], dataProducerId)
	}

	assert.True(t, worker.Closed())

	for _, router := range routers {
		assert.True(t, router.Closed())
	}
	for _, transport := range transports {
		assert.True(t, transport.Closed())
	}
	for _, producer := range producers {
		assert.True(t, producer.Closed())
	}
	for _, dataProducer := range dataProducers {
		assert.True(t, dataProducer.Closed())
	}
	for _, consumer := range consumers {
		assert.True(t, consumer.Closed())
	}
	for _, dataConsumer := range dataConsumers {
		assert.True(t, dataConsumer.Closed())
	}

	// The goroutines serving each object wind down once the worker is gone. This
	// has to be polled from this goroutine: assert.Eventually would run the check
	// on a goroutine of its own and count it.
	waitUntil(t, func() bool {
		return runtime.NumGoroutine() <= numOfGoroutines
	}, 10*time.Second, "the per-object goroutines to finish")
}
