package mediasoup

import (
	"encoding/json"
	"math/rand"
	"sync"
	"time"
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

func syncMapLen(m *sync.Map) (len int) {
	m.Range(func(key, val interface{}) bool {
		len++
		return true
	})
	return
}
