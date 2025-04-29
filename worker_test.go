package mediasoup

import (
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			o.LogTags = []WorkerLogTag{
				WorkerLogTagInfo,
			}
			o.Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelDebug,
				ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
					if a.Key == slog.SourceKey {
						source := a.Value.Any().(*slog.Source)
						source.File = filepath.Base(source.File)
					}
					return a
				},
			}))
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
