package main

import (
	"encoding/json"

	"github.com/jiyeyuran/mediasoup-go"
)

var logger = mediasoup.NewLogger("ExampleApp")

func main() {
	worker, err := mediasoup.NewWorker()
	if err != nil {
		panic(err)
	}
	worker.On("died", func(err error) {
		logger.Error("%s", err)
	})

	dump, _ := worker.Dump()
	logger.Debug("dump: %+v", dump)

	usage, err := worker.GetResourceUsage()
	if err != nil {
		panic(err)
	}
	data, _ := json.Marshal(usage)
	logger.Debug("usage: %s", data)

	wait := make(chan struct{})
	<-wait
}
