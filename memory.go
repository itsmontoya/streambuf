package streambuf

import (
	"io"
	"sync"
)

var _ backend = &memory{}

// newMemory constructs the in-memory backend used by Buffer.
func newMemory() (out *memory) {
	var m memory
	m.bs = make([]byte, 0, 1024)
	return &m
}

// memory is the backend that stores bytes and close state.
type memory struct {
	mux sync.RWMutex

	bs []byte

	closed bool
}

// Write appends bytes to the backend unless it is closed.
func (m *memory) Write(bs []byte) (n int, err error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	if m.closed {
		return 0, ErrIsClosed
	}

	m.bs = append(m.bs, bs...)
	return len(bs), nil
}

// ReadAt copies bytes from index into in.
func (m *memory) ReadAt(in []byte, index int) (n int, err error) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	switch {
	case index < len(m.bs):
		n = copy(in, m.bs[index:])
		return n, nil
	case m.closed:
		return 0, ErrIsClosed
	default:
		return 0, io.EOF
	}
}

// Close marks the backend as closed.
func (m *memory) Close() (err error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	if m.closed {
		return ErrIsClosed
	}

	m.closed = true
	return nil
}
