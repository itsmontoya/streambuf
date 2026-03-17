package streambuf

import (
	"context"
)

// New constructs a new file Buffer.
func New(filepath string) (out *Buffer, err error) {
	var w writable
	if w, err = newWritableFile(filepath); err != nil {
		return
	}

	var r readable
	if r, err = newReadableFile(filepath); err != nil {
		return
	}

	return newWithBackend(w, r), nil
}

// NewMemory constructs a new in-memory Buffer.
func NewMemory() (out *Buffer) {
	w := newWritableMemory(nil)
	r := newReadableMemory(w.m)
	return newWithBackend(w, r)
}

func newWithBackend(w writable, r readable) (out *Buffer) {
	var b Buffer
	b.w = w
	b.stream = newStreamWithReadable(r)
	return &b
}

// Buffer is a thread-safe byte buffer with reader support.
type Buffer struct {
	*stream

	w writable
}

// Write appends bytes to the buffer and wakes waiting readers.
// It returns ErrIsClosed if the buffer has been closed.
func (b *Buffer) Write(bs []byte) (n int, err error) {
	b.mux.RLock()
	defer b.mux.RUnlock()
	if n, err = b.w.Write(bs); err != nil {
		return n, err
	}

	if err = b.waiter.Refresh(); err != nil {
		return n, err
	}

	return n, err
}

// Close closes the writer side of the buffer and signals waiting readers.
// It does not wait for readers to call Close.
func (b *Buffer) Close() (err error) {
	return b.CloseAndWait(expiredContext)
}

// CloseAndWait closes the writer side of the buffer and signals waiting readers.
// It waits for readers to close until ctx is canceled.
// Once called, future Reader and Write calls return ErrIsClosed.
// ctx must be non-nil.
// If ctx is canceled before readers close, this call still returns and the
// buffer remains closed; readers should still be closed to complete internal
// wait cleanup.
func (b *Buffer) CloseAndWait(ctx context.Context) (err error) {
	b.mux.Lock()
	defer b.mux.Unlock()
	if b.closed {
		return ErrIsClosed
	}

	b.closed = true

	if err = b.w.Close(); err != nil {
		return err
	}

	if err = b.waiter.Close(); err != nil {
		return err
	}

	b.waitUntilDone(ctx)

	return b.r.Close()
}
