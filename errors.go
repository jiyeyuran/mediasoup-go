package mediasoup

import (
	"fmt"
)

type TypeError error

func NewTypeError(format string, args ...interface{}) error {
	return TypeError(fmt.Errorf(format, args...))
}

// UnsupportedError indicating not support for something.
type UnsupportedError struct {
	name    string
	message string
}

func NewUnsupportedError(format string, args ...interface{}) error {
	return UnsupportedError{
		name:    "UnsupportedError",
		message: fmt.Sprintf(format, args...),
	}
}

func (e UnsupportedError) Error() string {
	return fmt.Sprintf("%s:%s", e.name, e.message)
}

// InvalidStateError produced when calling a method in an invalid state.
type InvalidStateError struct {
	name    string
	message string
}

func NewInvalidStateError(format string, args ...interface{}) error {
	return UnsupportedError{
		name:    "InvalidStateError",
		message: fmt.Sprintf(format, args...),
	}
}

func (e InvalidStateError) Error() string {
	return fmt.Sprintf("%s:%s", e.name, e.message)
}
