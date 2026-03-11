package mediasoup

const (
	RouterIDPrefix       = "rt_"
	TransportIDPrefix    = "tr_"
	ProducerIDPrefix     = "pr_"
	ConsumerIDPrefix     = "co_"
	DataProducerIDPrefix = "dp_"
	DataConsumerIDPrefix = "dc_"
	WebRtcServerPrefix   = "ws_"
	RtpObserverIDPrefix  = "ro_"
)

// UUID is an alias for the uuid package's UUID type.
// It is used throughout the mediasoup package for unique identifiers.
// Users can replace this with their own UUID implementation if needed.
var UUID = uuid

// H is a type alias for map[string]any, specifically used as AppData throughout the mediasoup package.
// It represents arbitrary application-specific data that can be attached to mediasoup objects.
// Values are recommended to be JSON marshallable to ensure proper serialization.
type H = map[string]any

// KeyValue represents a key-value pair.
type KeyValue[K comparable, V any] struct {
	Key   K
	Value V
}

// KeyValues represents a key-values pair.
type KeyValues[K comparable, V any] struct {
	Key    K
	Values []V
}
