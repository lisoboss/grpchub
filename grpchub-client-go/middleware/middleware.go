package middleware

import (
	"context"
)

// Handler defines the handler invoked by Middleware.
type Handler func(ctx context.Context, req any) (any, error)

// Middleware is gRPC transport middleware.
type Middleware func(Handler) Handler

// Chain returns a Middleware that specifies the chained handler for endpoint.
func Chain(m ...Middleware) Middleware {
	return func(next Handler) Handler {
		for i := len(m) - 1; i >= 0; i-- {
			next = m[i](next)
		}
		return next
	}
}

type StreamTransportHandler func(ctx context.Context) error

// StreamTransportMiddleware is gRPC stream transport middleware.
type StreamTransportMiddleware func(StreamTransportHandler) StreamTransportHandler

func StreamTransportChain(m ...StreamTransportMiddleware) StreamTransportMiddleware {
	return func(next StreamTransportHandler) StreamTransportHandler {
		for i := len(m) - 1; i >= 0; i-- {
			next = m[i](next)
		}
		return next
	}
}

func NewWrappedStreamTransportMiddleware(ms ...Middleware) []StreamTransportMiddleware {
	sms := make([]StreamTransportMiddleware, len(ms))
	for i, m := range ms {
		sms[i] = func(sth StreamTransportHandler) StreamTransportHandler {
			mh := m(func(ctx context.Context, _ any) (_ any, err error) {
				err = sth(ctx)
				return
			})
			return func(ctx context.Context) (err error) {
				_, err = mh(ctx, nil)
				return
			}
		}
	}
	return sms
}
