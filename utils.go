package mediasoup

import (
	"math/rand"
	"reflect"
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
	copier.Copy(&to, from)

	return
}

func newBool(b bool) *bool {
	return &b
}

func isObject(object interface{}) bool {
	valObject := reflect.ValueOf(object)

	for valObject.Kind() == reflect.Ptr {
		valObject = valObject.Elem()
	}

	objectKind := valObject.Type().Kind()

	return objectKind == reflect.Struct || objectKind == reflect.Map
}
