package service

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"strings"
	"time"

	testpb "grpchub-test/gen/test"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestService 实现所有测试服务方法
type TestService struct {
	testpb.UnimplementedTestServiceServer
}

// UnaryCall 一元调用实现
func (s *TestService) UnaryCall(ctx context.Context, req *testpb.UnaryRequest) (*testpb.UnaryResponse, error) {
	// 检查元数据
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if values := md.Get("test-header"); len(values) > 0 {
			// 设置响应头
			header := metadata.Pairs("response-header", "unary-response")
			grpc.SetHeader(ctx, header)

			// 设置响应尾部
			trailer := metadata.Pairs("response-trailer", "unary-trailer")
			grpc.SetTrailer(ctx, trailer)
		}
	}

	return &testpb.UnaryResponse{
		Result:          fmt.Sprintf("Processed: %s", req.Message),
		ProcessedNumber: req.Number * 2,
		TagCount:        int32(len(req.Tags)),
		ServerTimestamp: &timestamppb.Timestamp{
			Seconds: req.Timestamp.Seconds + 1000,
		},
	}, nil
}

// ClientStream 客户端流实现
func (s *TestService) ClientStream(stream testpb.TestService_ClientStreamServer) error {
	var chunks []string
	var totalLength int
	chunkCount := 0

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			// 客户端完成发送
			break
		}
		if err != nil {
			return err
		}

		chunks = append(chunks, req.Chunk)
		totalLength += len(req.Chunk)
		chunkCount++
	}

	// 发送响应
	response := &testpb.ClientStreamResponse{
		CombinedResult: strings.Join(chunks, " "),
		TotalChunks:    int32(chunkCount),
		TotalLength:    int32(totalLength),
	}

	return stream.SendAndClose(response)
}

// ServerStream 服务端流实现
func (s *TestService) ServerStream(req *testpb.ServerStreamRequest, stream testpb.TestService_ServerStreamServer) error {
	count := req.Count
	if count <= 0 {
		count = 5 // 默认发送5条消息
	}
	if count > 100 {
		count = 100 // 限制最大数量
	}

	for i := int32(0); i < count; i++ {
		// 如果设置了延迟，则等待
		if req.DelayMs > 0 {
			time.Sleep(time.Duration(req.DelayMs) * time.Millisecond)
		}

		response := &testpb.ServerStreamResponse{
			Message:   fmt.Sprintf("%s-%d", req.Prefix, i),
			Index:     i,
			Timestamp: timestamppb.Now(),
		}

		if err := stream.Send(response); err != nil {
			return err
		}
	}

	return nil
}

// BidirectionalStream 双向流实现
func (s *TestService) BidirectionalStream(stream testpb.TestService_BidirectionalStreamServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		var responseType testpb.ResponseType
		var echo string

		switch req.Type {
		case testpb.RequestType_REQUEST_TYPE_ECHO:
			responseType = testpb.ResponseType_RESPONSE_TYPE_SUCCESS
			echo = fmt.Sprintf("Echo: %s", req.Message)
		case testpb.RequestType_REQUEST_TYPE_TRANSFORM:
			responseType = testpb.ResponseType_RESPONSE_TYPE_PROCESSED
			echo = strings.ToUpper(req.Message)
		case testpb.RequestType_REQUEST_TYPE_VALIDATE:
			if len(req.Message) < 5 {
				responseType = testpb.ResponseType_RESPONSE_TYPE_ERROR
				echo = "Error: Message too short"
			} else {
				responseType = testpb.ResponseType_RESPONSE_TYPE_SUCCESS
				echo = "Valid message"
			}
		default:
			responseType = testpb.ResponseType_RESPONSE_TYPE_ERROR
			echo = "Unknown request type"
		}

		response := &testpb.BidirectionalResponse{
			Echo:        echo,
			RequestId:   req.Id,
			Type:        responseType,
			ProcessedAt: timestamppb.Now(),
		}

		if err := stream.Send(response); err != nil {
			return err
		}
	}
}

