package streambuf

import (
	"io"
	"sync"
)

var _ backend = &readOnlyMemory{}

// newReadOnlyMemory constructs a read-only memory backend backed by in.
func newReadOnlyMemory(in []byte) (out *readOnlyMemory) {
	var m readOnlyMemory
	m.bs = in
	return &m
}

// readOnlyMemory is a read-only backend that stores bytes and close state.
type readOnlyMemory struct {
	mux sync.RWMutex

	bs []byte

	closed bool
}

// Write always returns ErrCannotWriteToReadOnly.
func (m *readOnlyMemory) Write(bs []byte) (n int, err error) {
	return 0, ErrCannotWriteToReadOnly
}

// ReadAt copies bytes from index into in.
// It returns ErrIsClosed when no bytes are available and the backend is closed.
func (m *readOnlyMemory) ReadAt(in []byte, index int64) (n int, err error) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	switch {
	case index < int64(len(m.bs)):
		n = copy(in, m.bs[index:])
		return n, nil
	case m.closed:
		return 0, ErrIsClosed
	default:
		return 0, io.EOF
	}
}

// CloseWriter is a no-op for readOnlyMemory and always returns nil.
func (m *readOnlyMemory) CloseWriter() (err error) {
	return nil
}

// CloseReader marks readOnlyMemory as closed and releases its byte slice.
func (m *readOnlyMemory) CloseReader() (err error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	if m.closed {
		return ErrIsClosed
	}

	m.closed = true
	m.bs = nil
	return nil
}
