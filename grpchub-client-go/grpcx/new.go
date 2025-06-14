package grpcx

import (
	"context"
	"fmt"

	"github.com/lisoboss/grpchub"
	"github.com/lisoboss/grpchub/grpchublog"
)

var logger = grpchublog.Component("grpcx")

func NewClient(name string, ghc *grpchub.GrpcHubClient, opts ...ClientOption) (gcConn *GrpcxClientConn, err error) {
	ctx := context.Background()
	ghc.SetId(fmt.Sprintf("%s-cli", name), fmt.Sprintf("%s-ser", name))

	tunnel, err := ghc.Connect(ctx)
	if err != nil {
		return
	}

	sm := grpchub.NewClientStreamManager(ctx,
		grpchub.NewTunnelConn(tunnel),
	)

	go func() {
		if err := sm.Loop(); err != nil {
			logger.Errorf("sm loop err: %s", err)
		}
	}()

	opts = append(opts, WithEndpoint(ghc.Eendpoint()))
	gcConn = newGrpcxClientConn(ctx, sm, opts...)

	return
}

func NewServer(name string, ghc *grpchub.GrpcHubClient, opts ...ServerOption) (gsConn *GrpcServer, err error) {
	ctx := context.Background()
	ghc.SetId(fmt.Sprintf("%s-ser", name), fmt.Sprintf("%s-cli", name))

	tunnel, err := ghc.Connect(ctx)
	if err != nil {
		return
	}

	sm, accept := grpchub.NewServerStreamManager(ctx,
		grpchub.NewTunnelConn(tunnel),
	)

	go func() {
		if err := sm.Loop(); err != nil {
			logger.Errorf("sm loop err: %s", err)
		}
	}()

	opts = append(opts, Endpoint(ghc.Eendpoint()))
	gsConn = newGrpcServer(ctx, accept, opts...)

	return
}
