package test

import (
	"context"
	"testing"

	"github.com/lisoboss/grpchub-test/test/utils"
	"github.com/lisoboss/grpchub/grpcx"
	"github.com/lisoboss/grpchub/middleware"
	"google.golang.org/grpc/metadata"
)

func TestNormalService_Error(t *testing.T) {
	addr, stopS := utils.StartServer(t, false)
	defer stopS()
	client, stopC := utils.StartClient(t, addr)
	defer stopC()

	ErrorCall(t, client)
}

func TestNormalService_NoAuth(t *testing.T) {
	addr, stopS := utils.StartServer(t, false)
	defer stopS()
	client, stopC := utils.StartClient(t, addr)
	defer stopC()

	EmptyCall(t, client)
	ErrorCall(t, client)
}

func TestNormalService_Auth(t *testing.T) {
	addr, stopS := utils.StartServer(t, true)
	defer stopS()
	client, stopC := utils.StartClient(t, addr)
	defer stopC()
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer yolo")

	AuthCall(t, client, ctx)
	BidirectionalStream(t, client, ctx)
}

func TestHubService_NoAuth(t *testing.T) {
	stopS := utils.StartHubServer(t, "no-auth")
	defer stopS()
	client, stopC := utils.StartHubClient(t, "no-auth")
	defer stopC()

	EmptyCall(t, client)
	ErrorCall(t, client)
}

func TestHubService_Auth(t *testing.T) {
	stopS := utils.StartHubServer(t, "auth",
		grpcx.Middleware(
			Auth("111111"),
		),
		grpcx.StreamTransportMiddleware(
			StreamAuth("222222"),
		),
	)
	defer stopS()
	client, stopC := utils.StartHubClient(t, "auth",
		grpcx.WithMiddleware(
			WithAuth("111111"),
		),
		grpcx.WithStreamTransportMiddleware(
			WithStreamAuth("222222"),
		))
	defer stopC()
	ctx := context.Background()

	AuthCall(t, client, ctx)
	BidirectionalStream(t, client, ctx)
}

func TestHubService_WrappedMiddleware(t *testing.T) {
	stopS := utils.StartHubServer(t, "auth",
		grpcx.Middleware(
			Auth("111111"),
		),
		grpcx.StreamTransportMiddleware(
			middleware.NewWrappedStreamTransportMiddleware(
				Auth("222222"),
			)...,
		),
	)
	defer stopS()
	client, stopC := utils.StartHubClient(t, "auth",
		grpcx.WithMiddleware(
			WithAuth("111111"),
		),
		grpcx.WithStreamTransportMiddleware(
			middleware.NewWrappedStreamTransportMiddleware(
				WithAuth("222222"),
			)...,
		))
	defer stopC()
	ctx := context.Background()

	AuthCall(t, client, ctx)
	BidirectionalStream(t, client, ctx)
}
