package mediasoup

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func generateRandomNumber() int64 {
	return rand.Int63n(900000000) + 100000000
}
