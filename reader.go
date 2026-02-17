package streambuf

import (
	"io"
)

var _ io.ReadCloser = &reader{}

func newReader(b *Buffer) (out *reader) {
	var r reader
	r.b = b
	r.waiter = newWaiter()
	r.closer = newWaiter()
	return &r
}

type reader struct {
	b *Buffer

	index int

	waiter *waiter
	closer *waiter
}

func (r *reader) Read(in []byte) (n int, err error) {
	for {
		n, err = r.b.b.ReadAt(in, r.index)
		switch {
		case n > 0:
			r.index += n
			return n, err
		case err == nil:
		case err == io.EOF:

		default:
			return 0, err
		}

		select {
		case <-r.closer.Wait():
			return 0, io.EOF
		case <-r.b.waiter.Wait():
		}

	}
}

func (r *reader) Close() (err error) {
	return r.closer.Close()
}
