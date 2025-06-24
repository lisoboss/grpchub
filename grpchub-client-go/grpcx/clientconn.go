package grpcx

import (
	"context"

	"github.com/google/uuid"
	"github.com/lisoboss/grpchub"
	"github.com/lisoboss/grpchub/middleware"
	ghTransport "github.com/lisoboss/grpchub/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

type ClientConn[Inner any] struct {
	ctx context.Context

	sm grpchub.ClientStreamManagerInterface[Inner]

	opts clientOptions
}

func (cc *ClientConn[Inner]) Close() {
	cc.sm.Close()
}

// Invoke implements grpc.ClientConnInterface.
func (cc *ClientConn[Inner]) Invoke(ctx context.Context, method string, args any, reply any, opts ...grpc.CallOption) error {
	s := cc.sm.Connect(uuid.NewString(), method)
	defer s.Close()

	stream := newWrappedClientStream(ctx, s, cc.opts.streamMacter)

	ctx = NewCTransportContext(ctx, cc.opts.endpoint, stream.GetStream())
	if cc.opts.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cc.opts.timeout)
		defer cancel()
	}
	h := func(ctx context.Context, req any) (any, error) {
		stream.SetContext(ctx)
		if err := stream.Handshake(); err != nil {
			return reply, err
		}
		if err := stream.SendMsgRaw(req); err != nil {
			return reply, err
		}
		return reply, stream.RecvMsgRaw(reply)
	}
	if len(cc.opts.middleware) > 0 {
		h = middleware.Chain(cc.opts.middleware...)(h)
	}
	_, err := h(ctx, args)
	return err
}

// NewStream implements grpc.ClientConnInterface.
func (cc *ClientConn[Inner]) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	stream := newWrappedClientStream(ctx, cc.sm.Connect(uuid.NewString(), method), cc.opts.streamMacter)

	ctx = NewCTransportContext(ctx, cc.opts.endpoint, stream.GetStream())
	if cc.opts.streamTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cc.opts.streamTimeout)
		defer cancel()
	}

	h := func(ctx context.Context) (err error) {
		stream.SetContext(ctx)
		return stream.Handshake()
	}

	if len(cc.opts.streamTransportMiddleware) > 0 {
		h = middleware.StreamTransportChain(cc.opts.streamTransportMiddleware...)(h)
	}

	return stream, h(ctx)
}

var _ grpc.ClientConnInterface = (*ClientConn[any])(nil)

func NewClientConn[Inner any](ctx context.Context, sm grpchub.ClientStreamManagerInterface[Inner], opts ...ClientOption) *ClientConn[Inner] {
	return &ClientConn[Inner]{
		ctx:  ctx,
		sm:   sm,
		opts: parseClientOptions(opts),
	}
}

// WrappedClientStream
type WrappedClientStream[Inner any] struct {
	ctx        context.Context
	stream     *ghTransport.ClientStream[Inner]
	middleware *middleware.Matcher
}

func (w *WrappedClientStream[Inner]) SetContext(ctx context.Context) {
	w.ctx = ctx
}

func (w *WrappedClientStream[Inner]) Handshake() error {
	return w.stream.Handshake()
}

func (w *WrappedClientStream[Inner]) GetStream() *ghTransport.ClientStream[Inner] {
	return w.stream
}

func (w *WrappedClientStream[Inner]) SendMsgRaw(m any) error {
	return w.stream.SendPayload(m.(proto.Message))
}

// SendMsg implements grpc.ClientStream.
func (w *WrappedClientStream[Inner]) SendMsg(m any) error {
	h := func(_ context.Context, req any) (any, error) {
		return req, w.SendMsgRaw(req)
	}

	if next := w.middleware.Match(w.stream.GetMethod()); len(next) > 0 {
		h = middleware.Chain(next...)(h)
	}

	_, err := h(w.Context(), m)
	return err
}

// CloseSend implements grpc.ClientStream.
func (w *WrappedClientStream[Inner]) CloseSend() error {
	return w.stream.SendClose()
}

// Context implements grpc.ClientStream.
func (w *WrappedClientStream[Inner]) Context() context.Context {
	return w.ctx
}

// Header implements grpc.ClientStream.
func (w *WrappedClientStream[Inner]) Header() (metadata.MD, error) {
	return metadata.MD(w.stream.RequestHeader()), nil
}

func (w *WrappedClientStream[Inner]) RecvMsgRaw(m any) error {
	reply, err := w.stream.Recv()
	if err != nil {
		return err
	}
	proto.Merge(m.(proto.Message), reply.(proto.Message))
	return nil
}

// RecvMsg implements grpc.ClientStream.
func (w *WrappedClientStream[Inner]) RecvMsg(m any) error {
	h := func(_ context.Context, req any) (any, error) {
		return req, w.RecvMsgRaw(req)
	}

	if next := w.middleware.Match(w.stream.GetMethod()); len(next) > 0 {
		h = middleware.Chain(next...)(h)
	}

	_, err := h(w.Context(), m)
	return err
}

// Trailer implements grpc.ClientStream.
func (w *WrappedClientStream[Inner]) Trailer() metadata.MD {
	return metadata.MD(w.stream.ReplyHeader())
}

var _ grpc.ClientStream = (*WrappedClientStream[any])(nil)

func newWrappedClientStream[Inner any](ctx context.Context, stream *ghTransport.ClientStream[Inner], middleware *middleware.Matcher) *WrappedClientStream[Inner] {
	return &WrappedClientStream[Inner]{
		ctx:        ctx,
		stream:     stream,
		middleware: middleware,
	}
}
