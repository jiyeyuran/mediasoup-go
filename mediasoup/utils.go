package mediasoup

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func generateRandomNumber() uint32 {
	return uint32(rand.Int63n(900000000)) + 100000000
}

func newBool(b bool) *bool {
	return &b
}
