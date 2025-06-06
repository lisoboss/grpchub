package grpcx

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/lisoboss/grpchub"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type serviceInfo struct {
	serviceImpl any
	methods     map[string]*grpc.MethodDesc
	streams     map[string]*grpc.StreamDesc
	mdata       any
}

type GrpcServer struct {
	ctx       context.Context
	cancel    context.CancelFunc
	closed    chan struct{}
	closeOnce sync.Once

	// TODO opts []grpc.ServerOption

	accept   chan grpchub.StreamTransportInterface
	mu       sync.Mutex // guards following
	services map[string]*serviceInfo
}

func (g *GrpcServer) register(sd *grpc.ServiceDesc, ss any) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, ok := g.services[sd.ServiceName]; ok {
		logger.Fatalf("grpcx: Server.RegisterService found duplicate service registration for %q", sd.ServiceName)
	}
	info := &serviceInfo{
		serviceImpl: ss,
		methods:     make(map[string]*grpc.MethodDesc),
		streams:     make(map[string]*grpc.StreamDesc),
		mdata:       sd.Metadata,
	}
	for i := range sd.Methods {
		d := &sd.Methods[i]
		info.methods[d.MethodName] = d
	}
	for i := range sd.Streams {
		d := &sd.Streams[i]
		info.streams[d.StreamName] = d
	}
	g.services[sd.ServiceName] = info
}

func (g *GrpcServer) close() error {
	g.closeOnce.Do(func() {
		g.cancel()
		close(g.closed)
		close(g.accept)
	})
	return nil
}

func (g *GrpcServer) handle(st grpchub.StreamTransportInterface) error {
	defer st.Close()
	ctx := g.Context()
	// ctx = contextWithServer(ctx, g)

	ghStream := grpchub.NewServerStream(ctx, st)
	defer ghStream.Close()

	err := ghStream.RecvHello()
	if err != nil {
		return err
	}

	// logger.Infof("pkg Method: %s\n", ghStream.GetMethod())
	// find method
	service, method, err := parseFullMethod(ghStream.GetMethod())
	// logger.Infof("service: %s, method: %s\n", service, method)
	if err != nil {
		return ghStream.WriteStatus(status.New(codes.Unimplemented, err.Error()))
	}

	// TODO 处理上下文 ctx

	srv, knownService := g.services[service]
	// logger.Infof("got knownService: %v\n", knownService)
	if knownService {
		if md, ok := srv.methods[method]; ok {
			return g.processUnaryRPC(ghStream, srv, md)
		}
		if sd, ok := srv.streams[method]; ok {
			return g.processStreamingRPC(ghStream, srv, sd)
		}
	}

	var errDesc string
	if !knownService {
		errDesc = fmt.Sprintf("unknown service %v", service)
	} else {
		errDesc = fmt.Sprintf("unknown method %v for service %v", method, service)
	}

	return ghStream.WriteStatus(status.New(codes.Unimplemented, errDesc))
}

func (g *GrpcServer) processStreamingRPC(ghStream *grpchub.ServerStream, srv *serviceInfo, sd *grpc.StreamDesc) error {
	// logger.Infof("processStreamingRPC: %s", sd.StreamName)

	stream := newGrpcxServerStream(ghStream)
	defer func() {
		stream.SendClose()
	}()

	// logger.Infof("sd.Handler")
	appErr := sd.Handler(srv.serviceImpl, stream)
	if appErr != nil {
		if appStatus, ok := status.FromError(appErr); ok {
			appErr = stream.WriteStatus(appStatus)
		}
	}

	return appErr
}

func (g *GrpcServer) processUnaryRPC(ghStream *grpchub.ServerStream, srv *serviceInfo, md *grpc.MethodDesc) error {
	// logger.Infof("processUnaryRPC: %s", md.MethodName)

	stream := newGrpcxServerStream(ghStream)
	defer func() {
		stream.SendClose()
	}()

	df := func(v any) error {
		return stream.RecvMsg(v)
	}

	reply, appErr := md.Handler(srv.serviceImpl, ghStream.Context(), df, nil)
	if appErr != nil {
		if appStatus, ok := status.FromError(appErr); ok {
			appErr = stream.WriteStatus(appStatus)
		}
	} else {
		appErr = stream.SendMsg(reply)
	}

	return appErr
}

func (g *GrpcServer) Context() context.Context {
	return g.ctx
}

