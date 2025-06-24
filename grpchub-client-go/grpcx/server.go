package grpcx

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/lisoboss/grpchub/middleware"
	"github.com/lisoboss/grpchub/transport"
	"github.com/lisoboss/grpchub/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var (
	statusTooManyRequests = status.New(codes.ResourceExhausted, "too many requests")
)

type serviceInfo struct {
	serviceImpl any
	methods     map[string]*grpc.MethodDesc
	streams     map[string]*grpc.StreamDesc
	mdata       any
}

type Server[Inner any] struct {
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	closeOnce sync.Once

	opts serverOptions

	accept func() <-chan *transport.ServerStream[Inner]

	mu       sync.RWMutex // guards services
	services map[string]*serviceInfo

	// Cached service info to avoid recreating on every call
	serviceInfoCache atomic.Value // map[string]grpc.ServiceInfo
	serviceInfoDirty int32

	pool *utils.WorkerPool[*transport.ServerStream[Inner]]
}

func (g *Server[Inner]) register(sd *grpc.ServiceDesc, ss any) {
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

	// Mark service info cache as dirty
	atomic.StoreInt32(&g.serviceInfoDirty, 1)
}

func (g *Server[Inner]) close() {
	g.cancel()
	g.wg.Wait()

	if g.pool != nil {
		g.pool.Close()
	}
}

func (g *Server[Inner]) Close() {
	g.closeOnce.Do(g.close)
}

func (g *Server[Inner]) replyErr(stream *transport.ServerStream[Inner]) {
	defer g.wg.Done()

	defer stream.Close()
	ctx := context.Background()

	stream.WaitHandshake()

	wrappedStream := newWrappedServerStream(ctx, stream, g.opts.streamMacter)

	if err := wrappedStream.WriteStatus(statusTooManyRequests); err != nil {
		logger.Errorf("failed to write status: %v", err)
	}
}

func (g *Server[Inner]) handle(stream *transport.ServerStream[Inner]) {
	defer g.wg.Done()

	defer stream.Close()
	ctx := context.Background()

	stream.WaitHandshake()

	wrappedStream := newWrappedServerStream(ctx, stream, g.opts.streamMacter)

	// Find method
	service, method, err := parseFullMethod(stream.GetMethod())
	if err != nil {
		if err := wrappedStream.WriteStatus(status.New(codes.Unimplemented, err.Error())); err != nil {
			logger.Errorf("failed to write status: %v", err)
		}
		return
	}

	g.mu.RLock()
	srv, knownService := g.services[service]
	g.mu.RUnlock()

	if knownService {
		if md, ok := srv.methods[method]; ok {
			if err := g.processUnaryRPC(wrappedStream, srv, md); err != nil {
				logger.Errorf("unary RPC error: %v", err)
			}
			return
		}
		if sd, ok := srv.streams[method]; ok {
			if err := g.processStreamingRPC(wrappedStream, srv, sd); err != nil {
				logger.Errorf("streaming RPC error: %v", err)
			}
			return
		}
	}

	var errStatus *status.Status
	if !knownService {
		errStatus = status.New(codes.Unimplemented, fmt.Sprintf("unknown service %v", service))
	} else {
		errStatus = status.New(codes.Unimplemented, fmt.Sprintf("unknown method %v for service %v", method, service))
	}

	if err := wrappedStream.WriteStatus(errStatus); err != nil {
		logger.Errorf("failed to write status: %v", err)
	}
}

func (g *Server[Inner]) processStreamingRPC(stream *WrappedServerStream[Inner], srv *serviceInfo, sd *grpc.StreamDesc) error {

	ctx := NewSTransportContext(stream.Context(), g.opts.endpoint, stream.GetStream())
	if g.opts.streamTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, g.opts.streamTimeout)
		defer cancel()
	}

	stream.SetContext(ctx)

	h := func(ctx context.Context) error {
		stream.SetContext(ctx)
		return sd.Handler(srv.serviceImpl, stream)
		// g.processStreamingRPCRaw(stream, srv, sd)
	}

	if len(g.opts.streamTransportMiddleware) > 0 {
		h = middleware.StreamTransportChain(g.opts.streamTransportMiddleware...)(h)
	}

	return g.processStreamingRPCRaw(stream, h)
}

