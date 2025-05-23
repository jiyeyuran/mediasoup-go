package mediasoup

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var WorkerBinPath = os.Getenv("MEDIASOUP_WORKER_BIN")

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
	worker := newTestWorker()
	worker.Close()
	assert.True(t, worker.Closed())
	assert.NoError(t, worker.Err())

	worker = newTestWorker()
	process, _ := os.FindProcess(worker.Pid())
	process.Kill()
	time.Sleep(100 * time.Millisecond)
	assert.True(t, worker.Closed())
	assert.Error(t, worker.Err())
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

	// wait all notifications are consumed
	time.Sleep(time.Millisecond * 10)

	for _, consumer := range consumers {
		assert.True(t, consumer.ProducerPaused())
	}

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

	// wait all notifications are consumed
	time.Sleep(time.Millisecond * 10)

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

	// wait all goroutines are finished
	time.Sleep(time.Millisecond * 500)

	assert.LessOrEqual(t, runtime.NumGoroutine(), numOfGoroutines)
}
