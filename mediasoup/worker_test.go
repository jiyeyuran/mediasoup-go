package mediasoup

import (
	"os"
	"testing"
)

var (
	worker      *Worker
	beforeEachs []func()
)

func init() {
	// logger := Logger()
	// logger.SetLevel(logrus.DebugLevel)
}

func AddBeforeEach(beforeEach func()) {
	beforeEachs = append(beforeEachs, beforeEach)
}

func setupWorker() {
	workerBin := os.Getenv("MEDIASOUP_WORKER_BIN")

	if len(workerBin) == 0 {
		os.Setenv("MEDIASOUP_WORKER_BIN", "../mediasoup-worker")
	}

	w, err := NewWorker("", WithLogLevel("warn"))
	if err != nil {
		panic(err)
	}

	worker = w
}

func closeWorker() {
	worker.Close()
}

func TestMain(m *testing.M) {
	setupWorker()
	for _, each := range beforeEachs {
		each()
	}

	code := m.Run()

	closeWorker()

	if code != 0 {
		os.Exit(code)
	}
}
