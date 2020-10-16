package mediasoup

import (
	"encoding/json"
	"fmt"
	"reflect"
	"runtime/debug"
	"sync"
)

const EventEmitterQueueSize = 128

type IEventEmitter interface {
	AddListener(evt string, listener interface{})
	Once(evt string, listener interface{})
	Emit(evt string, argv ...interface{}) (err error)
	SafeEmit(evt string, argv ...interface{})
	RemoveListener(evt string, listener interface{}) (ok bool)
	RemoveAllListeners(evt string)
	On(evt string, listener interface{})
	Off(evt string, listener interface{})
	ListenerCount(evt string) int
	Len() int
}

type intervalListener struct {
	FuncValue reflect.Value
	ArgTypes  []reflect.Type
	ArgValues chan []reflect.Value
	Once      bool
}

func newInternalListener(listener interface{}, once bool) *intervalListener {
	var argTypes []reflect.Type
	listenerValue := reflect.ValueOf(listener)
	listenerType := listenerValue.Type()

	for i := 0; i < listenerType.NumIn(); i++ {
		argTypes = append(argTypes, listenerType.In(i))
	}

	l := &intervalListener{
		FuncValue: listenerValue,
		ArgTypes:  argTypes,
		ArgValues: make(chan []reflect.Value, EventEmitterQueueSize),
		Once:      once,
	}

	go func() {
		var syncOnce sync.Once

		for args := range l.ArgValues {
			actualArgs := make([]reflect.Value, len(args))

			for i, arg := range args {
				actualArgs[i] = arg

				// auto unmarshal json data to golang type
				if typeIsBytes(arg.Type()) && !typeIsBytes(argTypes[i]) {
					b, ok := arg.Interface().(json.RawMessage)
					if !ok {
						b, ok = arg.Interface().([]byte)
					}
					if ok {
						val := reflect.New(argTypes[i]).Interface()
						if err := json.Unmarshal(b, val); err == nil {
							actualArgs[i] = reflect.ValueOf(val).Elem()
						}
					}
				}
			}

			if once {
				syncOnce.Do(func() {
					listenerValue.Call(actualArgs)
				})
			} else {
				listenerValue.Call(actualArgs)
			}
		}
	}()

	return l
}

type EventEmitter struct {
	evtListeners map[string][]*intervalListener
	mu           sync.Mutex
}

func NewEventEmitter() IEventEmitter {
	return &EventEmitter{}
}

func (e *EventEmitter) AddListener(evt string, listener interface{}) {
	if err := isValidListener(listener); err != nil {
		panic(err)
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.evtListeners == nil {
		e.evtListeners = make(map[string][]*intervalListener)
	}
	e.evtListeners[evt] = append(e.evtListeners[evt], newInternalListener(listener, false))
}

func (e *EventEmitter) Once(evt string, listener interface{}) {
	if err := isValidListener(listener); err != nil {
		panic(err)
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.evtListeners == nil {
		e.evtListeners = make(map[string][]*intervalListener)
	}
	e.evtListeners[evt] = append(e.evtListeners[evt], newInternalListener(listener, true))
}

// Emit fires a particular event
func (e *EventEmitter) Emit(evt string, args ...interface{}) (err error) {
	e.mu.Lock()
	if e.evtListeners == nil {
		e.mu.Unlock()
		return // has no listeners to emit yet
	}
	listeners := e.evtListeners[evt][:]
	e.mu.Unlock()

	var callArgs []reflect.Value

	for _, arg := range args {
		callArgs = append(callArgs, reflect.ValueOf(arg))
	}

	for _, listener := range listeners {
		if listener.FuncValue.Type().IsVariadic() {
			listener.ArgValues <- callArgs
		} else {
			listener.ArgValues <- buildActualArgs(listener.ArgTypes, callArgs)
		}
		if listener.Once {
			e.RemoveListener(evt, listener)
		}
	}

	return
}

// SafaEmit fires a particular event and ignore panic.
func (e *EventEmitter) SafeEmit(evt string, argv ...interface{}) {
	defer func() {
		if r := recover(); r != nil {
			debug.PrintStack()
		}
	}()

	e.Emit(evt, argv...)
}

func (e *EventEmitter) RemoveListener(evt string, listener interface{}) (ok bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.evtListeners == nil || listener == nil {
		return
	}

	idx := -1
	pointer := reflect.ValueOf(listener).Pointer()
	listeners := e.evtListeners[evt]

	for index, item := range listeners {
		if listener == item || item.FuncValue.Pointer() == pointer {
			idx = index
			break
		}
	}

	if idx < 0 {
		return false
	}

	e.evtListeners[evt] = append(listeners[:idx], listeners[idx+1:]...)

	return true
}

func (e *EventEmitter) RemoveAllListeners(evt string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.evtListeners, evt)
}

func (e *EventEmitter) On(evt string, listener interface{}) {
	e.AddListener(evt, listener)
}

func (e *EventEmitter) Off(evt string, listener interface{}) {
	e.RemoveListener(evt, listener)
}

func (e *EventEmitter) ListenerCount(evt string) int {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.evtListeners == nil {
		return 0
	}

	return len(e.evtListeners[evt])
}

func (e *EventEmitter) Len() int {
	e.mu.Lock()
	defer e.mu.Unlock()

	return len(e.evtListeners)
}

func isValidListener(fn interface{}) error {
	if reflect.TypeOf(fn).Kind() != reflect.Func {
		return fmt.Errorf("%s is not a reflect.Func", reflect.TypeOf(fn))
	}

	return nil
}

func buildActualArgs(argTypes []reflect.Type, callArgs []reflect.Value) (reflectedArgs []reflect.Value) {
	// delete unwanted arguments
	if argLen := len(argTypes); len(callArgs) >= argLen {
		reflectedArgs = callArgs[0:argLen]
	} else {
		reflectedArgs = callArgs[:]

		// append missing arguments with zero value
		for _, argType := range argTypes[len(callArgs):] {
			reflectedArgs = append(reflectedArgs, reflect.Zero(argType))
		}
	}

	return reflectedArgs
}

func typeIsBytes(tp reflect.Type) bool {
	return tp.Kind() == reflect.Slice && tp.Elem().Kind() == reflect.Uint8
}
