package mediasoup

import (
	"encoding/json"
	"math/rand"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/imdario/mergo"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type H map[string]interface{}

func Bool(b bool) *bool {
	return &b
}

func generateRandomNumber() uint32 {
	return uint32(rand.Int63n(900000000)) + 100000000
}

func clone(from, to interface{}) (err error) {
	data, err := json.Marshal(from)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, to)
}

func override(dst, src interface{}) error {
	return mergo.Merge(dst, src,
		mergo.WithOverride,
		mergo.WithTypeCheck,
		mergo.WithTransformers(ptrTransformers{}),
	)
}

func syncMapLen(m sync.Map) (len uint32) {
	m.Range(func(key, val interface{}) bool {
		atomic.AddUint32(&len, 1)
		return true
	})
	return
}

type ptrTransformers struct{}

// overwrites pointer type
func (ptrTransformers) Transformer(tp reflect.Type) func(dst, src reflect.Value) error {
	if tp.Kind() == reflect.Ptr {
		return func(dst, src reflect.Value) error {
			if !src.IsNil() {
				if dst.CanSet() {
					dst.Set(src)
				} else {
					dst = src
				}
			}
			return nil
		}
	}
	return nil
}
