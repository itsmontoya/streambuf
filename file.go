package streambuf

import (
	"fmt"
	"os"
	"sync"
)

// newFile constructs a file backend with separate read and append handles.
func newFile(filepath string) (out *file, err error) {
	var f file
	if f.w, err = os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644); err != nil {
		return nil, fmt.Errorf("open writer file: %w", err)
	}

	if f.r, err = os.Open(filepath); err != nil {
		f.w.Close()
		return nil, fmt.Errorf("open reader file: %w", err)
	}

	return &f, nil
}

// file is a backend backed by separate file handles for reading and writing.
type file struct {
	mux sync.RWMutex

	w *os.File
	r *os.File

	writerClosed bool
	readerClosed bool
}

// Write appends bytes to the backend unless it is closed.
func (f *file) Write(bs []byte) (n int, err error) {
	f.mux.RLock()
	defer f.mux.RUnlock()
	if f.writerClosed {
		return 0, ErrIsClosed
	}

	return f.w.Write(bs)
}

// ReadAt copies bytes from index into in.
// It returns ErrIsClosed when no bytes are read and the writer has been closed.
func (f *file) ReadAt(in []byte, index int64) (n int, err error) {
	f.mux.RLock()
	defer f.mux.RUnlock()
	n, err = f.r.ReadAt(in, index)
	switch {
	case n > 0:
		return n, nil
	case f.writerClosed:
		return 0, ErrIsClosed
	default:
		return 0, fmt.Errorf("read reader file at index %d: %w", index, err)
	}
}

// CloseWriter marks the file backend writer as closed and closes its file handle.
func (f *file) CloseWriter() (err error) {
	f.mux.Lock()
	defer f.mux.Unlock()
	if f.writerClosed {
		return ErrIsClosed
	}

	f.writerClosed = true

	if err = f.w.Close(); err != nil {
		return fmt.Errorf("close writer file: %w", err)
	}

	return nil
}

// CloseReader marks the file backend reader as closed and closes its file handle.
func (f *file) CloseReader() (err error) {
	f.mux.Lock()
	defer f.mux.Unlock()
	if f.readerClosed {
		return ErrIsClosed
	}

	f.readerClosed = true

	if err = f.r.Close(); err != nil {
		return fmt.Errorf("close reader file: %w", err)
	}

	return nil
}
