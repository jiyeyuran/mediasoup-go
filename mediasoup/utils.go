package mediasoup

import (
	"math/rand"
	"reflect"
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

func isObject(appData interface{}) bool {
	appDataKind := reflect.Indirect(reflect.ValueOf(appData)).Type().Kind()

	return appDataKind == reflect.Struct || appDataKind == reflect.Map
}
