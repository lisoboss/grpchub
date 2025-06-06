package grpchub

import (
	"context"
	"io"
	"sync"

	channelv1 "github.com/lisoboss/grpchub/gen/channel/v1"
)

type handler interface {
	handle(*channelv1.MessagePackage)
}

type StreamManagerInterface interface {
	NewStreamTransport(sid string) StreamTransportInterface
	Loop() error
	Close() error
}

type streamManager struct {
	ctx       context.Context
	cancel    context.CancelFunc
	closed    chan struct{}
	closeOnce sync.Once

	conn    TunnelConnInterface
	serv    chan StreamTransportInterface
	streams sync.Map // mpa[sid]streamTransport
}

func (sm *streamManager) delStream(key string) {
	sm.streams.Delete(key)
}

// NewStreamTransport implements StreamManagerInterface.
func (sm *streamManager) NewStreamTransport(sid string) StreamTransportInterface {
	conn := &streamTransport{
		sid: sid,
		send: func(pkg *channelv1.MessagePackage) error {
			return sm.conn.Send(&channelv1.ChannelMessage{
				Sid: sid,
				Pkg: pkg,
			})
		},
		recv:   make(chan *channelv1.MessagePackage, 16),
		closed: make(chan struct{}),
		sm:     sm,
	}
	sm.streams.Store(sid, conn)
	return conn
}

func (sm *streamManager) handleSer(msg *channelv1.ChannelMessage) {
	if !sm.handleCli(msg) {
		conn := sm.NewStreamTransport(msg.Sid)
		sm.serv <- conn

		// handle
		var val any = conn
		if h, ok := val.(handler); ok {
			h.handle(msg.Pkg)
		}
	}
}
func (sm *streamManager) handleCli(msg *channelv1.ChannelMessage) bool {
	if val, ok := sm.streams.Load(msg.Sid); ok {
		if h, ok := val.(handler); ok {
			h.handle(msg.Pkg)
			return true
		}
	}

	return false
}

// Loop implements StreamManagerInterface.
func (sm *streamManager) Loop() error {
	defer sm.Close()

	for {
		select {
		case <-sm.ctx.Done():
			return sm.ctx.Err()
		case <-sm.closed:
			return nil
		default:
		}

		msg, err := sm.conn.Recv()
		if err != nil {
			if err == io.EOF {
				return nil // 正常关闭
			}
			return err
		}

		if sm.serv != nil {
			sm.handleSer(msg)
		} else {
			sm.handleCli(msg)
		}
	}
}

// Close implements StreamManagerInterface.
func (sm *streamManager) Close() error {
	sm.closeOnce.Do(func() {
		if sm.cancel != nil {
			sm.cancel()
		}
		close(sm.closed)

		// 关闭所有tunnel
		sm.streams.Range(func(key, value interface{}) bool {
			if t, ok := value.(StreamTransportInterface); ok {
				t.Close()
			}
			return true
		})
	})
	return nil
}

var _ StreamManagerInterface = (*streamManager)(nil)

type streamTransport struct {
	sid       string
	send      func(*channelv1.MessagePackage) error
	recv      chan *channelv1.MessagePackage
	closed    chan struct{}
	closeOnce sync.Once
	sm        *streamManager
}

// Recv implements StreamTransportInterface.
func (t *streamTransport) Recv() (*channelv1.MessagePackage, error) {
	// logger.Infof("Recv t.recv: %d", len(t.recv))
	select {
	case pkg := <-t.recv:
		// logger.Infof("Recv t.recv return: %#v", pkg)
		return pkg, nil
	case <-t.closed:
		return nil, io.EOF
	}
}

// Send implements StreamTransportInterface.
func (t *streamTransport) Send(pkg *channelv1.MessagePackage) error {
	select {
	case <-t.closed:
		return io.EOF
	default:
	}

	return t.send(pkg)
}

// Close implements StreamTransportInterface.
func (t *streamTransport) Close() error {
	t.closeOnce.Do(func() {
		t.sm.delStream(t.sid)
		close(t.closed)
		close(t.recv)
	})
	return nil
}

var _ StreamTransportInterface = (*streamTransport)(nil)

func (t *streamTransport) handle(m *channelv1.MessagePackage) {
	// logger.Infof("handle t.recv: %#v", m)

	select {
	case t.recv <- m:
		// logger.Infof("handle t.recv sended: %#v", m)
	case <-t.closed:
		// tunnel已关闭，丢弃消息
		logger.Errorf("handle close: %#v", m)
	default:
		// TODO
		// 通道满了，可以考虑丢弃或者记录日志
		// 这里选择非阻塞发送，如果满了就丢弃
		logger.Errorf("handle over: %#v", m)
	}
}

var _ handler = (*streamTransport)(nil)

func NewClientStreamManager(ctx context.Context, conn TunnelConnInterface) StreamManagerInterface {
	ctx, cancel := context.WithCancel(ctx)

	sm := &streamManager{
		ctx:    ctx,
		cancel: cancel,
		closed: make(chan struct{}),
		conn:   conn,
	}

	return sm
}

func NewServerStreamManager(ctx context.Context, conn TunnelConnInterface) (StreamManagerInterface, chan StreamTransportInterface) {
	ctx, cancel := context.WithCancel(ctx)

	sm := &streamManager{
		ctx:    ctx,
		cancel: cancel,
		closed: make(chan struct{}),

		conn: conn,
		serv: make(chan StreamTransportInterface, 16),
	}

	return sm, sm.serv
}
