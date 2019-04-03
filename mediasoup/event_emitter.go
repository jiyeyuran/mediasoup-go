package mediasoup

import (
	"reflect"
	"sync"

	"github.com/sirupsen/logrus"
)

type EventEmitter interface {
	AddListener(evt string, listeners ...interface{})
	Emit(evt string, argv ...interface{}) (err error)
	SafeEmit(evt string, argv ...interface{})
	RemoveListener(evt string, listener interface{}) (ok bool)
	On(evt string, listener ...interface{})
	Off(evt string, listener interface{})
	ListenerCount(evt string) int
	Len() int
}

type (
	intervalListener struct {
		Value reflect.Value
		Argv  []reflect.Type
	}

	eventEmitter struct {
		logger       logrus.FieldLogger
		evtListeners map[string][]intervalListener
		mu           sync.Mutex
	}
)

func NewEventEmitter(logger logrus.FieldLogger) EventEmitter {
	return &eventEmitter{
		logger: logger,
	}
}

func (e *eventEmitter) AddListener(evt string, listeners ...interface{}) {
	if len(listeners) == 0 {
		return
	}
	var listenerValues []intervalListener

	for _, listener := range listeners {
		listenerValue := reflect.ValueOf(listener)
		listenerType := listenerValue.Type()

		if listenerType.Kind() != reflect.Func {
			continue
		}
		var argv []reflect.Type

		for i := 0; i < listenerType.NumIn(); i++ {
			argv = append(argv, listenerType.In(i))
		}

		listenerValues = append(listenerValues, intervalListener{
			Value: listenerValue,
			Argv:  argv,
		})
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.evtListeners == nil {
		e.evtListeners = make(map[string][]intervalListener)
	}

	e.evtListeners[evt] = append(e.evtListeners[evt], listenerValues...)
}

// Emit fires a particular event
func (e *eventEmitter) Emit(evt string, argv ...interface{}) (err error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.evtListeners == nil {
		return // has no listeners to emit yet
	}
	var callArgv []reflect.Value

	for _, a := range argv {
		callArgv = append(callArgv, reflect.ValueOf(a))
	}

	for _, listener := range e.evtListeners[evt] {
		// delete unwanted arguments
		if len(callArgv) > len(listener.Argv) {
			callArgv = callArgv[0:len(listener.Argv)]
		}

		// append missing arguments with zero value
		if len(callArgv) < len(listener.Argv) {
			for _, a := range listener.Argv[len(callArgv):] {
				callArgv = append(callArgv, reflect.Zero(a))
			}
		}

		listener.Value.Call(callArgv)
	}

	return
}

// SafaEmit fires a particular event and ignore panic.
func (e *eventEmitter) SafeEmit(evt string, argv ...interface{}) {
	defer func() {
		if r := recover(); r != nil {
			e.logger.WithField("event", evt).Errorln(r)
		}
	}()

	e.Emit(evt, argv...)
}

func (e *eventEmitter) RemoveListener(evt string, listener interface{}) (ok bool) {
	if e.evtListeners == nil {
		return
	}

	if listener == nil {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	idx := -1
	listenerPointer := reflect.ValueOf(listener).Pointer()
	listeners := e.evtListeners[evt]

	for index, item := range listeners {
		if item.Value.Pointer() == listenerPointer {
			idx = index
			break
		}
	}

	if idx < 0 {
		return
	}

	var modifiedListeners []intervalListener

	if len(listeners) > 1 {
		modifiedListeners = append(listeners[:idx], listeners[idx+1:]...)
	}

	e.evtListeners[evt] = modifiedListeners

	return true
}

func (e *eventEmitter) On(evt string, listener ...interface{}) {
	e.AddListener(evt, listener...)
}

func (e *eventEmitter) Off(evt string, listener interface{}) {
	e.RemoveListener(evt, listener)
}

func (e *eventEmitter) ListenerCount(evt string) int {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.evtListeners == nil {
		return 0
	}

	return len(e.evtListeners[evt])
}

func (e *eventEmitter) Len() int {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.evtListeners == nil {
		return 0
	}
	return len(e.evtListeners)
}
