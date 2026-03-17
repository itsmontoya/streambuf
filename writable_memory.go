package streambuf

import (
	"sync"
)

var _ writable = &writableMemory{}

// newWritableMemory constructs the writable memory backend used by Buffer.
func newWritableMemory(bs []byte) (out *writableMemory) {
	var m writableMemory
	if bs == nil {
		bs = make([]byte, 0, 1024)
	}

	m.bs = bs
	return &m
}

// writableMemory is a writable memory backend that stores bytes and close state.
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

// Close marks the writable memory backend as closed and releases its byte slice.
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
