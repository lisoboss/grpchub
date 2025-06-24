package transport

import (
	"errors"
	"fmt"
	"io"

	"github.com/lisoboss/grpchub/header"
	"github.com/lisoboss/grpchub/safe"
)

const (
	M_TYPE_HELLO int32 = iota + 1
	M_TYPE_HEADER
	M_TYPE_PAYLOAD
	M_TYPE_ERROR
	M_TYPE_CLOSE
)

var M_Type_name = map[int32]string{
	0: "M_TYPE_UNKNOWN",
	1: "M_TYPE_HELLO",
	2: "M_TYPE_HEADER",
	3: "M_TYPE_PAYLOAD",
	4: "M_TYPE_ERROR",
	5: "M_TYPE_CLOSE",
}

var ErrUnexpectedType = errors.New("unexpected type")

type MessageInterface[Inner any] interface {
	GetType() int32
	GetMethod() string
	GetHeader() header.Header
	GetPayload() (any, error)
	GetError() (appErr error, err error)

	SetType(t int32) MessageInterface[Inner]
	SetMethod(method string) MessageInterface[Inner]
	SetHeader(header header.Header) MessageInterface[Inner]
	SetPayload(payload any) MessageInterface[Inner]
	SetError(appErr error) MessageInterface[Inner]

	Raw() Inner
	FormRaw(Inner) MessageInterface[Inner]
	New() MessageInterface[Inner]
}

type Stream[Inner any] struct {
	safe.SafeClose

	//

	wrapped MessageInterface[Inner]

	method             string
	header             header.Header     // reply head data
	cacheHeader        header.HashHeader // aoto send head data
	cacheHeaderHash    uint64
	cacheHeaderChanged bool

	reqChan  chan any
	replyTo  func(Inner) error
	sayHello func()
}

func (stream *Stream[Inner]) safeInsertChan(req any) {
	stream.RLock()
	defer stream.RUnlock()

	defer func() {
		if r := recover(); r != nil {
			// TODO Ignore the error "send on closed channel"
			// How to fix it? fuck!!!
			// logger.Error(r)
		}
	}()

	select {
	case <-stream.Closed():
		return
	case stream.reqChan <- req:
		return
	}
}

func (stream *Stream[Inner]) Processe(inner Inner) {
	res, err := stream.processe(inner)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			logger.Errorf("processe inner err: %s", err)
		}
		return
	}
	if res != nil {
		stream.safeInsertChan(res)
	}
}

func (stream *Stream[Inner]) processe(inner Inner) (any, error) {
	// logger.Infof("Recv msg %s %s", M_Type_name[msg.GetType()], msg.GetMethod())
	msg := stream.wrapped.FormRaw(inner)

	switch msg.GetType() {
	case M_TYPE_ERROR:
		return msg.GetError()
	case M_TYPE_HEADER:
		stream.header.Extend(msg.GetHeader())
		return nil, nil
	case M_TYPE_PAYLOAD:
		return msg.GetPayload()
	case M_TYPE_CLOSE:
		return io.EOF, nil
	case M_TYPE_HELLO:
		stream.method = msg.GetMethod()
		stream.sayHello()
		return nil, nil
	default:
		return nil, ErrUnexpectedType
	}
}

func (stream *Stream[Inner]) GetMethod() string {
	return stream.method
}

func (stream *Stream[Inner]) GetHeader() header.Header {
	return stream.header
}

func (stream *Stream[Inner]) GetCacheHeader() header.Header {
	return header.Header(stream.cacheHeader)
}

func (stream *Stream[Inner]) SetCacheHeader(h header.Header) {
	if h != nil {
		header.Header(stream.cacheHeader).Extend(h)
		stream.checkHash()
	}
}

func (stream *Stream[Inner]) checkHash() bool {
	hash := stream.cacheHeader.Hash()
	if stream.cacheHeaderHash != hash {
		stream.cacheHeaderHash = hash
		stream.cacheHeaderChanged = true
	}
	return stream.cacheHeaderChanged
}

func (stream *Stream[Inner]) isCacheHeaderChanged() bool {
	if stream.cacheHeaderChanged {
		return true
	}
	return stream.checkHash()
}

func (stream *Stream[Inner]) autoSendHeader(t int32) error {
	msg := stream.wrapped.New()

	// auto send header
	switch t {
	case M_TYPE_PAYLOAD, M_TYPE_ERROR, M_TYPE_CLOSE:
		if stream.isCacheHeaderChanged() {
			req := msg.SetType(M_TYPE_HEADER).SetHeader(stream.GetCacheHeader()).Raw()
			if err := stream.replyTo(req); err != nil {
				return err
			}
			stream.cacheHeaderChanged = false
		}
	default:
	}

	return nil
}

func (stream *Stream[Inner]) send(t int32, args any) error {
	// logger.Infof("Send t: %s, args: %#v", M_Type_name[t], args)

	if err := stream.autoSendHeader(t); err != nil {
		return err
	}

	msg := stream.wrapped.New().SetType(t)
	switch t {
	case M_TYPE_CLOSE:
	case M_TYPE_HELLO:
		msg.SetMethod(stream.method)
	case M_TYPE_HEADER:
		h, ok := args.(header.Header)
		if !ok {
			return fmt.Errorf("invalid args: expected header.Header, got %T", args)
		}
		stream.SetCacheHeader(h)
		msg.SetHeader(stream.GetCacheHeader())
		stream.cacheHeaderChanged = false
	case M_TYPE_PAYLOAD:
		msg.SetPayload(args)
	case M_TYPE_ERROR:
		appErr, ok := args.(error)
		if !ok {
			return fmt.Errorf("invalid args: expected error, got %T", args)
		}
		msg.SetError(appErr)
	default:
		return ErrUnexpectedType
	}

	return stream.replyTo(msg.Raw())
}

func (stream *Stream[Inner]) SendHello() error {
	return stream.send(M_TYPE_HELLO, nil)
}

func (stream *Stream[Inner]) SendClose() error {
	return stream.send(M_TYPE_CLOSE, nil)
}

func (stream *Stream[Inner]) SendAppErr(appErr error) error {
	return stream.send(M_TYPE_ERROR, appErr)
}

func (stream *Stream[Inner]) SendPayload(payload any) error {
	return stream.send(M_TYPE_PAYLOAD, payload)
}

func (stream *Stream[Inner]) SendHeader(header map[string][]string) error {
	return stream.send(M_TYPE_HEADER, header)
}

func (stream *Stream[Inner]) Recv() (any, error) {
	rel, ok := <-stream.reqChan
	if !ok {
		return nil, io.EOF
	}
	if err, ok := rel.(error); ok {
		return nil, err
	}
	return rel, nil
}

func NewStream[Inner any](wrapped MessageInterface[Inner], replyTo func(Inner) error, callbacks ...func()) *Stream[Inner] {
	reqChan := make(chan any, 16)
	return &Stream[Inner]{
		SafeClose: safe.NewSafeClose(append([]func(){func() { close(reqChan) }}, callbacks...)...),

		wrapped: wrapped,

		header:      header.Header{},
		cacheHeader: header.HashHeader{},

		reqChan: reqChan,
		replyTo: replyTo,
	}
}
