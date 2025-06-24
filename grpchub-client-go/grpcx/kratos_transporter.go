package grpcx

import (
	"context"

	"github.com/go-kratos/kratos/v2/transport"
	ghTransport "github.com/lisoboss/grpchub/transport"
	"google.golang.org/grpc/metadata"
)

type clientTransporter[Inner any] struct {
	endpoint string

	stream *ghTransport.ClientStream[Inner]
}

// Endpoint implements transport.Transporter.
func (t *clientTransporter[Inner]) Endpoint() string {
	return t.endpoint
}

// Kind implements transport.Transporter.
func (t *clientTransporter[Inner]) Kind() transport.Kind {
	return "grpc"
}

// Operation implements transport.Transporter.
func (t *clientTransporter[Inner]) Operation() string {
	return t.stream.GetMethod()
}

// ReplyHeader implements transport.Transporter.
func (t *clientTransporter[Inner]) ReplyHeader() transport.Header {
	return headerCarrier(t.stream.ReplyHeader())
}

// RequestHeader implements transport.Transporter.
func (t *clientTransporter[Inner]) RequestHeader() transport.Header {
	return headerCarrier(t.stream.RequestHeader())
}

var _ transport.Transporter = (*clientTransporter[any])(nil)

type serverTransporter[Inner any] struct {
	endpoint string

	stream *ghTransport.ServerStream[Inner]
}

// Endpoint implements transport.Transporter.
func (s *serverTransporter[Inner]) Endpoint() string {
	return s.endpoint
}

// Kind implements transport.Transporter.
func (s *serverTransporter[Inner]) Kind() transport.Kind {
	return "grpc"
}

// Operation implements transport.Transporter.
func (s *serverTransporter[Inner]) Operation() string {
	return s.stream.GetMethod()
}

// ReplyHeader implements transport.Transporter.
func (s *serverTransporter[Inner]) ReplyHeader() transport.Header {
	return headerCarrier(s.stream.ReplyHeader())
}

// RequestHeader implements transport.Transporter.
func (s *serverTransporter[Inner]) RequestHeader() transport.Header {
	return headerCarrier(s.stream.RequestHeader())
}

var _ transport.Transporter = (*serverTransporter[any])(nil)

type (
	serverTransportKey struct{}
	clientTransportKey struct{}
)

func NewServerTransportContext[Inner any](ctx context.Context, str *serverTransporter[Inner]) context.Context {
	return context.WithValue(ctx, serverTransportKey{}, str)
}

func FromServerTransportContext[Inner any](ctx context.Context) (str *serverTransporter[Inner], ok bool) {
	str, ok = ctx.Value(serverTransportKey{}).(*serverTransporter[Inner])
	return
}

func NewClientTransportContext[Inner any](ctx context.Context, ctr *clientTransporter[Inner]) context.Context {
	return context.WithValue(ctx, clientTransportKey{}, ctr)
}

func FromClientTransportContext[Inner any](ctx context.Context) (ctr *clientTransporter[Inner], ok bool) {
	ctr, ok = ctx.Value(clientTransportKey{}).(*clientTransporter[Inner])
	return
}

func NewCTransportContext[Inner any](ctx context.Context, endpoint string, stream *ghTransport.ClientStream[Inner]) context.Context {
	ct := &clientTransporter[Inner]{
		endpoint: endpoint,
		stream:   stream,
	}
	ctx = NewClientTransportContext(ctx, ct)
	ctx = transport.NewClientContext(ctx, ct)
	return ctx
}

func NewSTransportContext[Inner any](ctx context.Context, endpoint string, stream *ghTransport.ServerStream[Inner]) context.Context {
	st := &serverTransporter[Inner]{
		endpoint: endpoint,
		stream:   stream,
	}
	ctx = NewServerTransportContext(ctx, st)
	ctx = transport.NewServerContext(ctx, st)
	return ctx
}

type headerCarrier metadata.MD

// Get returns the value associated with the passed key.
func (mc headerCarrier) Get(key string) string {
	vals := metadata.MD(mc).Get(key)
	if len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// Set stores the key-value pair.
func (mc headerCarrier) Set(key string, value string) {
	metadata.MD(mc).Set(key, value)
}

// Add append value to key-values pair.
func (mc headerCarrier) Add(key string, value string) {
	metadata.MD(mc).Append(key, value)
}

// Keys lists the keys stored in this carrier.
func (mc headerCarrier) Keys() []string {
	keys := make([]string, 0, len(mc))
	for k := range metadata.MD(mc) {
		keys = append(keys, k)
	}
	return keys
}

// Values returns a slice of values associated with the passed key.
func (mc headerCarrier) Values(key string) []string {
	return metadata.MD(mc).Get(key)
}

func MDFromTransportHeader(header transport.Header) (md metadata.MD) {
	for _, key := range header.Keys() {
		md.Append(key, header.Values(key)...)
	}
	return md
}

func MDToTransportHeader(header transport.Header, md metadata.MD) {
	hc := headerCarrier(md)
	for _, key := range hc.Keys() {
		for _, value := range hc.Values(key) {
			header.Set(key, value)
		}
	}
}
