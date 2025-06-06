package grpchub

import (
	"context"
	"io"

	channelv1 "github.com/lisoboss/grpchub/gen/channel/v1"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	PT_HELLO   channelv1.PackageType = channelv1.PackageType_PT_HELLO
	PT_HEADER  channelv1.PackageType = channelv1.PackageType_PT_HEADER
	PT_PAYLOAD channelv1.PackageType = channelv1.PackageType_PT_PAYLOAD
	PT_TRAILER channelv1.PackageType = channelv1.PackageType_PT_TRAILER
	PT_CLOSE   channelv1.PackageType = channelv1.PackageType_PT_CLOSE
	PT_ERROR   channelv1.PackageType = channelv1.PackageType_PT_ERROR
)

type StreamTransportInterface interface {
	Send(*channelv1.MessagePackage) error
	Recv() (*channelv1.MessagePackage, error)
	Close() error
}

type stream struct {
	ctx    context.Context
	method string
	conn   StreamTransportInterface

	header  metadata.MD
	trailer metadata.MD

	serv bool
}

func (s *stream) Context() context.Context {
	return s.ctx
}

func (s *stream) SetContext(ctx context.Context) {
	s.ctx = ctx
}

func (s *stream) GetMethod() string {
	return s.method
}

func (s *stream) GetHeader() metadata.MD {
	return s.header
}

func (s *stream) GetTrailer() metadata.MD {
	return s.trailer
}

func (s *stream) SetHeader(md metadata.MD) {
	s.header = metadata.Join(s.header, md)
}

func (s *stream) SetTrailer(md metadata.MD) {
	s.trailer = metadata.Join(s.trailer, md)
}

func (s *stream) Send(t channelv1.PackageType, args any) error {
	pkg := &channelv1.MessagePackage{
		Type:   t,
		Method: s.method,
	}

	// logger.Infof("Send pkg Method: %s, Type: %s", pkg.Method, pkg.Type.String())

	// auto send header trailer
	switch t {
	case PT_HEADER, PT_PAYLOAD:
		if s.header.Len() > 0 {
			if err := s.conn.Send(&channelv1.MessagePackage{
				Type:   PT_HEADER,
				Method: s.method,
				Md:     buildMetadataEntries(s.header),
			}); err != nil {
				return err
			}
		}
	case PT_TRAILER, PT_ERROR, PT_CLOSE:
		if s.trailer.Len() > 0 {
			if err := s.conn.Send(&channelv1.MessagePackage{
				Type:   PT_TRAILER,
				Method: s.method,
				Md:     buildMetadataEntries(s.trailer),
			}); err != nil {
				return err
			}
		}
	}

	switch t {
	case PT_HELLO, PT_CLOSE:
	case PT_HEADER, PT_TRAILER:
		pkg.Md = buildMetadataEntries(args.(metadata.MD))
	case PT_PAYLOAD, PT_ERROR:
		payload, err := anypb.New(args.(proto.Message))
		if err != nil {
			return err
		}
		pkg.Payload = payload
	default:
		return status.Error(codes.Internal, "unexpected message type")
	}

	return s.conn.Send(pkg)
}

func (s *stream) Recv(reply any) error {
	pkg, err := s.conn.Recv()
	if err != nil {
		return err
	}

	// logger.Infof("Recv pkg: %#v", pkg)

	switch pkg.Type {
	case PT_ERROR:
		var s spb.Status
		if err := anypb.UnmarshalTo(pkg.Payload, &s, proto.UnmarshalOptions{}); err != nil {
			return status.Error(codes.Internal, "invalid error payload")
		}
		return status.ErrorProto(&s)
	case PT_HEADER:
		s.SetHeader(parseMetadataEntries(pkg.Md))
		if s.serv {
			s.ctx = metadata.NewIncomingContext(s.ctx, s.header)
		}
		return s.Recv(reply)
	case PT_TRAILER:
		s.SetTrailer(parseMetadataEntries(pkg.Md))
		return s.Recv(reply)
	case PT_PAYLOAD:
		return anypb.UnmarshalTo(pkg.GetPayload(), reply.(proto.Message), proto.UnmarshalOptions{})
	case PT_CLOSE:
		return io.EOF
	default:
		return status.Error(codes.Internal, "unexpected message type")
	}
}

func (s *stream) Close() error {
	return s.conn.Close()
}

type ClientStream struct {
	stream
}

func (s *ClientStream) SendHello() error {
	return s.Send(PT_HELLO, nil)
}

func NewClientStream(ctx context.Context, method string, conn StreamTransportInterface) *ClientStream {
	return &ClientStream{
		stream: stream{
			ctx:    ctx,
			method: method,
			conn:   conn,
			serv:   false,
		},
	}
}

type ServerStream struct {
	stream
}

func (s *ServerStream) RecvHello() (err error) {
	pkg, err := s.conn.Recv()
	if err != nil {
		return err
	}

	if pkg.Type == PT_HELLO {
		s.method = pkg.Method
		return nil
	}

	return status.Error(codes.Internal, "unexpected message type")
}

func (s *ServerStream) WriteStatus(st *status.Status) error {
	return s.Send(PT_ERROR, st.Proto())
}

func NewServerStream(ctx context.Context, conn StreamTransportInterface) *ServerStream {
	return &ServerStream{
		stream: stream{
			ctx:  ctx,
			conn: conn,
			serv: true,
		},
	}
}