func (g *Server[Inner]) processStreamingRPCRaw(stream *WrappedServerStream[Inner], hendler func(ctx context.Context) error) error {
	appErr := hendler(stream.Context())
	if appErr != nil {
		if appStatus, ok := status.FromError(appErr); ok {
			return stream.WriteStatus(appStatus)
		}
	}
	return appErr
}

func (g *Server[Inner]) processUnaryRPC(stream *WrappedServerStream[Inner], srv *serviceInfo, md *grpc.MethodDesc) error {
	return g.processUnaryRPCRaw(stream, srv, md, func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		ctx = NewSTransportContext(ctx, g.opts.endpoint, stream.GetStream())

		if g.opts.timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, g.opts.timeout)
			defer cancel()
		}

		h := func(ctx context.Context, req any) (reply any, err error) {
			return handler(ctx, req)
		}

		if len(g.opts.middleware) > 0 {
			h = middleware.Chain(g.opts.middleware...)(h)
		}

		return h(ctx, req)
	})
}

func (g *Server[Inner]) processUnaryRPCRaw(stream *WrappedServerStream[Inner], srv *serviceInfo, md *grpc.MethodDesc, interceptor grpc.UnaryServerInterceptor) error {
	df := func(v any) error {
		return stream.RecvMsgRaw(v)
	}

	reply, appErr := md.Handler(srv.serviceImpl, stream.Context(), df, interceptor)
	if appErr != nil {
		if appStatus, ok := status.FromError(appErr); ok {
			return stream.WriteStatus(appStatus)
		}
		return appErr
	}

	return stream.SendMsgRaw(reply)
}

func (g *Server[Inner]) Context() context.Context {
	return g.ctx
}

func (g *Server[Inner]) Serve() error {
	defer g.close()

	// Create worker pool
	var poolEnabled = false
	if g.opts.maxWorker > 0 {
		poolEnabled = true
	}

	if poolEnabled {
		g.pool = utils.NewWorkerPool(g.ctx, g.opts.maxWorker, func(stream *transport.ServerStream[Inner]) {
			g.handle(stream)
		})
	}

	for {
		select {
		case <-g.ctx.Done():
			return g.ctx.Err()
		case stream, ok := <-g.accept():
			if !ok {
				return fmt.Errorf("accept chan closed")
			}
			g.wg.Add(1)
			if poolEnabled {
				// Try to submit work to pool, fallback to direct handling if pool is full
				if !g.pool.Submit(stream) {
					// Pool is full or closed, reply Err
					go g.replyErr(stream)
				}
			} else {
				go g.handle(stream)
			}

		}
	}
}

func (g *Server[Inner]) GracefulStop() {
	g.Stop()
}

func (g *Server[Inner]) Stop() {
	g.close()
}

func (g *Server[Inner]) RegisterService(sd *grpc.ServiceDesc, ss any) {
	if ss != nil {
		ht := reflect.TypeOf(sd.HandlerType).Elem()
		st := reflect.TypeOf(ss)
		if !st.Implements(ht) {
			logger.Fatalf("grpcx: Server.RegisterService found the handler of type %v that does not satisfy %v", st, ht)
		}
	}
	g.register(sd, ss)
}

var _ grpc.ServiceRegistrar = (*Server[any])(nil)

