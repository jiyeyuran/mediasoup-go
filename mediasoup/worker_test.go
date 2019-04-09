package mediasoup

import (
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var worker *Worker

func init() {
	workerBin := os.Getenv("MEDIASOUP_WORKER_BIN")

	if len(workerBin) == 0 {
		os.Setenv("MEDIASOUP_WORKER_BIN", "../mediasoup-worker")
	}
}

func TestMain(m *testing.M) {
	worker = CreateTestWorker()

	m.Run()

	worker.Close()
}

func CreateTestWorker(options ...Option) *Worker {
	options = append([]Option{WithLogLevel("warn")}, options...)

	worker, err := CreateWorker("", options...)
	if err != nil {
		panic(err)
	}

	return worker
}

func TestCreateWorker_Succeeds(t *testing.T) {
	worker := CreateTestWorker()
	assert.Greater(t, worker.Pid(), 0)
	assert.False(t, worker.Closed())

	worker.Close()
	assert.True(t, worker.Closed())

	worker = CreateTestWorker(
		WithLogLevel("debug"),
		WithLogTags([]string{"info"}),
		WithRTCMinPort(0),
		WithRTCMaxPort(9999),
		WithDTLSCert("testdata/dtls-cert.pem", "testdata/dtls-key.pem"),
	)
	assert.Greater(t, worker.Pid(), 0)
	assert.False(t, worker.Closed())

	worker.Close()
	assert.True(t, worker.Closed())
}

func TestCreateWorker_TypeError(t *testing.T) {
	_, err := CreateWorker("", WithLogLevel("chicken"))
	assert.IsType(t, err, NewTypeError(""))

	_, err = CreateWorker("", WithRTCMinPort(1000), WithRTCMaxPort(999))
	assert.IsType(t, err, NewTypeError(""))

	_, err = CreateWorker("", WithDTLSCert("notfuond/dtls-cert.pem", "testdata/dtls-key.pem"))
	assert.IsType(t, err, NewTypeError(""))

	_, err = CreateWorker("", WithDTLSCert("notfuond/dtls-cert.pem", "notfuond/dtls-key.pem"))
	assert.IsType(t, err, NewTypeError(""))
}

func TestWorkerUpdateSettings_Succeeds(t *testing.T) {
	worker := CreateTestWorker()
	resp := worker.UpdateSettings(Options{LogLevel: "debug", LogTags: []string{"ice"}})

	assert.NoError(t, resp.Err())

	worker.Close()
}

func TestWorkerUpdateSettings_TypeError(t *testing.T) {
	worker := CreateTestWorker()
	resp := worker.UpdateSettings(Options{LogLevel: "chicken"})

	assert.IsType(t, resp.Err(), NewTypeError(""))

	worker.Close()
}

func TestWorkerUpdateSettings_InvalidStateError(t *testing.T) {
	worker := CreateTestWorker()
	worker.Close()

	resp := worker.UpdateSettings(Options{LogLevel: "error"})

	assert.IsType(t, resp.Err(), NewInvalidStateError(""))

	worker.Close()
}

func TestWorkerDump(t *testing.T) {
	worker := CreateTestWorker()

	resp := worker.Dump()

	assert.JSONEq(
		t,
		string(resp.Data()),
		fmt.Sprintf(`{ "pid": %d, "routerIds": [] }`, worker.Pid()),
	)

	worker.Close()
}

func TestWorkerDump_InvalidStateError(t *testing.T) {
	worker := CreateTestWorker()
	worker.Close()

	resp := worker.Dump()
	assert.IsType(t, resp.Err(), NewInvalidStateError(""))

	worker.Close()
}

func TestWorkerClose_Succeeds(t *testing.T) {
	worker := CreateTestWorker(WithLogLevel("warn"))

	called := 0
	worker.Observer().Once("close", func() { called++ })

	worker.Close()

	assert.Equal(t, 1, called)
	assert.True(t, worker.Closed())
}

func TestWorkerEmitsDied(t *testing.T) {
	signals := []os.Signal{os.Interrupt, syscall.SIGTERM, os.Kill}

	for _, signal := range signals {
		worker := CreateTestWorker(WithLogLevel("warn"))

		nextCh := make(chan struct{})
		called := 0
		worker.Observer().Once("close", func() { called++ })

		process, err := os.FindProcess(worker.Pid())
		assert.NoError(t, err)

		worker.On("died", func() { close(nextCh) })
		process.Signal(signal)

		timer := time.NewTimer(time.Second)
		select {
		case <-nextCh:
		case <-timer.C:
			assert.FailNow(t, "timeout")
		}

		assert.Equal(t, 1, called)
		assert.True(t, worker.Closed())
	}
}

func TestWorkerProcessIgnoreSignals(t *testing.T) {
	worker := CreateTestWorker(WithLogLevel("warn"))

	nextCh := make(chan struct{})

	worker.On("died", func() { close(nextCh) })

	process, err := os.FindProcess(worker.Pid())
	assert.NoError(t, err)

	process.Signal(syscall.SIGPIPE)
	process.Signal(syscall.SIGHUP)
	process.Signal(syscall.SIGALRM)
	process.Signal(syscall.SIGUSR1)
	process.Signal(syscall.SIGUSR2)

	timer := time.NewTimer(time.Second)
	select {
	case <-nextCh:
	case <-timer.C:
	}

	assert.False(t, worker.Closed())
	worker.Close()
}
