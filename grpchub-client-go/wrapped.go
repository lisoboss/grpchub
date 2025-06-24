package grpchub

import (
	channelv1 "github.com/lisoboss/grpchub/gen/channel/v1"
	"github.com/lisoboss/grpchub/header"
	"github.com/lisoboss/grpchub/transport"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type WrappedMessagePackage struct {
	data *channelv1.MessagePackage
}

// FormRaw implements transport.MessageInterface.
func (w *WrappedMessagePackage) FormRaw(data *channelv1.MessagePackage) transport.MessageInterface[*channelv1.MessagePackage] {
	w.data = data
	return w
}

// New implements transport.MessageInterface.
func (w *WrappedMessagePackage) New() transport.MessageInterface[*channelv1.MessagePackage] {
	w.data = &channelv1.MessagePackage{}
	return w
}

// Raw implements transport.MessageInterface.
func (w *WrappedMessagePackage) Raw() *channelv1.MessagePackage {
	return w.data
}

// GetError implements transport.MessageInterface.
func (w *WrappedMessagePackage) GetError() (appErr error, err error) {
	var s spb.Status
	if err = anypb.UnmarshalTo(w.data.Payload, &s, proto.UnmarshalOptions{}); err != nil {
		return nil, err
	}
	return status.ErrorProto(&s), nil
}

// GetHeader implements transport.MessageInterface.
func (w *WrappedMessagePackage) GetHeader() header.Header {
	return header.Header(parseMetadataEntries(w.data.Md))
}

// GetMethod implements transport.MessageInterface.
func (w *WrappedMessagePackage) GetMethod() string {
	return w.data.Method
}

// GetPayload implements transport.MessageInterface.
func (w *WrappedMessagePackage) GetPayload() (any, error) {
	return anypb.UnmarshalNew(w.data.Payload, proto.UnmarshalOptions{})
}

// GetType implements transport.MessageInterface.
func (w *WrappedMessagePackage) GetType() int32 {
	return int32(w.data.Type)
}

// SetError implements transport.MessageInterface.
func (w *WrappedMessagePackage) SetError(appErr error) transport.MessageInterface[*channelv1.MessagePackage] {
	var (
		st *status.Status
		ok bool
	)

	st, ok = status.FromError(appErr)
	if !ok {
		st = status.New(codes.Unknown, appErr.Error())
	}

	return w.SetPayload(st.Proto())
}

// SetHeader implements transport.MessageInterface.
func (w *WrappedMessagePackage) SetHeader(header header.Header) transport.MessageInterface[*channelv1.MessagePackage] {
	w.data.Md = buildMetadataEntries(metadata.MD(header))
	return w
}

// SetMethod implements transport.MessageInterface.
func (w *WrappedMessagePackage) SetMethod(method string) transport.MessageInterface[*channelv1.MessagePackage] {
	w.data.Method = method
	return w
}

// SetPayload implements transport.MessageInterface.
func (w *WrappedMessagePackage) SetPayload(args any) transport.MessageInterface[*channelv1.MessagePackage] {
	payload, err := anypb.New(args.(proto.Message))
	if err != nil {
		logger.Fatalf("anypb.New err: %s", err)
	}
	w.data.Payload = payload
	return w
}

// SetType implements transport.MessageInterface.
func (w *WrappedMessagePackage) SetType(t int32) transport.MessageInterface[*channelv1.MessagePackage] {
	w.data.Type = channelv1.PackageType(t)
	return w
}

var _ transport.MessageInterface[*channelv1.MessagePackage] = (*WrappedMessagePackage)(nil)

type WrappedMessage struct {
	data *channelv1.ChannelMessage
}

// FormRaw implements MessageInterface.
func (w *WrappedMessage) FormRaw(data *channelv1.ChannelMessage) MessageInterface[*channelv1.ChannelMessage, *channelv1.MessagePackage] {
	w.data = data
	return w
}

// New implements MessageInterface.
func (w *WrappedMessage) New() MessageInterface[*channelv1.ChannelMessage, *channelv1.MessagePackage] {
	w.data = &channelv1.ChannelMessage{}
	return w
}

// Raw implements MessageInterface.
func (w *WrappedMessage) Raw() *channelv1.ChannelMessage {
	return w.data
}

// GetInner implements MessageInterface.
func (w *WrappedMessage) GetInner() *channelv1.MessagePackage {
	return w.data.Pkg
}

// GetSid implements MessageInterface.
func (w *WrappedMessage) GetSid() string {
	return w.data.Sid
}

// SetInner implements MessageInterface.
func (w *WrappedMessage) SetInner(inner *channelv1.MessagePackage) MessageInterface[*channelv1.ChannelMessage, *channelv1.MessagePackage] {
	w.data.Pkg = inner
	return w
}

// SetSid implements MessageInterface.
func (w *WrappedMessage) SetSid(sid string) MessageInterface[*channelv1.ChannelMessage, *channelv1.MessagePackage] {
	w.data.Sid = sid
	return w
}

var _ MessageInterface[*channelv1.ChannelMessage, *channelv1.MessagePackage] = (*WrappedMessage)(nil)
