package utils

import (
	"log/slog"
	"os"
	"testing"

	testpb "grpchub-test/gen/test"
	"grpchub-test/internal/service"

	"github.com/lisoboss/grpchub-go"
	"github.com/lisoboss/grpchub-go/grpcx"
	"github.com/lisoboss/grpchub-go/utils"
)

const (
	hubComponent = "grpchub-test-"
	hubAddr      = "[::1]:50055"
)

var logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))

func newGHC() *grpchub.GrpcHubClient {
	caPEM, certPEM, keyPEM, err := utils.LoadTLSCredentialsFromPEM("./client.pem")
	if err != nil {
		logger.Error("failed to load tls pem", "err", err)
		os.Exit(1)
	}

	ghc, err := grpchub.NewGrpcHubClient(hubAddr, caPEM, certPEM, keyPEM)
	if err != nil {
		logger.Error("failed to init hub client", "err", err)
		os.Exit(1)
	}

	return ghc
}

func StartHubClient(t *testing.T, name string, opts ...grpcx.ClientOption) (testpb.TestServiceClient, func()) {
	// Setup logging.
	ghc := newGHC()

	conn, err := grpcx.NewClient(
		hubComponent+name,
		ghc,
		opts...,
	)
	if err != nil {
		logger.Error("failed to init grpcx client", "err", err)
		os.Exit(1)
	}

	return testpb.NewTestServiceClient(conn), func() {
		conn.Close()
	}
}

func StartHubServer(t *testing.T, name string, opts ...grpcx.ServerOption) (stop func()) {
	start, stop := NewHubServer(t, name, opts...)
	go start()

	return stop
}

func NewHubServer(t *testing.T, name string, opts ...grpcx.ServerOption) (start func(), stop func()) {
	// Setup logging.
	ghc := newGHC()

	grpcSrv, err := grpcx.NewServer(
		hubComponent+name,
		ghc,
		opts...,
	)
	if err != nil {
		logger.Error("failed to init grpcx server", "err", err)
		os.Exit(1)
	}

	// 注册 gRPC 服务
	testpb.RegisterTestServiceServer(grpcSrv, &service.TestService{})

	return func() {
			_ = grpcSrv.Serve()
		}, func() {
			grpcSrv.Stop()
			_ = ghc.Close()

		}
}
