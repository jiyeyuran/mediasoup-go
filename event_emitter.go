package mediasoup

import (
	"encoding/json"
	"fmt"
	"reflect"
	"runtime/debug"
	"sync"

	"github.com/go-logr/logr"
)

// EventEmitter defines an interface of the Event-based architecture(like EventEmitter in JavaScript).
type IEventEmitter interface {
	// On adds the listener function to the end of the listeners array for the event named eventName.
	// No checks are made to see if the listener has already been added.
	// Multiple calls passing the same combination of eventName and listener will result in the listener
	// being added, and called, multiple times.
	// By default, a maximum of 10 listeners can be registered for any single event.
	// This is a useful default that helps finding memory leaks. Note that this is not a hard limit.
	// The EventEmitter instance will allow more listeners to be added but will output a trace warning
	// to log indicating that a "possible EventEmitter memory leak" has been detected.
	On(eventName string, listener interface{})

	// Once adds a one-time listener function for the event named eventName.
	// The next time eventName is triggered, this listener is removed and then invoked.
	Once(eventName string, listener interface{})

	// Emit calls each of the listeners registered for the event named eventName,
	// in the order they were registered, passing the supplied arguments to each.
	// Returns true if the event had listeners, false otherwise.
	Emit(eventName string, argv ...interface{}) bool

	// SafeEmit calls each of the listeners registered for the event named eventName. It recovers
	// panic and logs panic info with provided logger.
	SafeEmit(eventName string, argv ...interface{}) bool

	// Off removes the specified listener from the listener array for the event named eventName.
	Off(eventName string, listener interface{})

	// RemoveAllListeners removes all listeners, or those of the specified eventNames.
	RemoveAllListeners(eventNames ...string)
}

type EventEmitter struct {
	mu        sync.Mutex
	listeners map[string][]*intervalListener
	logger    logr.Logger
}

func NewEventEmitter() IEventEmitter {
	return &EventEmitter{
		logger: NewLogger("EventEmitter"),
	}
}

func (e *EventEmitter) On(event string, listener interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.listeners == nil {
		e.listeners = make(map[string][]*intervalListener)
	}
	e.listeners[event] = append(e.listeners[event], newInternalListener(listener, false))
}

func (e *EventEmitter) Once(event string, listener interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.listeners == nil {
		e.listeners = make(map[string][]*intervalListener)
	}
	e.listeners[event] = append(e.listeners[event], newInternalListener(listener, true))
}

func (e *EventEmitter) Emit(event string, args ...interface{}) bool {
	e.mu.Lock()
	if e.listeners == nil {
		e.mu.Unlock()
		return false
	}
	listeners := e.listeners[event]
	e.mu.Unlock()

	for _, listener := range listeners {
		if listener.once != nil {
			e.Off(event, listener.listenerValue.Interface())
		}
		// may panic
		listener.Call(args...)
	}
	return len(listeners) > 0
}

func (e *EventEmitter) SafeEmit(event string, args ...interface{}) bool {
	defer func() {
		if r := recover(); r != nil {
			e.logger.Error(fmt.Errorf("%v", r), "emit panic", "stack", debug.Stack())
		}
	}()

	return e.Emit(event, args...)
}

func (e *EventEmitter) Off(event string, listener interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.listeners == nil {
		return
	}
	listeners := e.listeners[event]
	handlerPtr := reflect.ValueOf(listener).Pointer()

	for i, internalListener := range listeners {
		if internalListener.listenerValue.Pointer() == handlerPtr {
			e.listeners[event] = append(listeners[0:i], listeners[i+1:]...)
			break
		}
	}
}

func (e *EventEmitter) RemoveAllListeners(events ...string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.listeners == nil {
		return
	}
	if len(events) == 0 {
		e.listeners = nil
		return
	}
	for _, event := range events {
		delete(e.listeners, event)
	}
}

type intervalListener struct {
	listenerValue reflect.Value
	argTypes      []reflect.Type
	once          *sync.Once
}

func newInternalListener(listener interface{}, once bool) *intervalListener {
	var argTypes []reflect.Type
	listenerValue := reflect.ValueOf(listener)
	listenerType := listenerValue.Type()

	for i := 0; i < listenerType.NumIn(); i++ {
		argTypes = append(argTypes, listenerType.In(i))
	}

	l := &intervalListener{
		listenerValue: listenerValue,
		argTypes:      argTypes,
	}
	if once {
		l.once = &sync.Once{}
	}

	return l
}

func (l *intervalListener) Call(args ...interface{}) {
	call := func() {
		argValues := make([]reflect.Value, len(args))
		for i, arg := range args {
			argValues[i] = reflect.ValueOf(arg)
		}
		if !l.listenerValue.Type().IsVariadic() {
			argValues = l.alignArguments(argValues)
		}
		argValues = l.convertArguments(argValues)

		// call listener function and ignore returns
		l.listenerValue.Call(argValues)
	}

	if l.once != nil {
		l.once.Do(call)
	} else {
		call()
	}
}

func (l intervalListener) convertArguments(args []reflect.Value) []reflect.Value {
	if len(args) != len(l.argTypes) {
		return args
	}
	actualArgs := make([]reflect.Value, len(args))

	for i, arg := range args {
		// Unmarshal bytes to golang type
		if isBytesType(arg.Type()) && !isBytesType(l.argTypes[i]) {
			val := reflect.New(l.argTypes[i]).Interface()
			if err := json.Unmarshal(arg.Bytes(), val); err == nil {
				actualArgs[i] = reflect.ValueOf(val).Elem()
			}
		} else if arg.Type() != l.argTypes[i] &&
			arg.Type().ConvertibleTo(l.argTypes[i]) {
			actualArgs[i] = arg.Convert(l.argTypes[i])
		} else {
			actualArgs[i] = arg
		}
	}

	return actualArgs
}

func (l intervalListener) alignArguments(args []reflect.Value) (actualArgs []reflect.Value) {
	// delete unwanted arguments
	if argLen := len(l.argTypes); len(args) >= argLen {
		actualArgs = args[0:argLen]
	} else {
		actualArgs = args[:]

		// append missing arguments with zero value
		for _, argType := range l.argTypes[len(args):] {
			actualArgs = append(actualArgs, reflect.Zero(argType))
		}
	}

	return actualArgs
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
