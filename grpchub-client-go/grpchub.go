package grpchub

import (
	"context"

	channelv1 "github.com/lisoboss/grpchub/gen/channel/v1"
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

func NewGrpcHubClient(target string, opts ...grpc.DialOption) (c *GrpcHubClient, err error) {
	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		return
	}

	return NewGrpcHubClientFrom(conn, func() error {
		return conn.Close()
	})
}
