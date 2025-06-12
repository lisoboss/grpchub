package test

import (
	"context"

	"github.com/go-kratos/kratos/v2/transport"
	"github.com/lisoboss/grpchub/middleware"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	authKey = "C-AUTH"
)

func WithAuth(token string) middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (resp any, err error) {
			return next(metadata.AppendToOutgoingContext(ctx, authKey, token), req)
		}
	}
}

func Auth(token string) middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (resp any, err error) {
			txp, ok := transport.FromServerContext(ctx)
			if !ok {
				return resp, status.Error(codes.Aborted, "Not found Transporter")
			}
			token2 := txp.RequestHeader().Get(authKey)
			if token2 == token {
				return next(ctx, req)
			} else {
				return resp, status.Error(codes.Unauthenticated, "Request unauthenticated with "+authKey)
			}
		}
	}
}

func WithStreamAuth(token string) middleware.StreamTransportMiddleware {
	return func(next middleware.StreamTransportHandler) middleware.StreamTransportHandler {
		return func(ctx context.Context) error {
			return next(metadata.AppendToOutgoingContext(ctx, authKey, token))
		}
	}
}

func StreamAuth(token string) middleware.StreamTransportMiddleware {
	return func(next middleware.StreamTransportHandler) middleware.StreamTransportHandler {
		return func(ctx context.Context) error {
			txp, ok := transport.FromServerContext(ctx)
			if !ok {
				return status.Error(codes.Aborted, "Not found Transporter")
			}
			token2 := txp.RequestHeader().Get(authKey)
			if token2 == token {
				return next(ctx)
			} else {
				return status.Error(codes.Unauthenticated, "Stream request unauthenticated with "+authKey)
			}
		}
	}
}
