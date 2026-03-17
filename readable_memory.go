package streambuf

import (
	"io"
	"sync"
)

var _ readable = &readableMemory{}

// newMemory constructs the in-readableMemory backend used by Buffer.
func newReadableMemory(bs *[]byte) (out *readableMemory) {
	var m readableMemory
	m.bs = bs
	return &m
}

// readableMemory is the backend that stores bytes and close state.
type readableMemory struct {
	mux sync.RWMutex

	bs *[]byte

	closed bool
}

// ReadAt copies bytes from index into in.
// It returns ErrIsClosed when no bytes are available and the writer is closed.
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

// CloseReader marks the readableMemory backend reader as closed and releases readableMemory.
func (m *readableMemory) Close() (err error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	if m.closed {
		return ErrIsClosed
	}

	m.closed = true
	return nil
}
