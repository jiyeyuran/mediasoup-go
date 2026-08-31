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

	// ErrNotFound reports that the referenced entity doesn't exist.
	ErrNotFound = channel.ErrNotFound

	// ErrNoWorkerAvailable reports that a WorkerPool has no worker left to hand
	// out, because it was closed or because every worker has died and has not
	// yet been replaced.
	ErrNoWorkerAvailable = errors.New("no worker available in the pool")
)
