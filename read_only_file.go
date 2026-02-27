package streambuf

import (
	"os"
	"sync"
)

// newReadOnlyFile constructs a readOnlyFile backend for an existing file path.
func newReadOnlyFile(filepath string) (out *readOnlyFile, err error) {
	var f readOnlyFile
	if f.r, err = os.Open(filepath); err != nil {
		return nil, err
	}

	return &f, nil
}

// readOnlyFile is a backend that supports reads only from a file handle.
type readOnlyFile struct {
	mux sync.RWMutex

	r *os.File

	closed bool
}

// Write returns ErrCannotWriteToReadOnly because readOnlyFile does not support writes.
func (f *readOnlyFile) Write(bs []byte) (n int, err error) {
	return 0, ErrCannotWriteToReadOnly
}

// ReadAt copies bytes from index into in.
// It returns ErrIsClosed when the backend has been closed.
func (f *readOnlyFile) ReadAt(in []byte, index int64) (n int, err error) {
	f.mux.RLock()
	defer f.mux.RUnlock()
	n, err = f.r.ReadAt(in, index)
	switch {
	case n > 0:
		return n, nil
	case f.closed:
		return 0, ErrIsClosed
	default:
		return 0, err
	}
}

// CloseWriter is a no-op for readOnlyFile and always returns nil.
func (f *readOnlyFile) CloseWriter() (err error) {
	return nil
}

// CloseReader marks readOnlyFile as closed and closes its file handle.
func (f *readOnlyFile) CloseReader() (err error) {
	f.mux.Lock()
	defer f.mux.Unlock()
	if f.closed {
		return ErrIsClosed
	}

	f.closed = true
	return f.r.Close()
}
