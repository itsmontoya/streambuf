package streambuf

import (
	"io"
	"sync"
)

var _ readable = &readableMemory{}

// newReadableMemory constructs the readable memory backend used by Buffer and Stream.
func newReadableMemory(in *memory) (out *readableMemory) {
	var m readableMemory
	m.m = in
	return &m
}

// readableMemory is a readable memory backend that shares an underlying byte slice.
type readableMemory struct {
	mux sync.RWMutex

	m *memory

	closed bool
}

// ReadAt copies bytes from index into in.
// It returns ErrIsClosed when no bytes are available and the readable memory is closed.
func (m *readableMemory) ReadAt(in []byte, index int64) (n int, err error) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	m.m.read(func(bs []byte) {
		switch {
		case index < int64(len(bs)):
			n = copy(in, bs[index:])
		case m.closed:
			err = ErrIsClosed
		default:
			err = io.EOF
		}
	})

	return n, err
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
