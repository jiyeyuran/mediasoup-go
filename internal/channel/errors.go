package channel

import (
	"errors"
	"fmt"
)

var (
	ErrChannelClosed         = errors.New("channel closed")
	ErrChannelRequestTimeout = errors.New("request timed out")
	ErrBodyTooLarge          = errors.New("request body is too large")
	ErrBadSubscription       = errors.New("invalid subscription")

	// ErrNotFound reports that the entity referenced by a request does not exist in the worker.
	ErrNotFound = errors.New("NotFoundError")

	// ErrInvalidType reports that a request carried an invalid value.
	ErrInvalidType = errors.New("TypeError")
)

// newRequestError builds the error of a rejected request, mapping the error types known by the
// worker to sentinel errors so that callers can match them with errors.Is.
func newRequestError(errType, reason string) error {
	switch errType {
	case "NotFoundError":
		return fmt.Errorf("%w: %s", ErrNotFound, reason)

	case "TypeError":
		return fmt.Errorf("%w: %s", ErrInvalidType, reason)

	default:
		return errors.New(errType + ": " + reason)
	}
}
