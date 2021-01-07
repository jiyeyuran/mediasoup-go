package mediasoup

import (
	"github.com/jiyeyuran/go-eventemitter"
)

type IEventEmitter = eventemitter.IEventEmitter

func NewEventEmitter() IEventEmitter {
	return eventemitter.NewEventEmitter(eventemitter.WithLogger(NewLogger("EventEmitter")))
}
