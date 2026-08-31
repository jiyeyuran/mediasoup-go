package mediasoup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The schedulers only choose between the workers they are handed, so they can be
// exercised without paying for a subprocess each.

func TestRoundRobinScheduler(t *testing.T) {
	workers := []*Worker{{}, {}, {}}
	scheduler := RoundRobin()

	// Two full cycles must visit every worker in order.
	for i := 0; i < len(workers)*2; i++ {
		assert.Same(t, workers[i%len(workers)], scheduler.Pick(workers), "cycle position %d", i)
	}
}

// The candidate set shrinks as workers die. Resuming by index into it would skip
// whoever moved up into the vacated position.
func TestRoundRobinSchedulerSurvivesShrinkingCandidates(t *testing.T) {
	first, second, third := &Worker{}, &Worker{}, &Worker{}
	all := []*Worker{first, second, third}
	scheduler := RoundRobin()

	require.Same(t, first, scheduler.Pick(all))
	require.Same(t, second, scheduler.Pick(all))

	// The worker picked last is gone, so there is no position to resume from and
	// the cycle starts over rather than losing track.
	assert.Same(t, first, scheduler.Pick([]*Worker{first, third}))
	assert.Same(t, third, scheduler.Pick([]*Worker{first, third}))
	assert.Same(t, first, scheduler.Pick([]*Worker{first, third}))
}

func TestRandomScheduler(t *testing.T) {
	workers := []*Worker{{}, {}, {}}
	scheduler := Random()

	picked := make(map[*Worker]bool)
	for i := 0; i < 300; i++ {
		worker := scheduler.Pick(workers)
		require.Contains(t, workers, worker)
		picked[worker] = true
	}

	// Three hundred draws across three workers: a worker missing here is one that
	// never gets a turn at all.
	assert.Len(t, picked, len(workers))
}

func TestLeastLoadedScheduler(t *testing.T) {
	idle, busy, busiest := &Worker{}, &Worker{}, &Worker{}
	load := map[*Worker]float64{idle: 0.2, busy: 0.7, busiest: 0.9}
	scheduler := LeastLoaded(func(worker *Worker) float64 {
		return load[worker]
	})

	candidates := []*Worker{busy, idle, busiest}
	assert.Same(t, idle, scheduler.Pick(candidates))

	// Load is read afresh on every pick, so the choice follows the workers as they
	// fill up rather than being decided once.
	load[idle] = 1.0
	assert.Same(t, busy, scheduler.Pick(candidates))
}

func TestLeastLoadedSchedulerBreaksTiesByPoolOrder(t *testing.T) {
	first, second := &Worker{}, &Worker{}
	scheduler := LeastLoaded(func(*Worker) float64 {
		return 0.5
	})

	// Every worker equally loaded: the earliest candidate wins, so an unloaded pool
	// fills up predictably instead of arbitrarily.
	assert.Same(t, first, scheduler.Pick([]*Worker{first, second}))
	assert.Same(t, first, scheduler.Pick([]*Worker{first, second}))
}
