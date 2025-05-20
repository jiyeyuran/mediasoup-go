package mediasoup

import (
	"errors"

	"github.com/jiyeyuran/mediasoup-go/v2/internal/channel"
)

var (
	ErrWorkerStartTimeout       = errors.New("start worker timed out")
	ErrWorkerClosed             = errors.New("worker is closed")
	ErrRouterClosed             = errors.New("router is closed")
	ErrTransportClosed          = errors.New("transport is closed")
	ErrMissSctpStreamParameters = errors.New("sctpStreamParameters is missing")
	ErrNotImplemented           = errors.New("not implemented")
	ErrChannelClosed            = channel.ErrChannelClosed
	ErrChannelRequestTimeout    = channel.ErrChannelRequestTimeout
	ErrBodyTooLarge             = channel.ErrBodyTooLarge
)
