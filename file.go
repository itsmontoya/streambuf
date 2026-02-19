package streambuf

import (
	"os"
	"sync"
)

func newFile(filepath string) (out *file, err error) {
	var f file
	if f.w, err = os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644); err != nil {
		return nil, err
	}

	if f.r, err = os.Open(filepath); err != nil {
		f.w.Close()
		return nil, err
	}

	return &f, nil
}

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
		return 0, err
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

	return f.w.Close()
}

// CloseReader marks the file backend reader as closed and closes its file handle.
func (f *file) CloseReader() (err error) {
	f.mux.Lock()
	defer f.mux.Unlock()
	if f.readerClosed {
		return ErrIsClosed
	}

	f.readerClosed = true

	return f.r.Close()
}
