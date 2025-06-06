package grpcx

import (
	"context"

	"github.com/google/uuid"
	"github.com/lisoboss/grpchub"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

type GrpcxClientConn struct {
	ctx context.Context
	sm  grpchub.StreamManagerInterface
}

func (cc *GrpcxClientConn) Close() error {
	return cc.sm.Close()
}

// Invoke implements grpc.ClientConnInterface.
func (cc *GrpcxClientConn) Invoke(ctx context.Context, method string, args any, reply any, opts ...grpc.CallOption) error {
	sConn := cc.sm.NewStreamTransport(uuid.NewString())
	defer sConn.Close()

	stream, err := newGrpcxClientStream(ctx, method, sConn)
	if err != nil {
		return err
	}

	if err := stream.SendMsg(args); err != nil {
		return err
	}

	return stream.RecvMsg(reply)
}

// NewStream implements grpc.ClientConnInterface.
func (cc *GrpcxClientConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	sConn := cc.sm.NewStreamTransport(uuid.NewString())
	return newGrpcxClientStream(ctx, method, sConn)
}

var _ grpc.ClientConnInterface = (*GrpcxClientConn)(nil)

func newGrpcxClientConn(ctx context.Context, sm grpchub.StreamManagerInterface) *GrpcxClientConn {
	return &GrpcxClientConn{
		ctx: ctx,
		sm:  sm,
	}
}

// GrpcxClientStream
type GrpcxClientStream struct {
	// TODO opts []grpc.CallOption

	stream *grpchub.ClientStream
}

func (g *GrpcxClientStream) SendHello() error {
	return g.stream.SendHello()
}

// SendMsg implements grpc.ClientStream.
func (cs *GrpcxClientStream) SendMsg(m any) error {
	return cs.stream.Send(grpchub.PT_PAYLOAD, m.(proto.Message))
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
	return cs.stream.Recv(m)
}

// Trailer implements grpc.ClientStream.
func (cs *GrpcxClientStream) Trailer() metadata.MD {
	return cs.stream.GetTrailer()
}

var _ grpc.ClientStream = (*GrpcxClientStream)(nil)

func newGrpcxClientStream(ctx context.Context, method string, sConn grpchub.StreamTransportInterface) (cs *GrpcxClientStream, err error) {
	cs = &GrpcxClientStream{
		stream: grpchub.NewClientStream(ctx, method, sConn),
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
