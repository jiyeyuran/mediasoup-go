package channel

import (
	"context"

	FbsNotification "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Notification"
)

type wrappedContextKeyType struct{}

var wrappedContextKey = wrappedContextKeyType{}

type WrappedContext struct {
	Event     FbsNotification.Event
	HandlerId string
	Context   context.Context
}

func withWrappedContext(ctx context.Context, data *WrappedContext) context.Context {
	return context.WithValue(ctx, wrappedContextKey, data)
}

func getWrappedContext(ctx context.Context) (*WrappedContext, bool) {
	if data, ok := ctx.Value(wrappedContextKey).(*WrappedContext); ok {
		return data, true
	}
	return nil, false
}

func UnwrapContext(ctx context.Context, handlerId string) context.Context {
	if data, ok := getWrappedContext(ctx); ok && handlerId == data.HandlerId {
		return data.Context
	}
	return ctx
}
