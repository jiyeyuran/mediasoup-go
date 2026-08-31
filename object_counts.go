package mediasoup

import "sync/atomic"

// objectCounts is how many objects of each kind are alive under a Router or a
// Worker. It is not exported: a scheduler only needs a single number, and the
// application already knows what it created.
type objectCounts struct {
	producers     int
	consumers     int
	dataProducers int
	dataConsumers int
}

// rtpStreams is producers plus consumers, the figure a worker's capacity is
// measured in. Data channel objects are left out: SCTP costs far less per object
// than forwarding RTP does.
func (c objectCounts) rtpStreams() int {
	return c.producers + c.consumers
}

func (c *objectCounts) add(other objectCounts) {
	c.producers += other.producers
	c.consumers += other.consumers
	c.dataProducers += other.dataProducers
	c.dataConsumers += other.dataConsumers
}

// objectCounters tracks the sizes of the sync.Maps a Router indexes its objects by,
// which have no length of their own.
type objectCounters struct {
	producers     atomic.Int64
	consumers     atomic.Int64
	dataProducers atomic.Int64
	dataConsumers atomic.Int64
}

func (c *objectCounters) load() objectCounts {
	return objectCounts{
		producers:     int(c.producers.Load()),
		consumers:     int(c.consumers.Load()),
		dataProducers: int(c.dataProducers.Load()),
		dataConsumers: int(c.dataConsumers.Load()),
	}
}
