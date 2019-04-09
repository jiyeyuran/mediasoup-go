package mediasoup

import (
	"sync"
)

func CreateWorker(workerBin string, options ...Option) (worker *Worker, err error) {
	worker, err = newWorker(workerBin, options...)
	if err != nil {
		return
	}

	var wg sync.WaitGroup

	wg.Add(1)

	worker.On("@failure", func(errr error) {
		worker, err = nil, errr
		wg.Done()
	})

	worker.On("@success", func(worker *Worker) {
		wg.Done()
	})

	wg.Wait()

	return
}
