package streambuf

import (
	"sync"
)

// newMemory constructs the shared in-memory storage used by Buffer and Stream.
func newMemory(bs []byte) (out *memory) {
	var m memory
	m.bs = bs
	return &m
}

// memory coordinates concurrent access to a shared byte slice.
type memory struct {
	mux sync.RWMutex

	bs []byte
}

// write applies fn while holding the write lock and stores the returned slice.
func (m *memory) write(fn func(in []byte) (out []byte)) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.bs = fn(m.bs)
}

// read invokes fn while holding the read lock.
func (m *memory) read(fn func(in []byte)) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	fn(m.bs)
}
