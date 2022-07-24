package mediasoup

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type internalData struct {
	RouterId       string `json:"routerId,omitempty"`
	TransportId    string `json:"transportId,omitempty"`
	ProducerId     string `json:"producerId,omitempty"`
	ConsumerId     string `json:"consumerId,omitempty"`
	DataProducerId string `json:"dataProducerId,omitempty"`
	DataConsumerId string `json:"dataConsumerId,omitempty"`
	RtpObserverId  string `json:"rtpObserverId,omitempty"`
	WebRtcServerId string `json:"webRtcServerId,omitempty"`
}

const (
	NS_MESSAGE_MAX_LEN = 4194308
	NS_PAYLOAD_MAX_LEN = 4194304
)

// workerRequest represents the json request sent to the worker
type workerRequest struct {
	Id       int64        `json:"id,omitempty"`
	Method   string       `json:"method,omitempty"`
	Internal internalData `json:"internal,omitempty"`
	Data     interface{}  `json:"data,omitempty"`
}

// workerResponse represents the json response returned from the worker
type workerResponse struct {
	data json.RawMessage
	err  error
}

func (r workerResponse) Unmarshal(v interface{}) error {
	if r.err != nil {
		return r.err
	}
	if len(r.data) == 0 {
		return nil
	}
	return json.Unmarshal([]byte(r.data), v)
}

func (r workerResponse) Data() []byte {
	return []byte(r.data)
}

func (r workerResponse) Err() error {
	return r.err
}

// sentInfo includes rpc info
type sentInfo struct {
	method      string              // method name
	requestData []byte              // request json data
	payloadData []byte              // payload json data, used by payload channel
	respCh      chan workerResponse // channel to hold response
}

// workerNotification is the notification meta info sent to worker
type workerNotification struct {
	Event    string       `json:"event,omitempty"`
	Internal internalData `json:"internal,omitempty"`
	Data     interface{}  `json:"data,omitempty"`
}

// notification represents a notification of the specified target from worker
type notification struct {
	TargetId string          `json:"targetId,omitempty"`
	Event    string          `json:"event,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
}

type listenerOption func(*intervalListener)

func listenOnce() listenerOption {
	return func(l *intervalListener) {
		l.once = true
	}
}

type intervalListener struct {
	listenerValue reflect.Value
	argTypes      []reflect.Type
	once          bool
}

func newInternalListener(listener interface{}, options ...listenerOption) *intervalListener {
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
	for _, o := range options {
		o(l)
	}

	return l
}

func (l *intervalListener) Call(args ...interface{}) {
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
