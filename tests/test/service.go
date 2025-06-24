package test

import (
	"context"
	"fmt"
	"testing"

	testpb "github.com/lisoboss/grpchub-test/gen/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func EmptyCall(t *testing.T, client testpb.TestServiceClient) {

}
func UnaryCall(t *testing.T, client testpb.TestServiceClient) {
	ctx := context.Background()
	req := &testpb.UnaryRequest{
		Message: "Test Unary",
		Number:  4523,
		Tags:    []string{"test unary 1", "test unary 2"},
		Timestamp: &timestamppb.Timestamp{
			Seconds: 1000000,
		},
	}
	resp, err := client.UnaryCall(ctx, req)
	require.NoError(t, err)

	assert.Contains(t, resp.Result, fmt.Sprintf("Processed: %s", req.Message))
	assert.Equal(t, resp.ProcessedNumber, req.Number*2)
	assert.Equal(t, resp.TagCount, int32(len(req.Tags)))
	assert.Equal(t, resp.ServerTimestamp.Seconds, req.Timestamp.Seconds+1000)
}
func ErrorCall(t *testing.T, client testpb.TestServiceClient) {
	ctx := context.Background()
	req := &testpb.ErrorRequest{
		ErrorType: testpb.ErrorType_ERROR_TYPE_NOT_FOUND,
		Message:   "ErrorRequest",
	}
	_, err := client.ErrorCall(ctx, req)
	require.Error(t, err)

	assert.Equal(t, err.Error(), fmt.Sprintf("rpc error: code = NotFound desc = Not found: %s", req.Message))
}
func ClientStream(t *testing.T, client testpb.TestServiceClient) {}
func ServerStream(t *testing.T, client testpb.TestServiceClient) {}
func BidirectionalStream(t *testing.T, client testpb.TestServiceClient, ctx context.Context) {
	stream, err := client.BidirectionalStream(ctx)
	require.NoError(t, err)
	defer func() {
		err := stream.CloseSend()
		require.NoError(t, err)
	}()

	req := &testpb.BidirectionalRequest{
		Message: "1111111",
		Id:      1,
		Type:    testpb.RequestType_REQUEST_TYPE_ECHO,
	}
	err = stream.Send(req)
	require.NoError(t, err)
	reply, err := stream.Recv()
	require.NoError(t, err)
	assert.Equal(t, reply.Echo, fmt.Sprintf("Echo: %s", req.Message))

	req = &testpb.BidirectionalRequest{
		Message: "11111112",
		Id:      1,
		Type:    testpb.RequestType_REQUEST_TYPE_ECHO,
	}
	err = stream.Send(req)
	require.NoError(t, err)
	reply, err = stream.Recv()
	require.NoError(t, err)
	assert.Equal(t, reply.Echo, fmt.Sprintf("Echo: %s", req.Message))

	reqs := map[int]*testpb.BidirectionalRequest{
		1: {
			Message: "11111112",
			Id:      1,
			Type:    testpb.RequestType_REQUEST_TYPE_ECHO,
		},
		2: {
			Message: "11111112",
			Id:      2,
			Type:    testpb.RequestType_REQUEST_TYPE_ECHO,
		},
	}

	err = stream.Send(reqs[1])
	require.NoError(t, err)
	err = stream.Send(reqs[2])
	require.NoError(t, err)

	reply, err = stream.Recv()
	require.NoError(t, err)
	assert.Equal(t, reply.Echo, fmt.Sprintf("Echo: %s", reqs[int(reply.RequestId)].Message))

	reply, err = stream.Recv()
	require.NoError(t, err)
	assert.Equal(t, reply.Echo, fmt.Sprintf("Echo: %s", reqs[int(reply.RequestId)].Message))
}
func LargeDataCall(t *testing.T, client testpb.TestServiceClient) {}
func TimeoutCall(t *testing.T, client testpb.TestServiceClient)   {}
func MetadataCall(t *testing.T, client testpb.TestServiceClient)  {}
func AuthCall(t *testing.T, client testpb.TestServiceClient, ctx context.Context) {
	req := &testpb.AuthRequest{
		Token:  "valid-token-123",
		UserId: "1111",
	}
	resp, err := client.AuthCall(ctx, req)
	require.NoError(t, err)

	assert.Equal(t, resp.Authenticated, true)
	assert.Equal(t, resp.UserInfo, "user1")
	assert.Equal(t, resp.Permissions, []string{"read", "write"})
}
