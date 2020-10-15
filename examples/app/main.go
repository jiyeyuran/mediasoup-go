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
	worker.On("@failure", func(err error) {
		logger.Error("%s", err)
	})
	worker.On("died", func(err error) {
		logger.Error("%s", err)
	})
	worker.On("@success", func() {
		dump, err := worker.Dump()
		if err != nil {
			panic(err)
		}
		logger.Debug("dump: %s", dump)

		usage, err := worker.GetResourceUsage()
		if err != nil {
			panic(err)
		}
		data, _ := json.Marshal(usage)
		logger.Debug("usage: %s", data)
	})

	wait := make(chan struct{})
	<-wait
}
