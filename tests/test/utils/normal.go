package utils

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"runtime/debug"
	"testing"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/selector"
	testpb "github.com/lisoboss/grpchub-test/gen/test"
	"github.com/lisoboss/grpchub-test/internal/service"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

const (
	component = "grpchub-test"
	grpcAddr  = ":8080"
)

// interceptorLogger adapts slog logger to interceptor logger.
// This code is simple enough to be copied and not imported.
func interceptorLogger(l *slog.Logger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(lvl), msg, fields...)
	})
}

func StartServer(t *testing.T, enauth bool) (addr string, stop func()) {
	// Setup logging.
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))
	rpcLogger := logger.With("service", "gRPC/server", "component", component)
	logTraceID := func(ctx context.Context) logging.Fields {
		if span := trace.SpanContextFromContext(ctx); span.IsSampled() {
			return logging.Fields{"traceID", span.TraceID().String()}
		}
		return nil
	}

	// Setup metrics.
	srvMetrics := grpcprom.NewServerMetrics(
		grpcprom.WithServerHandlingTimeHistogram(
			grpcprom.WithHistogramBuckets([]float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120}),
		),
	)
	reg := prometheus.NewRegistry()
	reg.MustRegister(srvMetrics)
	exemplarFromContext := func(ctx context.Context) prometheus.Labels {
		if span := trace.SpanContextFromContext(ctx); span.IsSampled() {
			return prometheus.Labels{"traceID": span.TraceID().String()}
		}
		return nil
	}

	// Set up OTLP tracing (stdout for debug).
	exporter, err := stdout.New(stdout.WithPrettyPrint())
	if err != nil {
		logger.Error("failed to init exporter", "err", err)
		os.Exit(1)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	// Setup custom auth.
	authFn := func(ctx context.Context) (context.Context, error) {
		if !enauth {
			return ctx, nil
		}
		token, err := auth.AuthFromMD(ctx, "bearer")
		if err != nil {
			return nil, err
		}
		// TODO: This is example only, perform proper Oauth/OIDC verification!
		if token != "yolo" {
			return nil, status.Error(codes.Unauthenticated, "invalid auth token")
		}
		// NOTE: You can also pass the token in the context for further interceptors or gRPC service code.
		return ctx, nil
	}

	// Setup auth matcher.
	allButHealthZ := func(ctx context.Context, callMeta interceptors.CallMeta) bool {
		return healthpb.Health_ServiceDesc.ServiceName != callMeta.Service
	}

	// Setup metric for panic recoveries.
	panicsTotal := promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Name: "grpc_req_panics_recovered_total",
		Help: "Total number of gRPC requests recovered from internal panic.",
	})
	grpcPanicRecoveryHandler := func(p any) (err error) {
		panicsTotal.Inc()
		rpcLogger.Error("recovered from panic", "panic", p, "stack", debug.Stack())
		fmt.Printf("stack: %s\n", debug.Stack())
		return status.Errorf(codes.Internal, "%s", p)
	}

	grpcSrv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			srvMetrics.UnaryServerInterceptor(grpcprom.WithExemplarFromContext(exemplarFromContext)),
			logging.UnaryServerInterceptor(interceptorLogger(rpcLogger), logging.WithFieldsFromContext(logTraceID)),
			selector.UnaryServerInterceptor(auth.UnaryServerInterceptor(authFn), selector.MatchFunc(allButHealthZ)),
			recovery.UnaryServerInterceptor(recovery.WithRecoveryHandler(grpcPanicRecoveryHandler)),
		),
		grpc.ChainStreamInterceptor(
			srvMetrics.StreamServerInterceptor(grpcprom.WithExemplarFromContext(exemplarFromContext)),
			logging.StreamServerInterceptor(interceptorLogger(rpcLogger), logging.WithFieldsFromContext(logTraceID)),
			selector.StreamServerInterceptor(auth.StreamServerInterceptor(authFn), selector.MatchFunc(allButHealthZ)),
			recovery.StreamServerInterceptor(recovery.WithRecoveryHandler(grpcPanicRecoveryHandler)),
		),
	)

	// 注册 gRPC 服务
	testpb.RegisterTestServiceServer(grpcSrv, &service.TestService{})
	srvMetrics.InitializeMetrics(grpcSrv)

	lis, err := net.Listen("tcp", grpcAddr) // 自动分配端口
	require.NoError(t, err)

	go func() {
		_ = grpcSrv.Serve(lis)
	}()

	return lis.Addr().String(), func() {
		_ = exporter.Shutdown(context.Background())
		grpcSrv.GracefulStop()
		_ = lis.Close()
	}
}

func StartClient(t *testing.T, addr string) (testpb.TestServiceClient, func()) {
	// Setup logging.
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))
	// rpcLogger := logger.With("service", "gRPC/client", "component", component)
	// logTraceID := func(ctx context.Context) logging.Fields {
	// 	if span := trace.SpanContextFromContext(ctx); span.IsSampled() {
	// 		return logging.Fields{"traceID", span.TraceID().String()}
	// 	}
	// 	return nil
	// }

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("failed to init client", "err", err)
		os.Exit(1)
	}

	return testpb.NewTestServiceClient(conn), func() {
		_ = conn.Close()
	}
}
