package streambuf

import (
	"errors"
	"io"
)

// ErrIsClosed is returned when an action is attempted on a closed instance
var ErrIsClosed = errors.New("cannot perform action on closed instance")

// New constructs a Buffer bound to the provided context.
func New() (out *Buffer) {
	var b Buffer
	b.b = newMemory()
	b.waiter = newWaiter()
	out = &b
	return out
}

// Buffer is a concurrent-safe byte buffer with reader support.
type Buffer struct {
	b backend

	waiter *waiter
}

func (b *Buffer) Write(bs []byte) (n int, err error) {
	if n, err = b.b.Write(bs); err != nil {
		return
	}

	b.waiter.Refresh()
	return
}

// Reader returns a new ReadCloser that streams data from the buffer.
func (b *Buffer) Reader() (r io.ReadCloser) {
	return newReader(b)
}

func (b *Buffer) Close() (err error) {
	if err = b.b.Close(); err != nil {
		return
	}

	return b.waiter.Close()
}
