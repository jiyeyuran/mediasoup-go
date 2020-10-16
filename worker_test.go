package mediasoup

import (
	"encoding/json"
	"os"
	"time"
)

func init() {
	os.Setenv("DEBUG_COLORS", "false")
	DefaultLevel = WarnLevel
}

func CreateTestWorker(options ...Option) *Worker {
	options = append([]Option{WithLogLevel("debug"), WithLogTags([]WorkerLogTag{"info"})}, options...)

	worker, err := NewWorker(options...)
	if err != nil {
		panic(err)
	}
	return worker
}

func Wait(d time.Duration) {
	time.Sleep(d)
}

func MarshalString(v interface{}) string {
	data, _ := json.Marshal(v)

	return string(data)
}
