package streambuf

import (
	"errors"
	"os"
	"sync"
)

func newFile(filepath string) (out *file, err error) {
	var f file
	if f.w, err = os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644); err != nil {
		return
	}

	if f.r, err = os.Open(filepath); err != nil {
		return
	}

	return &f, nil
}

type file struct {
	mux sync.RWMutex

	w *os.File
	r *os.File

	closed bool
}

// Write appends bytes to the backend unless it is closed.
func (f *file) Write(bs []byte) (n int, err error) {
	f.mux.RLock()
	defer f.mux.RUnlock()
	if f.closed {
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
	case f.closed:
		return 0, ErrIsClosed
	default:
		return 0, err
	}
}

// Close marks the backend as closed.
func (f *file) Close() (err error) {
	f.mux.Lock()
	defer f.mux.Unlock()
	if f.closed {
		return ErrIsClosed
	}

	f.closed = true

	var errs []error
	if err = f.w.Close(); err != nil {
		errs = append(errs, err)
	}

	//if err = f.r.Close(); err != nil {
	//	errs = append(errs, err)
	//}

	return errors.Join(errs...)
}
