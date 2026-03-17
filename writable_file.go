package streambuf

import (
	"fmt"
	"os"
	"sync"
)

var _ writable = &writableFile{}

// newWritableFile constructs a writable file backend for append-only writes.
func newWritableFile(filepath string) (out *writableFile, err error) {
	var f writableFile
	if f.f, err = os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644); err != nil {
		return nil, fmt.Errorf("open writer file: %w", err)
	}

	return &f, nil
}

// writableFile is a write-only backend backed by a file handle.
type writableFile struct {
	mux sync.RWMutex

	f *os.File

	closed bool
}

// Write appends bytes to the backend unless it is closed.
func (f *writableFile) Write(bs []byte) (n int, err error) {
	f.mux.RLock()
	defer f.mux.RUnlock()
	if f.closed {
		return 0, ErrIsClosed
	}

	return f.f.Write(bs)
}

// Close marks the writable file as closed and closes its file handle.
func (f *writableFile) Close() (err error) {
	f.mux.Lock()
	defer f.mux.Unlock()
	if f.closed {
		return ErrIsClosed
	}

	f.closed = true

	if err = f.f.Close(); err != nil {
		return fmt.Errorf("close writer file: %w", err)
	}

	return nil
}
