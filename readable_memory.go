package streambuf

import (
	"io"
	"sync"
)

var _ readable = &readableMemory{}

// newReadableMemory constructs the readable memory backend used by Buffer and Stream.
func newReadableMemory(bs *[]byte) (out *readableMemory) {
	var m readableMemory
	m.bs = bs
	return &m
}

// readableMemory is a readable memory backend that shares an underlying byte slice.
type readableMemory struct {
	mux sync.RWMutex

	bs *[]byte

	closed bool
}

// ReadAt copies bytes from index into in.
// It returns ErrIsClosed when no bytes are available and the readable memory is closed.
func (m *readableMemory) ReadAt(in []byte, index int64) (n int, err error) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	switch {
	case index < int64(len(*m.bs)):
		n = copy(in, (*m.bs)[index:])
		return n, nil
	case m.closed:
		return 0, ErrIsClosed
	default:
		return 0, io.EOF
	}
}

// Close marks the readable memory backend as closed.
func (m *readableMemory) Close() (err error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	if m.closed {
		return ErrIsClosed
	}

	m.closed = true
	return nil
}
