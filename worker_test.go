package mediasoup

import (
	"net"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

var WorkerBinPath = os.Getenv("WORKER_BIN")

func init() {
	if len(WorkerBinPath) == 0 {
		WorkerBinPath = "../mediasoup/worker/out/Debug/mediasoup-worker"
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

func TestWorkerCreateWebRtcServer(t *testing.T) {
	worker := newTestWorker()
	server, err := worker.CreateWebRtcServer(&WebRtcServerOptions{
		ListenInfos: []*TransportListenInfo{
			{Protocol: TransportProtocolUDP, IP: "127.0.0.1", Port: pickUdpPort()},
			{Protocol: TransportProtocolTCP, IP: "127.0.0.1", AnnouncedAddress: "foo.bar.org", Port: pickTcpPort()},
		},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, server)
	dump, _ := worker.Dump()
	assert.Contains(t, dump.WebRtcServerIds, server.Id())
}

func TestWorkerCreateRouter(t *testing.T) {
	worker := newTestWorker()
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
	worker := newTestWorker()
	worker.Close()
	assert.True(t, worker.Closed())
}

func TestWorkerNoGoroutineLeaks(t *testing.T) {
	defer goleak.VerifyNone(t)

	worker := newTestWorker(func(s *WorkerSettings) {
		s.LogLevel = WorkerLogLevelWarn
	})

	server, _ := worker.CreateWebRtcServer(&WebRtcServerOptions{
		ListenInfos: []*TransportListenInfo{
			{Protocol: TransportProtocolUDP, IP: "127.0.0.1", Port: 0},
			{Protocol: TransportProtocolTCP, IP: "127.0.0.1", AnnouncedAddress: "foo.bar.org", Port: 0},
		},
	})

	n := 10
	var mu sync.Mutex
	var wg sync.WaitGroup
	router1 := createRouter(worker)
	router2 := createRouter(worker)

	var transports []*Transport

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, router := range []*Router{router1, router2} {
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
		go func() {
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
		}()
	}

	wg.Wait()

	for i, producer := range producers {
		if i%3 == 0 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				producer.Pause()
			}()
		}
	}
	for _, dataProducer := range dataProducers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dataProducer.SendText("hello")
		}()
	}

	wg.Wait()
	worker.Close()

	assert.True(t, worker.Closed())

	for _, router := range []*Router{router1, router2} {
		router.Close()
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
}
