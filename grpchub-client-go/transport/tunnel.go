package transport

type Conn[T any] interface {
	Send(T) error
	Recv() (T, error)
}

type TunnelInterface[T any] interface {
	Send(T) error
	Recv() (T, error)
}

type tunnel[T any] struct {
	conn Conn[T]
}

// Recv implements TunnelInterface.
func (t *tunnel[T]) Recv() (m T, err error) {
	m, err = t.conn.Recv()
	err = checkIoError(err)
	return
}

// Send implements TunnelInterface.
func (t *tunnel[T]) Send(req T) error {
	return t.conn.Send(req)
}

var _ TunnelInterface[any] = (*tunnel[any])(nil)

func NewTunnel[T any](conn Conn[T]) TunnelInterface[T] {
	return &tunnel[T]{
		conn: conn,
	}
}
