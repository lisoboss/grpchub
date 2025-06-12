package grpcx

import (
	"context"

	"github.com/go-kratos/kratos/v2/transport"
	"google.golang.org/grpc/metadata"
)

type clientTransporter struct {
	endpoint string
	method   string
	rqHeader headerCarrier
	rpHeader func() metadata.MD
}

// Endpoint implements transport.Transporter.
func (t *clientTransporter) Endpoint() string {
	return t.endpoint
}

// Kind implements transport.Transporter.
func (t *clientTransporter) Kind() transport.Kind {
	return "grpc"
}

// Operation implements transport.Transporter.
func (t *clientTransporter) Operation() string {
	return t.method
}

// ReplyHeader implements transport.Transporter.
func (t *clientTransporter) ReplyHeader() transport.Header {
	return headerCarrier(t.rpHeader())
}

// RequestHeader implements transport.Transporter.
func (t *clientTransporter) RequestHeader() transport.Header {
	return t.rqHeader
}

var _ transport.Transporter = (*clientTransporter)(nil)

type serverTransporter struct {
	endpoint string
	method   string
	rqHeader func() metadata.MD
	rpHeader func() metadata.MD
}

// Endpoint implements transport.Transporter.
func (s *serverTransporter) Endpoint() string {
	return s.endpoint
}

// Kind implements transport.Transporter.
func (s *serverTransporter) Kind() transport.Kind {
	return "grpc"
}

// Operation implements transport.Transporter.
func (s *serverTransporter) Operation() string {
	return s.method
}

// ReplyHeader implements transport.Transporter.
func (s *serverTransporter) ReplyHeader() transport.Header {
	return headerCarrier(s.rpHeader())
}

// RequestHeader implements transport.Transporter.
func (s *serverTransporter) RequestHeader() transport.Header {
	return headerCarrier(s.rqHeader())
}

var _ transport.Transporter = (*serverTransporter)(nil)

type (
	serverTransportKey struct{}
	clientTransportKey struct{}
)

func NewServerTransportContext(ctx context.Context, str *serverTransporter) context.Context {
	return context.WithValue(ctx, serverTransportKey{}, str)
}

func FromServerTransportContext(ctx context.Context) (str *serverTransporter, ok bool) {
	str, ok = ctx.Value(serverTransportKey{}).(*serverTransporter)
	return
}

func NewClientTransportContext(ctx context.Context, ctr *clientTransporter) context.Context {
	return context.WithValue(ctx, clientTransportKey{}, ctr)
}

func FromClientTransportContext(ctx context.Context) (ctr *clientTransporter, ok bool) {
	ctr, ok = ctx.Value(clientTransportKey{}).(*clientTransporter)
	return
}

func NewCTransportContext(ctx context.Context, endpoint, method string) context.Context {
	ct := &clientTransporter{
		endpoint: endpoint,
		method:   method,
	}
	ctx = NewClientTransportContext(ctx, ct)
	ctx = transport.NewClientContext(ctx, ct)
	return ctx
}

func NewSTransportContext(ctx context.Context, endpoint, method string, frq, frp func() metadata.MD) context.Context {
	st := &serverTransporter{
		endpoint: endpoint,
		method:   method,
		rqHeader: frq,
		rpHeader: frp,
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
