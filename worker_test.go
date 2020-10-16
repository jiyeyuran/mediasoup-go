package mediasoup

import (
	"encoding/json"
	"os"
	"time"
)

func init() {
	os.Setenv("DEBUG_COLORS", "false")
}

func CreateTestWorker(options ...Option) *Worker {
	options = append([]Option{WithLogLevel("debug"), WithLogTags([]WorkerLogTag{"info"})}, options...)

	worker, err := NewWorker(options...)
	if err != nil {
		panic(err)
	}
	return worker
}

func JSONUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func Wait(d time.Duration) {
	time.Sleep(d)
}
