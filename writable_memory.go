package streambuf

import (
	"sync"
)

var _ writable = &writableMemory{}

// newWritableMemory constructs the in-writableMemory backend used by Buffer.
func newWritableMemory(bs []byte) (out *writableMemory) {
	var m writableMemory
	if bs == nil {
		bs = make([]byte, 0, 1024)
	}

	m.bs = bs
	return &m
}

// writableMemory is the backend that stores bytes and close state.
type writableMemory struct {
	mux sync.RWMutex

	bs []byte

	closed bool
}

// Write appends bytes to the backend unless it is closed.
func (m *writableMemory) Write(bs []byte) (n int, err error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	if m.closed {
		return 0, ErrIsClosed
	}

	m.bs = append(m.bs, bs...)
	return len(bs), nil
}

// CloseReader marks the writableMemory backend reader as closed and releases writableMemory.
func (m *writableMemory) Close() (err error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	if m.closed {
		return ErrIsClosed
	}

	m.closed = true
	m.bs = nil
	return nil
}
