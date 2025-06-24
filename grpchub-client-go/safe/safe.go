package safe

import "sync"

type SafeClose interface {
	RUnlock()
	RLock()
	Closed() <-chan struct{}
	AddCloseCallbaks(callbacks ...func())
	Close()
}

type safeClose struct {
	sync.RWMutex
	closed         chan struct{}
	closeOnce      sync.Once
	closeCallbacks []func()
}

func (sc *safeClose) close() {
	close(sc.closed)

	sc.Lock()
	defer sc.Unlock()
	for _, f := range sc.closeCallbacks {
		f()
	}
}

func (sc *safeClose) AddCloseCallbaks(callbacks ...func()) {
	sc.closeCallbacks = append(sc.closeCallbacks, callbacks...)
}

func (sc *safeClose) Close() {
	sc.closeOnce.Do(sc.close)
}

func (sc *safeClose) Closed() <-chan struct{} {
	return sc.closed
}

func NewSafeClose(callbacks ...func()) SafeClose {
	return &safeClose{
		closed:         make(chan struct{}),
		closeCallbacks: callbacks,
	}
}
