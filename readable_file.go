package streambuf

import (
	"fmt"
	"os"
	"sync"
)

var _ readable = &readableFile{}

// newReadableFile constructs a readable file backend for an existing file path.
func newReadableFile(filepath string) (out *readableFile, err error) {
	var f readableFile
	if f.f, err = os.Open(filepath); err != nil {
		return nil, fmt.Errorf("open reader file: %w", err)
	}

	return &f, nil
}

// readableFile is a read-only backend backed by a file handle.
type readableFile struct {
	mux sync.RWMutex

	f *os.File

	closed bool
}

// ReadAt copies bytes from index into in.
// It returns ErrIsClosed when no bytes are read and the readable file is closed.
func (f *readableFile) ReadAt(in []byte, index int64) (n int, err error) {
	f.mux.RLock()
	defer f.mux.RUnlock()
	n, err = f.f.ReadAt(in, index)
	switch {
	case n > 0:
		return n, nil
	case f.closed:
		return 0, ErrIsClosed
	default:
		return 0, fmt.Errorf("read reader file at index %d: %w", index, err)
	}
}

// Close marks the readable file as closed and closes its file handle.
func (f *readableFile) Close() (err error) {
	f.mux.Lock()
	defer f.mux.Unlock()
	if f.closed {
		return ErrIsClosed
	}

	f.closed = true

	if err = f.f.Close(); err != nil {
		return fmt.Errorf("close reader file: %w", err)
	}

	return nil
}
