package mediasoup

import (
	"math/rand"
	"reflect"
	"sync"
	"time"
	"unsafe"

	"github.com/rs/xid"
)

const (
	charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	routerPrefix       = "rt-"
	transportPrefix    = "tr-"
	producerPrefix     = "pr-"
	consumerPrefix     = "co-"
	dataProducerPrefix = "dp-"
	dataConsumerPrefix = "dc-"
	webRtcServerPrefix = "ws-"
	rtpObserverPrefix  = "ro-"
)

func ref[T any](t T) *T {
	return &t
}

func unref[T any](t *T, defaultValue ...T) T {
	if t != nil {
		return *t
	}
	var zero T
	if len(defaultValue) > 0 {
		zero = defaultValue[0]
	}
	return zero
}

func ifElse[T any](pref bool, trueFn func() T, falseFn ...func() T) T {
	if pref {
		return trueFn()
	}
	if len(falseFn) > 0 {
		return falseFn[0]()
	}
	var zero T
	return zero
}

func orElse[T any](pred bool, a, b T) T {
	if pred {
		return a
	}
	return b
}

func safeCall[T any](fn func(T), v T) {
	if fn != nil {
		fn(v)
	}
}

func collect[Slice ~[]E, E any, R any](s Slice, f func(E) R) []R {
	results := make([]R, len(s))
	for i, e := range s {
		results[i] = f(e)
	}
	return results
}

func filter[Slice ~[]E, E any](s Slice, f func(E) bool) []E {
	results := make([]E, 0, len(s))
	for _, e := range s {
		if f(e) {
			results = append(results, e)
		}
	}
	return results
}

func uuid(prefix string) string {
	return prefix + xid.New().String()
}

func randString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func generateSsrc() uint32 {
	return uint32(rand.Int63n(900000000)) + 100000000
}

// clone creates a deep copy of whatever is passed to it and returns the copy
// in an any.  The returned value will need to be asserted to the
// correct type.
func clone[T any](src T) T {
	// Make the interface a reflect.Value
	original := reflect.ValueOf(src)

	// If src is a nil pointer, return a zero value.
	if !original.IsValid() || (original.Kind() == reflect.Ptr && original.IsNil()) {
		var zero T
		return zero
	}

	// Make a copy of the same type as the original.
	cpy := reflect.New(original.Type()).Elem()

	// Recursively copy the original.
	copyRecursive(original, cpy)

	// Return the copy as an interface.
	return cpy.Interface().(T)
}

// copyRecursive does the actual copying of the interface. It currently has
// limited support for what it can handle. Add as needed.
func copyRecursive(original, cpy reflect.Value) {
	// handle according to original's Kind
	switch original.Kind() {
	case reflect.Ptr:
		// Get the actual value being pointed to.
		originalValue := original.Elem()

		// if  it isn't valid, return.
		if !originalValue.IsValid() {
			return
		}
		cpy.Set(reflect.New(originalValue.Type()))
		copyRecursive(originalValue, cpy.Elem())

	case reflect.Interface:
		// If this is a nil, don't do anything
		if original.IsNil() {
			return
		}
		// Get the value for the interface, not the pointer.
		originalValue := original.Elem()

		// Get the value by calling Elem().
		copyValue := reflect.New(originalValue.Type()).Elem()
		copyRecursive(originalValue, copyValue)
		cpy.Set(copyValue)

	case reflect.Struct:
		t, ok := original.Interface().(time.Time)
		if ok {
			cpy.Set(reflect.ValueOf(t))
			return
		}
		// Go through each field of the struct and copy it.
		for i := 0; i < original.NumField(); i++ {
			// The Type's StructField for a given field is checked to see if StructField.PkgPath
			// is set to determine if the field is exported or not because CanSet() returns false
			// for settable fields.  I'm not sure why.  -mohae
			if original.Type().Field(i).PkgPath != "" {
				continue
			}
			copyRecursive(original.Field(i), cpy.Field(i))
		}

	case reflect.Slice:
		if original.IsNil() {
			return
		}
		// Make a new slice and copy each element.
		cpy.Set(reflect.MakeSlice(original.Type(), original.Len(), original.Cap()))
		for i := 0; i < original.Len(); i++ {
			copyRecursive(original.Index(i), cpy.Index(i))
		}

	case reflect.Array:
		for i := 0; i < original.Len(); i++ {
			copyRecursive(original.Index(i), cpy.Index(i))
		}

	case reflect.Map:
		if original.IsNil() {
			return
		}
		cpy.Set(reflect.MakeMap(original.Type()))
		for _, key := range original.MapKeys() {
			originalValue := original.MapIndex(key)
			copyValue := reflect.New(originalValue.Type()).Elem()
			copyRecursive(originalValue, copyValue)
			cpy.SetMapIndex(key, copyValue)
		}

	default:
		cpy.Set(original)
	}
}

func clearSyncMap(m *sync.Map) {
	m.Range(func(key, value any) bool {
		m.Delete(key)
		return true
	})
}
