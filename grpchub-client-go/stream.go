package grpchub

import (
	"context"
	"errors"
	"io"
	"sync"

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

	headerSended  bool
	trailerSended bool

	serv     bool
	sayHello chan struct{}

	recvC     chan any
	closed    chan struct{}
	closeOnce sync.Once
}

func (s *stream) recv() error {
	pkg, err := s.conn.Recv()
	if err != nil {
		return err
	}

	logger.Infof("Recv pkg: %#v", pkg)

	switch pkg.Type {
	case PT_ERROR:
		var ss spb.Status
		if err = anypb.UnmarshalTo(pkg.Payload, &ss, proto.UnmarshalOptions{}); err != nil {
			return errors.New("anypb UnmarshalTo invalid error payload")
		}
		s.recvC <- status.ErrorProto(&ss)
		return nil

	case PT_HEADER:
		s.SetHeader(parseMetadataEntries(pkg.Md))
		if s.serv {
			s.ctx = metadata.NewIncomingContext(s.ctx, s.header)
		}
		return nil
	case PT_TRAILER:
		s.SetTrailer(parseMetadataEntries(pkg.Md))
		return nil
	case PT_PAYLOAD:
		s.recvC <- pkg.GetPayload()
		return nil
		// return anypb.UnmarshalTo(pkg.GetPayload(), reply.(proto.Message), proto.UnmarshalOptions{})
	case PT_CLOSE:
		s.recvC <- io.EOF
		return nil
	case PT_HELLO:
		s.sayHello <- struct{}{}
		s.method = pkg.Method
		return nil
	default:
		return errors.New("recv unexpected message type")
	}
}

func (s *stream) Loop() {
	defer s.Close()

	for {
		select {
		case <-s.ctx.Done():
			logger.Errorf("stream loop ctx err %s", s.ctx.Err())
			return
		case <-s.closed:
			return
		default:
		}

		if err := s.recv(); err != nil {
			logger.Errorf("stream loop err %s", err)
			return
		}
	}

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
	// metadata.MD 是 map 类型（引用类型），所以 GetHeader() 返回的是对原 map 的引用，只要没有做拷贝，是可以通过 .Set() 修改原始 s.header 的。
	// s.header.Copy() 不能修改
	return s.header
}

func (s *stream) GetTrailer() metadata.MD {
	return s.trailer
}

func (s *stream) SetHeader(md metadata.MD) {
	MD(s.header).Append(md)
	s.headerSended = false
}

func (s *stream) SetTrailer(md metadata.MD) {
	MD(s.trailer).Append(md)
	s.trailerSended = false
}

func (s *stream) Send(t channelv1.PackageType, args any) error {
	logger.Infof("Send t: %s, args: %#v", t.String(), args)
	pkg := &channelv1.MessagePackage{
		Type:   t,
		Method: s.method,
	}

	// logger.Infof("Send pkg Method: %s, Type: %s", pkg.Method, pkg.Type.String())

	// auto send header trailer
	switch t {
	case PT_HEADER, PT_PAYLOAD:
		if !s.serv && !s.headerSended && s.header.Len() > 0 {
			if err := s.conn.Send(&channelv1.MessagePackage{
				Type:   PT_HEADER,
				Method: s.method,
				Md:     buildMetadataEntries(s.header),
			}); err != nil {
				return err
			}
			s.headerSended = true
		}
	case PT_TRAILER, PT_ERROR, PT_CLOSE:
		if s.serv && !s.trailerSended && s.trailer.Len() > 0 {
			if err := s.conn.Send(&channelv1.MessagePackage{
				Type:   PT_TRAILER,
				Method: s.method,
				Md:     buildMetadataEntries(s.trailer),
			}); err != nil {
				return err
			}
			s.trailerSended = true
		}
	}

	switch t {
	case PT_HELLO, PT_CLOSE:
		logger.Error(t.String())
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
	select {
	case <-s.closed:
		return io.EOF
	case rel := <-s.recvC:
		if err, ok := rel.(error); ok {
			return err
		}
		return anypb.UnmarshalTo(rel.(*anypb.Any), reply.(proto.Message), proto.UnmarshalOptions{})
	}

}

func (s *stream) Close() error {
	s.closeOnce.Do(func() {
		close(s.closed)
		close(s.recvC)
		close(s.sayHello)
		s.conn.Close()
	})
	return nil
}

func newStream(ctx context.Context, method string, conn StreamTransportInterface, serv bool) stream {
	return stream{
		ctx:      ctx,
		method:   method,
		conn:     conn,
		serv:     serv,
		header:   metadata.MD{},
		trailer:  metadata.MD{},
		recvC:    make(chan any, 16),
		closed:   make(chan struct{}),
		sayHello: make(chan struct{}),
	}
}

type ClientStream struct {
	*stream
}

func (s *ClientStream) SayHello() error {
	return s.Send(PT_HELLO, nil)
}

func NewClientStream(ctx context.Context, method string, conn StreamTransportInterface) *ClientStream {
	s := newStream(ctx, method, conn, false)
	go s.Loop()
	return &ClientStream{
		stream: &s,
	}
}

type ServerStream struct {
	*stream
}

func (s *ServerStream) RecvHello() {
	<-s.sayHello
}

func (s *ServerStream) WriteStatus(st *status.Status) error {
	return s.Send(PT_ERROR, st.Proto())
}

func NewServerStream(ctx context.Context, conn StreamTransportInterface) *ServerStream {
	s := newStream(ctx, "", conn, true)
	go s.Loop()
	return &ServerStream{
		stream: &s,
	}
}

type MD metadata.MD

func (m MD) Append(mdmd metadata.MD) {
	md := metadata.MD(m)
	for key, vals := range mdmd {
		md.Append(key, vals...)
	}
}
