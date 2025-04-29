package channel

import "errors"

var (
	ErrChannelClosed         = errors.New("worker: channel closed")
	ErrChannelRequestTimeout = errors.New("worker: request timed out")
	ErrBodyTooLarge          = errors.New("worker: request body is too large")
	ErrBadSubscription       = errors.New("worker: invalid subscription")
)
