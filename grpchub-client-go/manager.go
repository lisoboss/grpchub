package grpchub

import (
	"io"
	"sync"

	"github.com/lisoboss/grpchub/safe"
	"github.com/lisoboss/grpchub/transport"
)

type MessageInterface[T, Inner any] interface {
	GetSid() string
	GetInner() Inner

	SetSid(sid string) MessageInterface[T, Inner]
	SetInner(inner Inner) MessageInterface[T, Inner]

	Raw() T
	FormRaw(T) MessageInterface[T, Inner]
	New() MessageInterface[T, Inner]
}

type handler[Inner any] interface {
	Processe(Inner)
	Close()
}

type streamManager[T, Inner any] struct {
	safe.SafeClose

	wrapped          MessageInterface[T, Inner]
	wrappedTransport transport.MessageInterface[Inner]
	tunnel           transport.TunnelInterface[T]

	streams            sync.Map // mpa[sid]streamTransport
	noProcesseCallback func(msg MessageInterface[T, Inner], newStream func(sid string) *transport.Stream[Inner], addStrem func(sid string, h handler[Inner]))
}

func (sm *streamManager[T, Inner]) delStream(key string) {
	sm.streams.Delete(key)
}

func (sm *streamManager[T, Inner]) addStream(key string, h handler[Inner]) {
	sm.streams.Store(key, h)
}

func (sm *streamManager[T, Inner]) newStream(sid string) *transport.Stream[Inner] {
	stream := transport.NewStream(
		sm.wrappedTransport,
		func(inner Inner) error {
			return sm.tunnel.Send(sm.wrapped.New().SetSid(sid).SetInner(inner).Raw())
		},
		func() { sm.delStream(sid) },
	)
	return stream
}

func (sm *streamManager[T, Inner]) processe(t T) {
	msg := sm.wrapped.FormRaw(t)
	if val, ok := sm.streams.Load(msg.GetSid()); ok {
		if h, ok := val.(handler[Inner]); ok {
			h.Processe(msg.GetInner())
		}
	} else {
		sm.noProcesseCallback(msg, sm.newStream, sm.addStream)
	}
}

// Loop implements StreamManagerInterface.
func (sm *streamManager[T, Inner]) Loop() error {
	defer sm.Close()

	for {
		sm.RLock()
		select {
		case <-sm.Closed():
			sm.RUnlock()
			return nil
		default:
		}
		sm.RUnlock()

		msg, err := sm.tunnel.Recv()
		if err != nil {
			if err == io.EOF {
				return nil // 正常关闭
			}
			return err
		}

		sm.processe(msg)
	}
}

type ClientStreamManagerInterface[Inner any] interface {
	Connect(sid string, method string) *transport.ClientStream[Inner]
	Loop() error
	Close()
}

type ServerStreamManagerInterface[Inner any] interface {
	Accept() <-chan *transport.ServerStream[Inner]
	Loop() error
	Close()
}

type ClientStreamManager[T, Inner any] struct {
	*streamManager[T, Inner]
}

// Connect implements ClientStreamManagerInterface.
func (c *ClientStreamManager[T, Inner]) Connect(sid string, method string) *transport.ClientStream[Inner] {
	stream := transport.NewClientStream(c.newStream(sid), method)
	c.addStream(sid, stream)
	return stream
}

var _ ClientStreamManagerInterface[any] = (*ClientStreamManager[any, any])(nil)

type ServerStreamManager[T, Inner any] struct {
	*streamManager[T, Inner]

	tChan chan *transport.ServerStream[Inner]
}

// Accept implements ServerStreamManagerInterface.
func (sm *ServerStreamManager[T, Inner]) Accept() <-chan *transport.ServerStream[Inner] {
	return sm.tChan
}

var _ ServerStreamManagerInterface[any] = (*ServerStreamManager[any, any])(nil)

func newStreamManager[T, Inner any](
	wrapped MessageInterface[T, Inner],
	wrappedTransport transport.MessageInterface[Inner],
	tunnel transport.TunnelInterface[T],
	noProcesseCallback func(msg MessageInterface[T, Inner], newStream func(sid string) *transport.Stream[Inner], addStrem func(sid string, h handler[Inner])),
	callbacks ...func(),
) *streamManager[T, Inner] {
	safeClose := safe.NewSafeClose()
	sm := &streamManager[T, Inner]{
		SafeClose: safeClose,

		wrapped:          wrapped,
		wrappedTransport: wrappedTransport,
		tunnel:           tunnel,

		noProcesseCallback: noProcesseCallback,
	}
	safeClose.AddCloseCallbaks(append([]func(){func() {
		// 关闭所有tunnel
		sm.streams.Range(func(key, value any) bool {
			if t, ok := value.(handler[Inner]); ok {
				t.Close()
			}
			return true
		})
	}}, callbacks...)...)
	return sm
}

func NewClientStreamManager[T, Inner any](
	wrapped MessageInterface[T, Inner],
	wrappedTransport transport.MessageInterface[Inner],
	tunnel transport.TunnelInterface[T],
) ClientStreamManagerInterface[Inner] {
	return &ClientStreamManager[T, Inner]{
		streamManager: newStreamManager(
			wrapped,
			wrappedTransport,
			tunnel,
			func(
				msg MessageInterface[T, Inner],
				newStream func(sid string) *transport.Stream[Inner],
				addStrem func(sid string, h handler[Inner]),
			) {
			},
		),
	}
}

func NewServerStreamManager[T, Inner any](
	wrapped MessageInterface[T, Inner],
	wrappedTransport transport.MessageInterface[Inner],
	tunnel transport.TunnelInterface[T],
) ServerStreamManagerInterface[Inner] {
	tChan := make(chan *transport.ServerStream[Inner], 16)
	noProcesseCallback := func(msg MessageInterface[T, Inner], newStream func(string) *transport.Stream[Inner], addStrem func(sid string, h handler[Inner])) {
		sid := msg.GetSid()
		stream := transport.NewServerStream(newStream(sid))
		tChan <- stream
		addStrem(sid, stream)
		stream.Processe(msg.GetInner())
	}

	return &ServerStreamManager[T, Inner]{
		streamManager: newStreamManager(
			wrapped,
			wrappedTransport,
			tunnel,
			noProcesseCallback,
			func() { close(tChan) },
		),
		tChan: tChan,
	}
}
