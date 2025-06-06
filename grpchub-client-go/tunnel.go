package grpchub

import (
	"io"

	channelv1 "github.com/lisoboss/grpchub/gen/channel/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TunnelConnInterface interface {
	Send(*channelv1.ChannelMessage) error
	Recv() (*channelv1.ChannelMessage, error)
}

type tunnelConn struct {
	t channelv1.ChannelService_ChannelClient
}

// Recv implements TunnelConnInterface.
func (tc *tunnelConn) Recv() (m *channelv1.ChannelMessage, err error) {
	m, err = tc.t.Recv()
	err = checkIoError(err)
	return
}

// Send implements TunnelConnInterface.
func (tc *tunnelConn) Send(m *channelv1.ChannelMessage) error {
	return tc.t.Send(m)
}

var _ TunnelConnInterface = (*tunnelConn)(nil)

func NewTunnelConn(t channelv1.ChannelService_ChannelClient) TunnelConnInterface {
	tConn := &tunnelConn{
		t: t,
	}
	return tConn
}

func checkIoError(err error) error {
	st, ok := status.FromError(err)
	if ok && st.Code() == codes.Canceled {
		// logger.Warningf("tunnel recv canceled: %v", st.Message())
		return io.EOF
	}
	return err
}