func (g *GrpcServer) Serve() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	defer g.close()

	for {
		select {
		case <-g.ctx.Done():
			return g.ctx.Err()
		case <-g.closed:
			return nil
		case st := <-g.accept:
			go func() {
				err := g.handle(st)
				if err != nil {
					logger.Errorf("GrpcServer handle err: %s", err)
				}
			}()
		}
	}
}

func (g *GrpcServer) GracefulStop() {
	g.close()
}

// RegisterService implements grpc.ServiceRegistrar.
func (g *GrpcServer) RegisterService(sd *grpc.ServiceDesc, ss any) {
	if ss != nil {
		ht := reflect.TypeOf(sd.HandlerType).Elem()
		st := reflect.TypeOf(ss)
		if !st.Implements(ht) {
			logger.Fatalf("grpcx: Server.RegisterService found the handler of type %v that does not satisfy %v", st, ht)
		}
	}
	g.register(sd, ss)
}

var _ grpc.ServiceRegistrar = (*GrpcServer)(nil)

// GetServiceInfo implements reflection.ServiceInfoProvider.
func (g *GrpcServer) GetServiceInfo() map[string]grpc.ServiceInfo {
	ret := make(map[string]grpc.ServiceInfo)
	for n, srv := range g.services {
		methods := make([]grpc.MethodInfo, 0, len(srv.methods)+len(srv.streams))
		for m := range srv.methods {
			methods = append(methods, grpc.MethodInfo{
				Name:           m,
				IsClientStream: false,
				IsServerStream: false,
			})
		}
		for m, d := range srv.streams {
			methods = append(methods, grpc.MethodInfo{
				Name:           m,
				IsClientStream: d.ClientStreams,
				IsServerStream: d.ServerStreams,
			})
		}

		ret[n] = grpc.ServiceInfo{
			Methods:  methods,
			Metadata: srv.mdata,
		}
	}
	return ret
}

var _ reflection.ServiceInfoProvider = (*GrpcServer)(nil)

func newGrpcServer(ctx context.Context, accept chan grpchub.StreamTransportInterface) *GrpcServer {
	ctx, cancel := context.WithCancel(ctx)

	return &GrpcServer{
		ctx:      ctx,
		cancel:   cancel,
		closed:   make(chan struct{}),
		services: make(map[string]*serviceInfo),
		accept:   accept,
	}
}

// type serverKey struct{}

// // serverFromContext gets the Server from the context.
// func serverFromContext(ctx context.Context) *GrpcServer {
// 	s, _ := ctx.Value(serverKey{}).(*GrpcServer)
// 	return s
// }

// // contextWithServer sets the Server in the context.
// func contextWithServer(ctx context.Context, server *GrpcServer) context.Context {
// 	return context.WithValue(ctx, serverKey{}, server)
// }

// GrpcxServerStream
type GrpcxServerStream struct {
	stream *grpchub.ServerStream
}

func (g *GrpcxServerStream) WriteStatus(st *status.Status) error {
	return g.stream.WriteStatus(st)
}

func (g *GrpcxServerStream) SendClose() error {
	return g.stream.Send(grpchub.PT_CLOSE, nil)
}

// Context implements grpc.ServerStream.
func (g *GrpcxServerStream) Context() context.Context {
	return g.stream.Context()
}

// RecvMsg implements grpc.ServerStream.
func (g *GrpcxServerStream) RecvMsg(m any) error {
	return g.stream.Recv(m)
}

// SendHeader implements grpc.ServerStream.
func (g *GrpcxServerStream) SendHeader(md metadata.MD) error {
	return g.stream.Send(grpchub.PT_HEADER, md)
}

// SendMsg implements grpc.ServerStream.
func (g *GrpcxServerStream) SendMsg(m any) error {
	return g.stream.Send(grpchub.PT_PAYLOAD, m)
}

// SetHeader implements grpc.ServerStream.
func (g *GrpcxServerStream) SetHeader(md metadata.MD) error {
	g.stream.SetHeader(md)
	return nil
}

// SetTrailer implements grpc.ServerStream.
func (g *GrpcxServerStream) SetTrailer(md metadata.MD) {
	g.stream.SetTrailer(md)
}

var _ grpc.ServerStream = (*GrpcxServerStream)(nil)

func newGrpcxServerStream(stream *grpchub.ServerStream) *GrpcxServerStream {
	return &GrpcxServerStream{
		stream: stream,
	}
}
