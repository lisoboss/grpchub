package transport

import (
	"github.com/lisoboss/grpchub/header"
)

type ClientStream[Inner any] struct {
	*Stream[Inner]
}

func (stream *ClientStream[Inner]) Handshake() error {
	return stream.SendHello()
}

func (stream *ClientStream[Inner]) RequestHeader() header.Header {
	return stream.GetCacheHeader()
}

func (stream *ClientStream[Inner]) ReplyHeader() header.Header {
	return stream.GetHeader()
}

func NewClientStream[Inner any](stream *Stream[Inner], method string) *ClientStream[Inner] {
	stream.method = method
	stream.sayHello = func() {}

	return &ClientStream[Inner]{
		Stream: stream,
	}
}

type ServerStream[Inner any] struct {
	*Stream[Inner]

	okChan chan struct{}
}

func (stream *ServerStream[Inner]) WaitHandshake() {
	<-stream.okChan
}

func (stream *ServerStream[Inner]) RequestHeader() header.Header {
	return stream.GetHeader()
}

func (stream *ServerStream[Inner]) ReplyHeader() header.Header {
	return stream.GetCacheHeader()
}

func (stream *ServerStream[Inner]) WriteError(err error) error {
	return stream.SendAppErr(err)
}

func NewServerStream[Inner any](stream *Stream[Inner]) *ServerStream[Inner] {
	okChan := make(chan struct{})
	stream.sayHello = func() {
		close(okChan)
	}

	return &ServerStream[Inner]{
		Stream: stream,
		okChan: okChan,
	}
}
