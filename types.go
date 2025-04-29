package mediasoup

type H map[string]any

// KeyValue represents a key-value pair.
type KeyValue[K any, V any] struct {
	Key   K
	Value V
}

// KeyValues represents a key-values pair.
type KeyValues[K any, V any] struct {
	Key    K
	Values []V
}
