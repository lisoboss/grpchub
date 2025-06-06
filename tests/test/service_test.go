package test

import (
	"context"
	"testing"

	"github.com/lisoboss/grpchub-test/test/utils"
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
	stopS := utils.StartHubServer(t, false)
	defer stopS()
	client, stopC := utils.StartHubClient(t)
	defer stopC()

	EmptyCall(t, client)
	ErrorCall(t, client)
}

func TestHubService_Auth(t *testing.T) {
	stopS := utils.StartHubServer(t, true)
	defer stopS()
	client, stopC := utils.StartHubClient(t)
	defer stopC()
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer yolo")

	AuthCall(t, client, ctx)
	BidirectionalStream(t, client, ctx)
}