func (g *Server[Inner]) GetServiceInfo() map[string]grpc.ServiceInfo {
	// Check if cache is dirty
	if atomic.LoadInt32(&g.serviceInfoDirty) == 0 {
		if cached := g.serviceInfoCache.Load(); cached != nil {
			return cached.(map[string]grpc.ServiceInfo)
		}
	}

	// Rebuild cache
	g.mu.RLock()
	ret := make(map[string]grpc.ServiceInfo, len(g.services))
	for n, srv := range g.services {
		methodsLen := len(srv.methods) + len(srv.streams)
		methods := make([]grpc.MethodInfo, 0, methodsLen)

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
	g.mu.RUnlock()

	// Update cache
	g.serviceInfoCache.Store(ret)
	atomic.StoreInt32(&g.serviceInfoDirty, 0)

	return ret
}

var _ reflection.ServiceInfoProvider = (*Server[any])(nil)

func newServer[Inner any](ctx context.Context, accept func() <-chan *transport.ServerStream[Inner], opts ...ServerOption) *Server[Inner] {
	ctx, cancel := context.WithCancel(ctx)
	return &Server[Inner]{
		ctx:      ctx,
		cancel:   cancel,
		services: make(map[string]*serviceInfo),
		accept:   accept,
		opts:     parseServerOptions(opts),
	}
}

type WrappedServerStream[Inner any] struct {
	ctx        context.Context
	stream     *transport.ServerStream[Inner]
	middleware *middleware.Matcher
}

func (w *WrappedServerStream[Inner]) WriteStatus(status *status.Status) error {
	return w.stream.WriteError(status.Err())
}

func (w *WrappedServerStream[Inner]) GetStream() *transport.ServerStream[Inner] {
	return w.stream
}

func (w *WrappedServerStream[Inner]) CloseSend() error {
	return w.stream.SendClose()
}

// SendHeader implements grpc.ServerStream.
func (w *WrappedServerStream[Inner]) SendHeader(md metadata.MD) error {
	return w.stream.SendHeader(md)
}

func (w *WrappedServerStream[Inner]) RecvMsgRaw(m any) error {
	reply, err := w.stream.Recv()
	if err != nil {
		return err
	}
	proto.Merge(m.(proto.Message), reply.(proto.Message))
	return nil
}

// RecvMsg implements grpc.ServerStream.
func (w *WrappedServerStream[Inner]) RecvMsg(m any) error {
	h := func(_ context.Context, req any) (any, error) {
		return req, w.RecvMsgRaw(req)
	}

	if next := w.middleware.Match(w.stream.GetMethod()); len(next) > 0 {
		h = middleware.Chain(next...)(h)
	}

	_, err := h(w.Context(), m)
	return err
}

func (w *WrappedServerStream[Inner]) SendMsgRaw(m any) error {
	return w.stream.SendPayload(m.(proto.Message))
}

// SendMsg implements grpc.ServerStream.
func (w *WrappedServerStream[Inner]) SendMsg(m any) error {
	h := func(_ context.Context, req any) (any, error) {
		return req, w.SendMsgRaw(req)
	}

	if next := w.middleware.Match(w.stream.GetMethod()); len(next) > 0 {
		h = middleware.Chain(next...)(h)
	}

	_, err := h(w.Context(), m)
	return err
}

// SetHeader implements grpc.ServerStream.
func (w *WrappedServerStream[Inner]) SetHeader(md metadata.MD) error {
	w.stream.RequestHeader().Extend(map[string][]string(md))
	return nil
}

// SetTrailer implements grpc.ServerStream.
func (w *WrappedServerStream[Inner]) SetTrailer(md metadata.MD) {
	w.stream.ReplyHeader().Extend(map[string][]string(md))
}

func (w *WrappedServerStream[Inner]) Context() context.Context {
	return w.ctx
}

func (w *WrappedServerStream[Inner]) SetContext(ctx context.Context) {
	w.ctx = ctx
}

var _ grpc.ServerStream = (*WrappedServerStream[any])(nil)

func newWrappedServerStream[Inner any](ctx context.Context, stream *transport.ServerStream[Inner], middleware *middleware.Matcher) *WrappedServerStream[Inner] {
	return &WrappedServerStream[Inner]{
		ctx:        ctx,
		middleware: middleware,
		stream:     stream,
	}
}
