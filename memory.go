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

	writerClosed bool
	readerClosed bool
}

// Write appends bytes to the backend unless it is closed.
func (m *memory) Write(bs []byte) (n int, err error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	if m.writerClosed {
		return 0, ErrIsClosed
	}

	m.bs = append(m.bs, bs...)
	return len(bs), nil
}

// ReadAt copies bytes from index into in.
func (m *memory) ReadAt(in []byte, index int64) (n int, err error) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	switch {
	case index < int64(len(m.bs)):
		n = copy(in, m.bs[index:])
		return n, nil
	case m.writerClosed:
		return 0, ErrIsClosed
	default:
		return 0, io.EOF
	}
}

// CloseWriter marks the memory backend writer as closed.
func (m *memory) CloseWriter() (err error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	if m.writerClosed {
		return ErrIsClosed
	}

	m.writerClosed = true
	return nil
}

// CloseReader marks the memory backend reader as closed and releases memory.
func (m *memory) CloseReader() (err error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	if m.readerClosed {
		return ErrIsClosed
	}

	m.readerClosed = true
	m.bs = nil
	return nil
}
