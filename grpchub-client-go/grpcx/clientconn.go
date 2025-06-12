package grpcx

import (
	"context"

	"github.com/go-kratos/kratos/v2/transport"
	"github.com/google/uuid"
	"github.com/lisoboss/grpchub"
	"github.com/lisoboss/grpchub/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

type GrpcxClientConn struct {
	ctx context.Context
	sm  grpchub.StreamManagerInterface

	opts clientOptions
}

func (cc *GrpcxClientConn) Close() error {
	return cc.sm.Close()
}

// Invoke implements grpc.ClientConnInterface.
func (cc *GrpcxClientConn) Invoke(ctx context.Context, method string, args any, reply any, opts ...grpc.CallOption) error {
	ctx = NewCTransportContext(ctx, cc.opts.endpoint, method)
	if cc.opts.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cc.opts.timeout)
		defer cancel()
	}
	h := func(ctx context.Context, req any) (any, error) {
		return reply, cc.invokeRaw(ctx, method, req, reply, opts...)
	}
	if len(cc.opts.middleware) > 0 {
		h = middleware.Chain(cc.opts.middleware...)(h)
	}
	_, err := h(ctx, args)
	return err
}

func (cc *GrpcxClientConn) invokeRaw(ctx context.Context, method string, args any, reply any, _ ...grpc.CallOption) error {
	sConn := cc.sm.NewStreamTransport(uuid.NewString())
	defer sConn.Close()

	// set header
	md, _ := metadata.FromOutgoingContext(ctx)
	tr, _ := transport.FromClientContext(ctx)
	md = metadata.Join(md, MDFromTransportHeader(tr.RequestHeader()))
	ctx = metadata.NewOutgoingContext(ctx, md)

	stream, err := newGrpcxClientStream(ctx, method, sConn, cc.opts.streamMacter)
	if err != nil {
		return err
	}

	ctr, _ := FromClientTransportContext(ctx)
	ctr.rpHeader = stream.stream.GetTrailer

	if err := stream.SendMsg(args); err != nil {
		return err
	}
	return stream.RecvMsg(reply)
}

// NewStream implements grpc.ClientConnInterface.
func (cc *GrpcxClientConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	ctx = NewCTransportContext(ctx, cc.opts.endpoint, method)
	if cc.opts.streamTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cc.opts.streamTimeout)
		defer cancel()
	}

	var gcs *GrpcxClientStream
	h := func(ctx context.Context) (err error) {
		gcs, err = cc.newStreamRaw(ctx, desc, method, opts...)
		return err
	}
	if len(cc.opts.streamTransportMiddleware) > 0 {
		h = middleware.StreamTransportChain(cc.opts.streamTransportMiddleware...)(h)
	}

	return gcs, h(ctx)
}

func (cc *GrpcxClientConn) newStreamRaw(ctx context.Context, _ *grpc.StreamDesc, method string, _ ...grpc.CallOption) (*GrpcxClientStream, error) {
	sConn := cc.sm.NewStreamTransport(uuid.NewString())

	// set header
	md, _ := metadata.FromOutgoingContext(ctx)
	tr, _ := transport.FromClientContext(ctx)
	md = metadata.Join(md, MDFromTransportHeader(tr.RequestHeader()))
	ctx = metadata.NewOutgoingContext(ctx, md)

	stream, err := newGrpcxClientStream(ctx, method, sConn, cc.opts.streamMacter)
	if err != nil {
		return nil, err
	}

	ctr, _ := FromClientTransportContext(ctx)
	ctr.rpHeader = stream.stream.GetTrailer

	return stream, nil
}

var _ grpc.ClientConnInterface = (*GrpcxClientConn)(nil)

func newGrpcxClientConn(ctx context.Context, sm grpchub.StreamManagerInterface, opts ...ClientOption) *GrpcxClientConn {
	return &GrpcxClientConn{
		ctx:  ctx,
		sm:   sm,
		opts: parseClientOptions(opts),
	}
}

// GrpcxClientStream
type GrpcxClientStream struct {
	middleware *middleware.Matcher

	stream *grpchub.ClientStream
}

func (g *GrpcxClientStream) SendHello() error {
	return g.stream.SayHello()
}

// SendMsg implements grpc.ClientStream.
func (cs *GrpcxClientStream) SendMsg(m any) error {
	h := func(_ context.Context, req any) (any, error) {
		return req, cs.stream.Send(grpchub.PT_PAYLOAD, m.(proto.Message))
	}

	if next := cs.middleware.Match(cs.stream.GetMethod()); len(next) > 0 {
		h = middleware.Chain(next...)(h)
	}

	_, err := h(cs.Context(), m)
	return err
}

// CloseSend implements grpc.ClientStream.
func (cs *GrpcxClientStream) CloseSend() error {
	return cs.stream.Send(grpchub.PT_CLOSE, nil)
}

// Context implements grpc.ClientStream.
func (cs *GrpcxClientStream) Context() context.Context {
	return cs.stream.Context()
}

// Header implements grpc.ClientStream.
func (cs *GrpcxClientStream) Header() (metadata.MD, error) {
	return cs.stream.GetHeader(), nil
}

// RecvMsg implements grpc.ClientStream.
func (cs *GrpcxClientStream) RecvMsg(m any) error {
	h := func(_ context.Context, req any) (any, error) {
		return req, cs.stream.Recv(req)
	}

	if next := cs.middleware.Match(cs.stream.GetMethod()); len(next) > 0 {
		h = middleware.Chain(next...)(h)
	}

	_, err := h(cs.Context(), m)
	return err
}

// Trailer implements grpc.ClientStream.
func (cs *GrpcxClientStream) Trailer() metadata.MD {
	return cs.stream.GetTrailer()
}

var _ grpc.ClientStream = (*GrpcxClientStream)(nil)

func newGrpcxClientStream(ctx context.Context, method string, sConn grpchub.StreamTransportInterface, middleware *middleware.Matcher) (cs *GrpcxClientStream, err error) {
	cs = &GrpcxClientStream{
		middleware: middleware,
		stream:     grpchub.NewClientStream(ctx, method, sConn),
	}

	if err = cs.SendHello(); err != nil {
		return
	}

	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		err = cs.stream.Send(grpchub.PT_HEADER, md)
		if err != nil {
			return
		}
	}

	return
}
