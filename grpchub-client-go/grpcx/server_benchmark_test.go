package grpcx

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lisoboss/grpchub"
	channelv1 "github.com/lisoboss/grpchub/gen/channel/v1"
	"github.com/lisoboss/grpchub/transport"
	"github.com/lisoboss/grpchub/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
)

// Mock stream transport for benchmarking
type mockStreamTransport struct {
	ctx       context.Context
	cancel    context.CancelFunc
	r         chan *channelv1.MessagePackage
	s         chan *channelv1.MessagePackage
	method    string
	stream    *transport.ServerStream[*channelv1.MessagePackage]
	closeOnce sync.Once
}

func (m *mockStreamTransport) Close() error {
	m.closeOnce.Do(func() {
		m.r <- &channelv1.MessagePackage{Type: channelv1.PackageType_PT_CLOSE, Method: m.method}
		m.cancel()
	})
	return nil
}

func (m *mockStreamTransport) Send(p *channelv1.MessagePackage) error {
	select {
	case <-m.ctx.Done():
		return m.ctx.Err()
	case m.s <- p:
		return nil
	}

}

func (m *mockStreamTransport) loop() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case reply := <-m.r:
			if m.stream != nil {
				m.stream.Processe(reply)
			}
		}
	}
}

func (s *mockStreamTransport) newServerStream() *transport.ServerStream[*channelv1.MessagePackage] {
	s.stream = transport.NewServerStream(transport.NewStream(&grpchub.WrappedMessagePackage{}, func(m *channelv1.MessagePackage) error {
		return s.Send(m)
	}))
	go s.loop()
	return s.stream
}

func (m *mockStreamTransport) reply() *channelv1.MessagePackage {
	var r *channelv1.MessagePackage
	for {
		r = <-m.s
		switch r.Type {
		case channelv1.PackageType_PT_PAYLOAD, channelv1.PackageType_PT_ERROR:
			return r
		default:
			// logger.Info(r)
		}
	}

}

func newMockStreamTransport(ctx context.Context) *mockStreamTransport {
	ctx, cancel := context.WithCancel(ctx)
	r := make(chan *channelv1.MessagePackage, 10)
	m := &mockStreamTransport{
		ctx:    ctx,
		cancel: cancel,
		r:      r,
		s:      make(chan *channelv1.MessagePackage, 10),
		method: "/test.MockService/TestMethod",
	}
	r <- &channelv1.MessagePackage{Type: channelv1.PackageType_PT_HELLO, Method: m.method}
	p, _ := anypb.New(&testRequest{
		ChannelMessage: channelv1.ChannelMessage{
			Sid: "testRequest",
		},
	})
	r <- &channelv1.MessagePackage{Type: channelv1.PackageType_PT_PAYLOAD, Method: m.method, Payload: p}
	return m
}

type mockServiceInterface interface {
	testMethod(context.Context, *testRequest) (*testResponse, error)
}

// Mock service for testing
type mockService struct{}

func (m *mockService) testMethod(ctx context.Context, req *testRequest) (*testResponse, error) {
	return &testResponse{ChannelMessage: channelv1.ChannelMessage{
		Sid: req.Sid + "---rep",
	}}, nil
}

type testRequest struct {
	channelv1.ChannelMessage
}

type testResponse struct {
	channelv1.ChannelMessage
}

var testServiceDesc = grpc.ServiceDesc{
	ServiceName: "test.MockService",
	HandlerType: (*mockServiceInterface)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "TestMethod",
			Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				in := new(testRequest)
				if err := dec(in); err != nil {
					return nil, err
				}
				if interceptor == nil {
					return srv.(*mockService).testMethod(ctx, in)
				}
				info := &grpc.UnaryServerInfo{
					Server:     srv,
					FullMethod: "/test.MockService/TestMethod",
				}
				handler := func(ctx context.Context, req interface{}) (interface{}, error) {
					return srv.(*mockService).testMethod(ctx, req.(*testRequest))
				}
				return interceptor(ctx, in, info, handler)
			},
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "test.proto",
}

func BenchmarkServerConcurrency(b *testing.B) {
	tests := []struct {
		name       string
		workers    int
		concurrent int
	}{
		{"NoPool_10Workers", 0, 50},
		{"WithPool_10Workers", 10, 50},
		{"NoPool_100Workers", 0, 500},
		{"WithPool_100Workers", 100, 500},
		{"NoPool_1000Workers", 0, 2000},
		{"WithPool_1000Workers", 1000, 2000},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			accept := make(chan *transport.ServerStream[*channelv1.MessagePackage], tt.concurrent)

			var opts []ServerOption
			opts = append(opts, MaxWorker(tt.workers))

			server := newServer(ctx, func() <-chan *transport.ServerStream[*channelv1.MessagePackage] { return accept }, opts...)
			server.RegisterService(&testServiceDesc, &mockService{})

			go server.Serve()
			defer server.Stop()

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					mockTransport := newMockStreamTransport(server.Context())
					select {
					case accept <- mockTransport.newServerStream():
						// Simulate some processing time
						// time.Sleep(time.Millisecond)
						// b.Logf("reply: %#v\n", mockTransport.reply())
						mockTransport.reply()
					case <-time.After(time.Second):
						b.Fatal("timeout sending to accept channel")
					}
					mockTransport.Close()
				}
			})
		})
	}
}

func BenchmarkServiceInfoCache(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	accept := make(chan *transport.ServerStream[*channelv1.MessagePackage], 1)
	server := newServer(ctx, func() <-chan *transport.ServerStream[*channelv1.MessagePackage] { return accept })

	// Register multiple services
	for i := range 100 {
		desc := testServiceDesc
		desc.ServiceName = fmt.Sprintf("test.TestService%d", i)
		server.RegisterService(&desc, &mockService{})
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = server.GetServiceInfo()
		}
	})
}

func BenchmarkErrorStatusCreation(b *testing.B) {
	b.Run("CachedStatus", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = statusTooManyRequests
			}
		})
	})

	b.Run("NewStatus", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = status.New(codes.Unimplemented, "unknown service")
			}
		})
	})
}

func BenchmarkWorkerPool(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var processed int64
	// var transports []*mockStreamTransport
	handler := func(stream *transport.ServerStream[*channelv1.MessagePackage]) {
		atomic.AddInt64(&processed, 1)
		// time.Sleep(time.Millisecond)
		stream.Close()
	}

	pool := utils.NewWorkerPool(ctx, 100, handler)
	defer pool.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if !pool.SubmitBlock(newMockStreamTransport(ctx).newServerStream()) {
				b.Fatal("failed to submit work")
			}
		}
	})

	// Wait for all work to complete
	for atomic.LoadInt64(&processed) < int64(b.N) {
		time.Sleep(time.Microsecond)
	}
}
