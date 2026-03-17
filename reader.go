package streambuf

import (
	"errors"
	"io"
)

var _ io.ReadSeekCloser = &reader{}

// newReader constructs a reader bound to a shared stream.
func newReader(s *stream) (out *reader) {
	var r reader
	r.s = s
	r.closer = newWaiter()
	return &r
}

// reader streams bytes while tracking its own read position.
type reader struct {
	s *stream

	index int64

	closer *waiter
}

// Read copies available bytes into in and blocks until data is written,
// the stream is closed, or the reader is closed.
// A zero-length read returns (0, nil) immediately.
// When no bytes are read, it returns ErrIsClosed after either the stream
// closes or the reader closes.
func (r *reader) Read(in []byte) (n int, err error) {
	if len(in) == 0 {
		return 0, nil
	}

	for {
		n, err = r.s.r.ReadAt(in, r.index)
		switch {
		case n > 0:
			r.index += int64(n)
			return n, err
		case err == nil:
		case errors.Is(err, io.EOF):

		default:
			return 0, err
		}

		select {
		case <-r.closer.Wait():
			return 0, ErrIsClosed
		case <-r.s.waiter.Wait():
		}
	}
}

// Seek updates the reader offset using whence semantics.
// SeekStart sets the absolute position to offset, SeekCurrent moves relative
// to the current position, and SeekEnd returns ErrSeekEndNotSupported.
// If the computed position is negative, the position is clamped to 0 and
// ErrNegativeIndex is returned.
func (r *reader) Seek(offset int64, whence int) (pos int64, err error) {
	switch whence {
	case io.SeekStart:
		r.index = offset
	case io.SeekCurrent:
		r.index += offset
	case io.SeekEnd:
		return 0, ErrSeekEndNotSupported
	default:
		return 0, ErrInvalidWhence
	}

	if r.index < 0 {
		r.index = 0
		err = ErrNegativeIndex
	}

	return r.index, err
}

// Close closes the reader and unblocks any pending Read calls.
// Subsequent Read calls return ErrIsClosed when no bytes are read.
func (r *reader) Close() (err error) {
	if err = r.closer.Close(); err != nil {
		return err
	}

	r.s.wg.Done()
	return nil
}
