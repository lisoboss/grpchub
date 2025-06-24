package grpchub

import (
	"context"

	channelv1 "github.com/lisoboss/grpchub/gen/channel/v1"
	_ "github.com/mostynb/go-grpc-compression/zstd" // zstd 压缩器
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type GrpcHubClient struct {
	senderId   string
	receiverId string
	cClient    channelv1.ChannelServiceClient
	closeFn    func() error
}

func (c *GrpcHubClient) SetId(senderId, receiverId string) {
	c.senderId, c.receiverId = senderId, receiverId
}

// TODO Eendpoint is senderId or receiverId ?
func (c *GrpcHubClient) Eendpoint() string {
	return c.senderId
}

func (c *GrpcHubClient) Connect(ctx context.Context) (channelv1.ChannelService_ChannelClient, error) {
	md := metadata.Pairs(
		"sender_id", c.senderId,
		"receiver_id", c.receiverId,
	)
	ctx = metadata.NewOutgoingContext(ctx, md)
	return c.cClient.Channel(ctx)
}

func (c *GrpcHubClient) Close() error {
	return c.closeFn()
}

func NewGrpcHubClientFrom(cc grpc.ClientConnInterface, closeFn func() error) (c *GrpcHubClient, err error) {
	c = &GrpcHubClient{
		cClient: channelv1.NewChannelServiceClient(cc),
		closeFn: closeFn,
	}
	return
}

func NewGrpcHubClient(target string, caPEMBlock, certPEMBlock, keyPEMBlock []byte, opts ...grpc.DialOption) (c *GrpcHubClient, err error) {
	defaultOpts, err := defaultGrpchubDialOptionsWithTLS(caPEMBlock, certPEMBlock, keyPEMBlock)
	if err != nil {
		return nil, err
	}
	opts = append(defaultOpts, opts...)

	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		return
	}

	return NewGrpcHubClientFrom(conn, func() error {
		return conn.Close()
	})
}
