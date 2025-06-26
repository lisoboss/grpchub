package utils

import (
	"encoding/pem"
	"fmt"
	"log/slog"
	"os"
	"testing"

	testpb "grpchub-test/gen/test"
	"grpchub-test/internal/service"

	"github.com/lisoboss/grpchub-go"
	"github.com/lisoboss/grpchub-go/grpcx"
)

const (
	hubComponent = "grpchub-test-"
	hubAddr      = "[::1]:50055"
)

var logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))

func loadTLSCredentialsFromPEM(pemFile string) ([]byte, []byte, []byte, error) {
	data, err := os.ReadFile(pemFile)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read pem file: %w", err)
	}
	var certPEM, keyPEM, caPEM []byte

	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			break
		}
		switch block.Type {
		case "CERTIFICATE":
			// 第一个 CERTIFICATE 是 client cert，最后一个是 CA
			if certPEM == nil {
				certPEM = pem.EncodeToMemory(block)
			} else {
				caPEM = append(caPEM, pem.EncodeToMemory(block)...)
			}
		case "PRIVATE KEY":
			keyPEM = pem.EncodeToMemory(block)
		default:
			// 忽略其他类型
		}
	}

	if certPEM == nil || keyPEM == nil || caPEM == nil {
		return nil, nil, nil, fmt.Errorf("incomplete PEM data (cert/key/ca)")
	}

	return caPEM, certPEM, keyPEM, nil

}

func newGHC() *grpchub.GrpcHubClient {
	caPEM, certPEM, keyPEM, err := loadTLSCredentialsFromPEM("./client.pem")
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
