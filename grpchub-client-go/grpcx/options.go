package grpcx

import (
	"time"

	"github.com/lisoboss/grpchub/middleware"
)

type clientOptions struct {
	endpoint                  string
	timeout                   time.Duration
	streamTimeout             time.Duration
	middleware                []middleware.Middleware
	streamTransportMiddleware []middleware.StreamTransportMiddleware // 作用于发起链接和结束链接过程
	streamMacter              *middleware.Matcher                    // 作用于grpc流内部的请求
}

// ClientOption is gRPC client option.
type ClientOption func(o *clientOptions)

// WithEndpoint with client endpoint.
func WithEndpoint(endpoint string) ClientOption {
	return func(o *clientOptions) {
		o.endpoint = endpoint
	}
}

// WithTimeout with client timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.timeout = timeout
	}
}

// WithStreamTimeout with client stream timeout.
func WithStreamTimeout(timeout time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.streamTimeout = timeout
	}
}

// WithMiddleware with client middleware.
func WithMiddleware(m ...middleware.Middleware) ClientOption {
	return func(o *clientOptions) {
		o.middleware = m
	}
}

// WithStreamTransportMiddleware with client stream transport middleware.
func WithStreamTransportMiddleware(m ...middleware.StreamTransportMiddleware) ClientOption {
	return func(o *clientOptions) {
		o.streamTransportMiddleware = m
	}
}

// WithStreamMessageMiddleware with client stream message middleware.
func WithStreamMessageMiddleware(selector string, m ...middleware.Middleware) ClientOption {
	return func(o *clientOptions) {
		o.streamMacter.Add(selector, m...)
	}
}

func parseClientOptions(opts []ClientOption) clientOptions {
	options := clientOptions{
		timeout:      2000 * time.Millisecond,
		streamMacter: middleware.NewMatcher(),
	}

	for _, o := range opts {
		o(&options)
	}

	return options
}

type serverOptions struct {
	endpoint                  string
	timeout                   time.Duration
	streamTimeout             time.Duration
	middleware                []middleware.Middleware
	streamTransportMiddleware []middleware.StreamTransportMiddleware
	streamMacter              *middleware.Matcher
	maxWorker                 int
}

// ServerOption is gRPC server option.
type ServerOption func(o *serverOptions)

// Endpoint with client endpoint.
func Endpoint(endpoint string) ServerOption {
	return func(o *serverOptions) {
		o.endpoint = endpoint
	}
}

// Timeout with server timeout.
func Timeout(timeout time.Duration) ServerOption {
	return func(s *serverOptions) {
		s.timeout = timeout
	}
}

// StreamTimeout with server stream timeout.
func StreamTimeout(timeout time.Duration) ServerOption {
	return func(s *serverOptions) {
		s.streamTimeout = timeout
	}
}

// Middleware with server middleware.
func Middleware(m ...middleware.Middleware) ServerOption {
	return func(s *serverOptions) {
		s.middleware = m
	}
}

// StreamTransportMiddleware with client stream transport middleware.
func StreamTransportMiddleware(m ...middleware.StreamTransportMiddleware) ServerOption {
	return func(o *serverOptions) {
		o.streamTransportMiddleware = m
	}
}

// StreamMessageMiddleware with client stream message middleware.
func StreamMessageMiddleware(selector string, m ...middleware.Middleware) ServerOption {
	return func(s *serverOptions) {
		s.streamMacter.Add(selector, m...)
	}
}

// MaxWorker sets the maximum number of concurrent streams.
func MaxWorker(max int) ServerOption {
	return func(s *serverOptions) {
		s.maxWorker = max
	}
}

func parseServerOptions(opts []ServerOption) serverOptions {
	options := serverOptions{
		timeout:      2000 * time.Millisecond,
		streamMacter: middleware.NewMatcher(),
		maxWorker:    100, // Default pool size
	}

	for _, o := range opts {
		o(&options)
	}

	return options
}
