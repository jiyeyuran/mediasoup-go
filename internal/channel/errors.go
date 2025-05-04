package channel

import "errors"

var (
	ErrChannelClosed         = errors.New("channel closed")
	ErrChannelRequestTimeout = errors.New("request timed out")
	ErrBodyTooLarge          = errors.New("request body is too large")
	ErrBadSubscription       = errors.New("invalid subscription")
)
