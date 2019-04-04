package mediasoup

import (
	"errors"
	"fmt"
)

type TypeError error

func NewTypeError(msg string) error {
	return TypeError(errors.New(msg))
}

// UnsupportedError indicating not support for something.
type UnsupportedError struct {
	name    string
	message string
}

func NewUnsupportedError(message string) error {
	return UnsupportedError{
		name:    "UnsupportedError",
		message: message,
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

func NewInvalidStateError(message string) error {
	return UnsupportedError{
		name:    "InvalidStateError",
		message: message,
	}
}

func (e InvalidStateError) Error() string {
	return fmt.Sprintf("%s:%s", e.name, e.message)
}
