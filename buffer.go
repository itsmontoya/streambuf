package streambuf

import (
	"context"
	"errors"
	"io"
	"sync"
)

// ErrIsClosed is returned when an action is attempted on a closed instance.
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
	mux sync.RWMutex
	b   backend
	wg  sync.WaitGroup

	waiter *waiter

	closed bool
}

// Write appends bytes to the buffer and wakes waiting readers.
// It returns ErrIsClosed if the buffer has been closed.
func (b *Buffer) Write(bs []byte) (n int, err error) {
	b.mux.RLock()
	defer b.mux.RUnlock()
	if n, err = b.b.Write(bs); err != nil {
		return
	}

	if err = b.waiter.Refresh(); err != nil {
		return
	}

	return
}

// Reader returns a new ReadSeekCloser that streams data from the buffer.
// Each reader tracks its own read offset and supports seeking relative to
// the start or current position. It returns ErrIsClosed if the buffer is closed.
func (b *Buffer) Reader() (r io.ReadSeekCloser, err error) {
	b.mux.RLock()
	defer b.mux.RUnlock()
	if b.closed {
		return nil, ErrIsClosed
	}

	b.wg.Add(1)
	return newReader(b), nil
}

// Close closes the writer side of the buffer and signals waiting readers.
// It does not wait for readers to call Close.
func (b *Buffer) Close() (err error) {
	return b.CloseAndWait(expiredContext)
}

// CloseAndWait closes the writer side of the buffer and signals waiting readers.
// It waits for readers to close until ctx is canceled.
// Once called successfully, future Reader and Write calls return ErrIsClosed.
// ctx must be non-nil.
func (b *Buffer) CloseAndWait(ctx context.Context) (err error) {
	b.mux.Lock()
	defer b.mux.Unlock()
	if b.closed {
		return ErrIsClosed
	}

	b.closed = true

	if err = b.b.CloseWriter(); err != nil {
		return
	}

	if err = b.waiter.Close(); err != nil {
		return
	}

	b.waitUntilDone(ctx)

	return b.b.CloseReader()
}

func (b *Buffer) waitUntilDone(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-b.waitForReaders():
	}
}

func (b *Buffer) waitForReaders() (out <-chan struct{}) {
	done := make(chan struct{}, 1)
	go func() {
		b.wg.Wait()
		done <- struct{}{}
	}()

	return done
}
