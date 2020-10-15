package mediasoup

import (
	"math/rand"
	"time"

	"github.com/jinzhu/copier"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func generateRandomNumber() uint32 {
	return uint32(rand.Int63n(900000000)) + 100000000
}

func clone(from, to interface{}) (err error) {
	copier.Copy(to, from)

	return
}
