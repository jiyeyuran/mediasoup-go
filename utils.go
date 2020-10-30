package mediasoup

import (
	"encoding/json"
	"math/rand"
	"sync"
	"time"

	"github.com/imdario/mergo"
)

func init() {
	rand.Seed(time.Now().UnixNano())
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

func override(dst, src interface{}) {
	mergo.Merge(dst, src, mergo.WithOverride)
}

func syncMapLen(m *sync.Map) (len int) {
	m.Range(func(key, val interface{}) bool {
		len++
		return true
	})
	return
}
