package mediasoup

import (
	"math/rand"
	"slices"
	"sync"
)

// Scheduler decides which worker of a WorkerPool a new router is created on.
//
// Pick receives the pool's live workers in pool order, never an empty slice, and
// runs with no pool lock held, so it may inspect the workers it is given. It is on
// the path of every WorkerPool.CreateRouter call and should stay cheap: read what
// the application already tracks rather than asking the subprocess through
// Worker.GetResourceUsage.
//
// Returning nil makes WorkerPool.CreateRouter fail with ErrNoWorkerAvailable.
type Scheduler interface {
	Pick(candidates []*Worker) *Worker
}

// SchedulerFunc adapts an ordinary function to Scheduler.
type SchedulerFunc func(candidates []*Worker) *Worker

func (f SchedulerFunc) Pick(candidates []*Worker) *Worker {
	return f(candidates)
}

// RoundRobin hands out each worker in turn. It is the default, and it spreads
// routers evenly while knowing nothing about how expensive each one turns out to
// be. Use LeastLoaded when the rooms differ widely in size.
func RoundRobin() Scheduler {
	return &roundRobin{}
}

type roundRobin struct {
	mu   sync.Mutex
	last *Worker
}

func (r *roundRobin) Pick(candidates []*Worker) *Worker {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Resuming from the last worker by identity rather than by index: the candidate
	// set shrinks as workers die, and an index into it would silently skip whoever
	// moved up into that position.
	next := 0
	if i := slices.Index(candidates, r.last); i >= 0 {
		next = (i + 1) % len(candidates)
	}
	r.last = candidates[next]

	return r.last
}

// Random picks a worker uniformly at random. Over many rooms this spreads them
// about as evenly as RoundRobin, without any shared state to serialise on.
func Random() Scheduler {
	return SchedulerFunc(func(candidates []*Worker) *Worker {
		return candidates[rand.Intn(len(candidates))]
	})
}

// LeastLoaded picks the least loaded worker, breaking ties towards the earliest
// candidate in pool order.
//
// A nil load falls back to how many producers and consumers the worker is
// carrying, which is what its capacity is measured in, regardless of how many
// routers those streams are spread over. Data channels are left out: SCTP costs
// far less per object than forwarding RTP does.
//
// Pass a load of your own where that is the wrong weighting: to count data
// channels too, to weigh simulcast producers above audio ones, or to use a room
// size the application already tracks.
//
// load runs once per candidate on every WorkerPool.CreateRouter call, so it must
// be cheap and must not block. In particular it must not call
// Worker.GetResourceUsage, which is a round trip to the subprocess.
func LeastLoaded(load func(*Worker) float64) Scheduler {
	if load == nil {
		load = func(worker *Worker) float64 {
			return float64(worker.objectCounts().rtpStreams())
		}
	}

	return SchedulerFunc(func(candidates []*Worker) *Worker {
		best, bestLoad := candidates[0], load(candidates[0])

		for _, worker := range candidates[1:] {
			if workerLoad := load(worker); workerLoad < bestLoad {
				best, bestLoad = worker, workerLoad
			}
		}

		return best
	})
}
