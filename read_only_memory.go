package streambuf

import (
	"io"
	"sync"
)

var _ backend = &readOnlyMemory{}

func newReadOnlyMemory(in []byte) (out *readOnlyMemory) {
	var m readOnlyMemory
	m.bs = in
	return &m
}

// readOnlyMemory is the backend that stores bytes and close state.
type readOnlyMemory struct {
	mux sync.RWMutex

	bs []byte

	closed bool
}

// Write appends bytes to the backend unless it is closed.
func (m *readOnlyMemory) Write(bs []byte) (n int, err error) {
	return 0, ErrCannotWriteToReadOnly
}

// ReadAt copies bytes from index into in.
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

// CloseWriter marks the readOnlyMemory backend writer as closed.
func (m *readOnlyMemory) CloseWriter() (err error) {
	return nil
}

// CloseReader marks the readOnlyMemory backend reader as closed and releases readOnlyMemory.
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