// EmptyCall 空调用实现
func (s *TestService) EmptyCall(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

// ErrorCall 错误测试实现
func (s *TestService) ErrorCall(ctx context.Context, req *testpb.ErrorRequest) (*testpb.ErrorResponse, error) {
	switch req.ErrorType {
	case testpb.ErrorType_ERROR_TYPE_INVALID_ARGUMENT:
		return nil, status.Error(codes.InvalidArgument, "Invalid argument: "+req.Message)
	case testpb.ErrorType_ERROR_TYPE_NOT_FOUND:
		return nil, status.Error(codes.NotFound, "Not found: "+req.Message)
	case testpb.ErrorType_ERROR_TYPE_PERMISSION_DENIED:
		return nil, status.Error(codes.PermissionDenied, "Permission denied: "+req.Message)
	case testpb.ErrorType_ERROR_TYPE_RESOURCE_EXHAUSTED:
		return nil, status.Error(codes.ResourceExhausted, "Resource exhausted: "+req.Message)
	case testpb.ErrorType_ERROR_TYPE_INTERNAL:
		return nil, status.Error(codes.Internal, "Internal error: "+req.Message)
	case testpb.ErrorType_ERROR_TYPE_UNAVAILABLE:
		return nil, status.Error(codes.Unavailable, "Service unavailable: "+req.Message)
	case testpb.ErrorType_ERROR_TYPE_DEADLINE_EXCEEDED:
		return nil, status.Error(codes.DeadlineExceeded, "Deadline exceeded: "+req.Message)
	default:
		return &testpb.ErrorResponse{
			Result: "No error",
		}, nil
	}
}

// LargeDataCall 大数据测试实现
func (s *TestService) LargeDataCall(ctx context.Context, req *testpb.LargeDataRequest) (*testpb.LargeDataResponse, error) {
	originalSize := len(req.Data)

	// 计算MD5校验和
	hash := md5.Sum(req.Data)
	checksum := fmt.Sprintf("%x", hash)

	// 处理数据（这里只是简单地复制）
	processedData := make([]byte, len(req.Data))
	copy(processedData, req.Data)

	return &testpb.LargeDataResponse{
		ProcessedData: processedData,
		OriginalSize:  int32(originalSize),
		ProcessedSize: int32(len(processedData)),
		Checksum:      checksum,
	}, nil
}

// TimeoutCall 超时测试实现
func (s *TestService) TimeoutCall(ctx context.Context, req *testpb.TimeoutRequest) (*testpb.TimeoutResponse, error) {
	delayDuration := time.Duration(req.DelaySeconds) * time.Second

	// 使用select来处理超时
	select {
	case <-time.After(delayDuration):
		return &testpb.TimeoutResponse{
			Result:      fmt.Sprintf("Completed after %d seconds: %s", req.DelaySeconds, req.Message),
			ActualDelay: req.DelaySeconds,
		}, nil
	case <-ctx.Done():
		// 上下文被取消（超时或客户端取消）
		return nil, status.Error(codes.DeadlineExceeded, "Request timeout")
	}
}

// MetadataCall 元数据测试实现
func (s *TestService) MetadataCall(ctx context.Context, req *testpb.MetadataRequest) (*testpb.MetadataResponse, error) {
	receivedMetadata := make(map[string]string)

	// 读取传入的元数据
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		for key, values := range md {
			if len(values) > 0 {
				receivedMetadata[key] = values[0] // 只取第一个值
			}
		}
	}

	// 设置响应头
	responseHeader := metadata.Pairs(
		"server-response", "metadata-call-response",
		"timestamp", time.Now().Format(time.RFC3339),
	)
	grpc.SetHeader(ctx, responseHeader)

	// 设置响应尾部
	responseTrailer := metadata.Pairs(
		"processing-time", "fast",
		"server-version", "1.0.0",
	)
	grpc.SetTrailer(ctx, responseTrailer)

	return &testpb.MetadataResponse{
		ReceivedMetadata: receivedMetadata,
		Result:           fmt.Sprintf("Processed metadata call with key: %s, value: %s", req.Key, req.Value),
	}, nil
}

// AuthCall 认证测试实现
func (s *TestService) AuthCall(ctx context.Context, req *testpb.AuthRequest) (*testpb.AuthResponse, error) {
	// 简单的认证逻辑
	validTokens := map[string]string{
		"valid-token-123": "user1",
		"admin-token-456": "admin",
		"test-token-789":  "test-user",
	}

	userInfo, exists := validTokens[req.Token]
	if !exists {
		return nil, status.Error(codes.Unauthenticated, "Invalid token")
	}

	var permissions []string
	switch userInfo {
	case "admin":
		permissions = []string{"read", "write", "admin"}
	case "user1":
		permissions = []string{"read", "write"}
	case "test-user":
		permissions = []string{"read"}
	default:
		permissions = []string{}
	}

	return &testpb.AuthResponse{
		Authenticated: true,
		UserInfo:      userInfo,
		Permissions:   permissions,
	}, nil
}
