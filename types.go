package mediasoup

var UUID = uuid

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
