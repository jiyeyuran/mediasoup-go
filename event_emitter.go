package mediasoup

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
)

const EventEmitterQueueSize = 128

type IEventEmitter interface {
	AddListener(evt string, listener interface{})
	Once(evt string, listener interface{})
	Emit(evt string, argv ...interface{})
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
	Once      *sync.Once
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
	}

	if once {
		l.Once = &sync.Once{}
	}

	go func() {
		for args := range l.ArgValues {
			if args == nil {
				continue
			}
			if l.Once != nil {
				l.Once.Do(func() {
					listenerValue.Call(args)
				})
			} else {
				listenerValue.Call(args)
			}
		}
	}()

	return l
}

func (l intervalListener) TryUnmarshalArguments(args []reflect.Value) []reflect.Value {
	var actualArgs []reflect.Value

	for i, arg := range args {
		// Unmarshal bytes to golang type
		if isBytesType(arg.Type()) && !isBytesType(l.ArgTypes[i]) {
			val := reflect.New(l.ArgTypes[i]).Interface()
			if err := json.Unmarshal(arg.Bytes(), val); err == nil {
				if actualArgs == nil {
					actualArgs = make([]reflect.Value, len(args))
					copy(actualArgs, args)
				}
				actualArgs[i] = reflect.ValueOf(val).Elem()
			}
		}
	}

	if actualArgs != nil {
		return actualArgs
	}

	return args
}

func (l intervalListener) AlignArguments(args []reflect.Value) (actualArgs []reflect.Value) {
	// delete unwanted arguments
	if argLen := len(l.ArgTypes); len(args) >= argLen {
		actualArgs = args[0:argLen]
	} else {
		actualArgs = args[:]

		// append missing arguments with zero value
		for _, argType := range l.ArgTypes[len(args):] {
			actualArgs = append(actualArgs, reflect.Zero(argType))
		}
	}

	return actualArgs
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
func (e *EventEmitter) Emit(evt string, args ...interface{}) {
	e.emit(evt, true, args...)
}

// SafaEmit fires a particular event asynchronously.
func (e *EventEmitter) SafeEmit(evt string, args ...interface{}) {
	e.emit(evt, false, args...)
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

func (e *EventEmitter) emit(evt string, sync bool, args ...interface{}) {
	e.mu.Lock()

	if e.evtListeners == nil {
		e.mu.Unlock()
		return // has no listeners to emit yet
	}
	listeners := e.evtListeners[evt][:]
	e.mu.Unlock()

	callArgs := make([]reflect.Value, 0, len(args))

	for _, arg := range args {
		callArgs = append(callArgs, reflect.ValueOf(arg))
	}

	for _, listener := range listeners {
		if !listener.FuncValue.Type().IsVariadic() {
			callArgs = listener.AlignArguments(callArgs)
		}
		if actualArgs := listener.TryUnmarshalArguments(callArgs); sync {
			if listener.Once != nil {
				listener.Once.Do(func() {
					listener.FuncValue.Call(actualArgs)
				})
			} else {
				listener.FuncValue.Call(actualArgs)
			}
		} else {
			listener.ArgValues <- actualArgs
		}
		if listener.Once != nil {
			e.RemoveListener(evt, listener)
		}
	}
}

func isValidListener(fn interface{}) error {
	if reflect.TypeOf(fn).Kind() != reflect.Func {
		return fmt.Errorf("%s is not a reflect.Func", reflect.TypeOf(fn))
	}

	return nil
}

func isBytesType(tp reflect.Type) bool {
	return tp.Kind() == reflect.Slice && tp.Elem().Kind() == reflect.Uint8
}
