package streambuf

import (
	"context"
	"io"
	"sync"
)

// NewStream constructs a read-only file-backed Stream.
func NewStream(filepath string) (out *Stream, err error) {
	var r readable
	if r, err = newReadableFile(filepath); err != nil {
		return nil, err
	}

	var s Stream
	s.stream = newStreamWithReadable(r)
	return &s, nil
}

// NewMemoryStream constructs a read-only memory-backed Stream over bs.
func NewMemoryStream(bs []byte) (out *Stream) {
	var s Stream
	r := newReadableMemory(newMemory(bs))
	s.stream = newStreamWithReadable(r)
	return &s
}

// Stream is a thread-safe read-only stream with reader support.
type Stream struct {
	*stream
}

func newStreamWithReadable(r readable) (out *stream) {
	var s stream
	s.r = r
	s.waiter = newWaiter()
	return &s
}

// stream contains the shared reader and lifecycle behavior used by Buffer and Stream.
type stream struct {
	mux sync.RWMutex
	wg  sync.WaitGroup

	r      readable
	waiter *waiter

	closed bool
}

// Reader returns a new io.ReadSeekCloser that tracks its own read offset and
// supports seeking relative to the start or current position.
// When the reader reaches the current end, Read returns EOF instead of waiting
// for future bytes. It returns ErrIsClosed if the stream is closed.
func (s *stream) Reader() (r io.ReadSeekCloser, err error) {
	if err = s.checkoutReader(); err != nil {
		return nil, err
	}

	return newReader(s, false), nil
}

// Close closes the stream and signals waiting readers.
// It does not wait for readers to call Close.
func (s *stream) Close() (err error) {
	return s.CloseAndWait(expiredContext)
}

// CloseAndWait closes the stream and signals waiting readers.
// It waits for readers to close until ctx is canceled.
// Once called, future Reader calls return ErrIsClosed.
// ctx must be non-nil.
// If ctx is canceled before readers close, this call still returns and the
// stream remains closed; readers should still be closed to complete internal
// wait cleanup.
func (s *stream) CloseAndWait(ctx context.Context) (err error) {
	s.mux.Lock()
	defer s.mux.Unlock()
	if s.closed {
		return ErrIsClosed
	}

	s.closed = true

	if err = s.r.Close(); err != nil {
		return err
	}

	if err = s.waiter.Close(); err != nil {
		return err
	}

	s.waitUntilDone(ctx)
	return nil
}

func (s *stream) checkoutReader() (err error) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	if s.closed {
		return ErrIsClosed
	}

	s.wg.Add(1)
	return nil
}

func (s *stream) waitUntilDone(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-s.waitForReaders():
	}
}

func (s *stream) waitForReaders() (out <-chan struct{}) {
	done := make(chan struct{}, 1)
	go func() {
		s.wg.Wait()
		done <- struct{}{}
	}()

	return done
}

func (s *stream) isClosed() (closed bool) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.closed
}
