package mediasoup

import (
	"os"
)

func init() {
	workerBin := os.Getenv("MEDIASOUP_WORKER_BIN")

	if len(workerBin) == 0 {
		os.Setenv("MEDIASOUP_WORKER_BIN", "../mediasoup-worker")
	}
}
