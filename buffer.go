package streambuf

import (
	"errors"
	"io"
)

// ErrIsClosed is returned when an action is attempted on a closed instance
var ErrIsClosed = errors.New("cannot perform action on closed instance")

// New constructs a new file Buffer.
func New(filepath string) (out *Buffer, err error) {
	var b Buffer
	if b.b, err = newFile(filepath); err != nil {
		return
	}

	b.waiter = newWaiter()
	return &b, nil
}

// NewMemory constructs a new in-memory Buffer.
func NewMemory() (out *Buffer) {
	var b Buffer
	b.b = newMemory()
	b.waiter = newWaiter()
	return &b
}

// Buffer is a concurrent-safe byte buffer with reader support.
type Buffer struct {
	b backend

	waiter *waiter
}

// Write appends bytes to the buffer and wakes waiting readers.
func (b *Buffer) Write(bs []byte) (n int, err error) {
	if n, err = b.b.Write(bs); err != nil {
		return
	}

	b.waiter.Refresh()
	return
}

// Reader returns a new ReadSeekCloser that streams data from the buffer.
// Each reader tracks its own read offset and supports seeking relative to
// the start or current position.
func (b *Buffer) Reader() (r io.ReadSeekCloser) {
	return newReader(b)
}

// Close closes the buffer and signals any waiting readers.
func (b *Buffer) Close() (err error) {
	if err = b.b.Close(); err != nil {
		return
	}

	return b.waiter.Close()
}
