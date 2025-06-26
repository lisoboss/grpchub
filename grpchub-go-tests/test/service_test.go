package test

import (
	"context"
	"testing"

	"grpchub-test/test/utils"

	"github.com/lisoboss/grpchub-go/grpcx"
	"github.com/lisoboss/grpchub-go/middleware"
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
	var name = "no-auth"
	stopS := utils.StartHubServer(t, name)
	defer stopS()
	client, stopC := utils.StartHubClient(t, name)
	defer stopC()

	EmptyCall(t, client)
	ErrorCall(t, client)
}

func TestHubService_Auth(t *testing.T) {
	var name = "auth"
	stopS := utils.StartHubServer(t, name,
		grpcx.Middleware(
			Auth("111111"),
		),
		grpcx.StreamTransportMiddleware(
			StreamAuth("222222"),
		),
	)
	defer stopS()
	client, stopC := utils.StartHubClient(t, name,
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
	var name = "wrapped-middleware"
	stopS := utils.StartHubServer(t, name,
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
	client, stopC := utils.StartHubClient(t, name,
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
