package grpcx

import (
	"context"
	"fmt"

	"github.com/lisoboss/grpchub"
	channelv1 "github.com/lisoboss/grpchub/gen/channel/v1"
	"github.com/lisoboss/grpchub/grpchublog"
	"github.com/lisoboss/grpchub/transport"
)

var logger = grpchublog.Component("grpcx")

func NewClient(name string, ghc *grpchub.GrpcHubClient, opts ...ClientOption) (gcConn *ClientConn[*channelv1.MessagePackage], err error) {
	ctx := context.Background()
	ghc.SetId(fmt.Sprintf("%s-cli", name), fmt.Sprintf("%s-ser", name))

	tunnel, err := ghc.Connect(ctx)
	if err != nil {
		return
	}

	sm := grpchub.NewClientStreamManager(
		&grpchub.WrappedMessage{},
		&grpchub.WrappedMessagePackage{},
		transport.NewTunnel(tunnel),
	)

	go func() {
		if err := sm.Loop(); err != nil {
			logger.Errorf("sm loop err: %s", err)
		}
	}()

	opts = append(opts, WithEndpoint(ghc.Eendpoint()))
	gcConn = NewClientConn(ctx, sm, opts...)

	return
}

func NewServer(name string, ghc *grpchub.GrpcHubClient, opts ...ServerOption) (gsConn *Server[*channelv1.MessagePackage], err error) {
	ctx := context.Background()
	ghc.SetId(fmt.Sprintf("%s-ser", name), fmt.Sprintf("%s-cli", name))

	tunnel, err := ghc.Connect(ctx)
	if err != nil {
		return
	}

	sm := grpchub.NewServerStreamManager(
		&grpchub.WrappedMessage{},
		&grpchub.WrappedMessagePackage{},
		transport.NewTunnel(tunnel),
	)

	go func() {
		if err := sm.Loop(); err != nil {
			logger.Errorf("sm loop err: %s", err)
		}
	}()

	opts = append(opts, Endpoint(ghc.Eendpoint()))
	gsConn = newServer(ctx, sm.Accept, opts...)

	return
}
