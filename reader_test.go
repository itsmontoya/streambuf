package streambuf

import (
	"context"
	"io"
	"testing"
	"testing/synctest"
	"time"
)

func TestReaderReadExistingData(t *testing.T) {
	runForEachBackend(t, func(t *testing.T, b *Buffer) {
		_, _ = b.Write([]byte("hello"))
		_, _ = b.Write([]byte(" world"))

		var r io.ReadCloser
		r = mustReader(t, b)

		assertRead(t, r, 2, "he")
		assertRead(t, r, 16, "llo world")

		if err := b.Close(); err != nil {
			t.Fatalf("Close = %v, want nil", err)
		}

		var buf []byte
		buf = make([]byte, 16)

		var (
			n   int
			err error
		)
		n, err = r.Read(buf)
		if err != ErrIsClosed || n != 0 {
			t.Fatalf("Read after close = (%d, %v), want (0, %v)", n, err, ErrIsClosed)
		}
	})
}

func TestReaderReturnsErrIsClosedWhenBufferClosedAndEmpty(t *testing.T) {
	runForEachBackend(t, func(t *testing.T, b *Buffer) {
		if err := b.Close(); err != nil {
			t.Fatalf("Close = %v, want nil", err)
		}

		var r io.ReadSeekCloser
		var err error
		r, err = b.Reader()
		if err != ErrIsClosed || r != nil {
			t.Fatalf("Reader() = (%v, %v), want (nil, %v)", r, err, ErrIsClosed)
		}
	})
}

func TestReaderWaitsForData(t *testing.T) {
	runForEachBackend(t, func(t *testing.T, b *Buffer) {
		synctest.Test(t, func(t *testing.T) {
			var r io.ReadCloser
			r = mustReader(t, b)

			var buf []byte
			buf = make([]byte, 8)

			var started chan struct{}
			started = make(chan struct{})

			var (
				n   int
				err error
			)
			go func() {
				close(started)
				n, err = r.Read(buf)
			}()

			<-started
			_, _ = b.Write([]byte("late"))
			synctest.Wait()

			if err != nil || n != len("late") {
				t.Fatalf("Read = (%d, %v), want (%d, nil)", n, err, len("late"))
			}
			if got := string(buf[:n]); got != "late" {
				t.Fatalf("Read data = %q, want %q", got, "late")
			}

			if closeErr := r.Close(); closeErr != nil {
				t.Fatalf("Close = %v, want nil", closeErr)
			}

			if closeErr := b.CloseAndWait(context.Background()); closeErr != nil {
				t.Fatalf("CloseAndWait = %v, want nil", closeErr)
			}
		})
	})
}

func TestReaderCloseUnblocks(t *testing.T) {
	runForEachBackend(t, func(t *testing.T, b *Buffer) {
		synctest.Test(t, func(t *testing.T) {
			var r io.ReadCloser
			r = mustReader(t, b)

			var (
				n   int
				err error
			)

			var started chan struct{}
			started = make(chan struct{})
			go func() {
				var buf []byte
				buf = make([]byte, 4)
				close(started)
				n, err = r.Read(buf)
			}()

			<-started
			if closeErr := r.Close(); closeErr != nil {
				t.Fatalf("Close = %v, want nil", closeErr)
			}
			synctest.Wait()

			if err != ErrIsClosed || n != 0 {
				t.Fatalf("Read after close = (%d, %v), want (0, %v)", n, err, ErrIsClosed)
			}
		})
	})
}

func TestReaderMultipleReadersReceiveAllData(t *testing.T) {
	runForEachBackend(t, func(t *testing.T, b *Buffer) {
		var r1 io.ReadCloser
		r1 = mustReader(t, b)

		var r2 io.ReadCloser
		r2 = mustReader(t, b)

		_, _ = b.Write([]byte("hello "))
		_, _ = b.Write([]byte("world"))

		var cancel context.CancelFunc
		var ctx context.Context
		ctx, cancel = context.WithCancel(context.Background())
		defer cancel()

		var closeDone chan error
		closeDone = make(chan error, 1)
		go func() {
			closeDone <- b.CloseAndWait(ctx)
		}()

		assertRead(t, r1, len("hello world"), "hello world")
		assertRead(t, r2, len("hello world"), "hello world")

		var buf []byte
		buf = make([]byte, 1)

		var (
			n1   int
			err1 error
		)
		n1, err1 = r1.Read(buf)
		if err1 != ErrIsClosed || n1 != 0 {
			t.Fatalf("r1 terminal read = (%d, %v), want (0, %v)", n1, err1, ErrIsClosed)
		}

		var (
			n2   int
			err2 error
		)
		n2, err2 = r2.Read(buf)
		if err2 != ErrIsClosed || n2 != 0 {
			t.Fatalf("r2 terminal read = (%d, %v), want (0, %v)", n2, err2, ErrIsClosed)
		}

		if err := r1.Close(); err != nil {
			t.Fatalf("r1 Close = %v, want nil", err)
		}
		if err := r2.Close(); err != nil {
			t.Fatalf("r2 Close = %v, want nil", err)
		}

		select {
		case err := <-closeDone:
			if err != nil {
				t.Fatalf("CloseAndWait = %v, want nil", err)
			}
		case <-time.After(time.Second):
			t.Fatalf("CloseAndWait did not return")
		}
	})
}

func TestReaderCloseIdempotent(t *testing.T) {
	runForEachBackend(t, func(t *testing.T, b *Buffer) {
		var r io.ReadCloser
		r = mustReader(t, b)

		if err := r.Close(); err != nil {
			t.Fatalf("Close = %v, want nil", err)
		}

		if err := r.Close(); err != ErrIsClosed {
			t.Fatalf("Close second call = %v, want %v", err, ErrIsClosed)
		}
	})
}

func TestReaderZeroLengthReadReturnsImmediately(t *testing.T) {
	runForEachBackend(t, func(t *testing.T, b *Buffer) {
		var (
			n   int
			err error
		)

		r := mustReader(t, b)
		done := make(chan struct{})

		go func() {
			var empty []byte
			empty = make([]byte, 0)
			n, err = r.Read(empty)
			close(done)
		}()

		select {
		case <-done:
			if err != nil || n != 0 {
				t.Fatalf("Read(empty) = (%d, %v), want (0, nil)", n, err)
			}
		case <-time.After(time.Second):
			if closeErr := r.Close(); closeErr != nil {
				t.Fatalf("Close = %v, want nil", closeErr)
			}

			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatalf("Read(empty) blocked and did not unblock after Close")
			}

			t.Fatalf("Read(empty) blocked, want immediate (0, nil)")
		}

		if closeErr := r.Close(); closeErr != nil {
			t.Fatalf("Close = %v, want nil", closeErr)
		}
	})
}

func TestReaderReadAfterClose(t *testing.T) {
	runForEachBackend(t, func(t *testing.T, b *Buffer) {
		var r io.ReadCloser
		r = mustReader(t, b)

		if err := r.Close(); err != nil {
			t.Fatalf("Close = %v, want nil", err)
		}

		var buf []byte
		buf = make([]byte, 4)

		var (
			n   int
			err error
		)
		n, err = r.Read(buf)
		if err != ErrIsClosed || n != 0 {
			t.Fatalf("Read after close = (%d, %v), want (0, %v)", n, err, ErrIsClosed)
		}
	})
}

func TestReaderReadsRemainderAfterBufferClose(t *testing.T) {
	runForEachBackend(t, func(t *testing.T, b *Buffer) {
		_, _ = b.Write([]byte("abcdef"))

		var r io.ReadCloser
		r = mustReader(t, b)

		assertRead(t, r, 3, "abc")

		var cancel context.CancelFunc
		var ctx context.Context
		ctx, cancel = context.WithCancel(context.Background())
		defer cancel()

		var closeDone chan error
		closeDone = make(chan error, 1)
		go func() {
			closeDone <- b.CloseAndWait(ctx)
		}()

		assertRead(t, r, 16, "def")

		var buf []byte
		buf = make([]byte, 1)

		var (
			n   int
			err error
		)
		n, err = r.Read(buf)
		if err != ErrIsClosed || n != 0 {
			t.Fatalf("terminal read = (%d, %v), want (0, %v)", n, err, ErrIsClosed)
		}

		if closeErr := r.Close(); closeErr != nil {
			t.Fatalf("Close = %v, want nil", closeErr)
		}

		select {
		case closeErr := <-closeDone:
			if closeErr != nil {
				t.Fatalf("CloseAndWait = %v, want nil", closeErr)
			}
		case <-time.After(time.Second):
			t.Fatalf("CloseAndWait did not return")
		}
	})
}

func TestBufferCloseIsImmediateWithOpenReader(t *testing.T) {
	runForEachBackend(t, func(t *testing.T, b *Buffer) {
		var r io.ReadCloser
		r = mustReader(t, b)

		var closeDone chan error
		closeDone = make(chan error, 1)
		go func() {
			closeDone <- b.Close()
		}()

		select {
		case err := <-closeDone:
			if err != nil {
				t.Fatalf("Close = %v, want nil", err)
			}
		case <-time.After(time.Second):
			t.Fatalf("Close did not return")
		}

		if err := r.Close(); err != nil {
			t.Fatalf("Close reader = %v, want nil", err)
		}
	})
}

func TestReaderUnreadDataDroppedAfterImmediateClose(t *testing.T) {
	runForEachBackend(t, func(t *testing.T, b *Buffer) {
		_, _ = b.Write([]byte("abcdef"))

		var r io.ReadCloser
		r = mustReader(t, b)
		assertRead(t, r, 3, "abc")

		if err := b.Close(); err != nil {
			t.Fatalf("Close = %v, want nil", err)
		}

		var buf []byte
		buf = make([]byte, 8)

		var (
			n   int
			err error
		)
		n, err = r.Read(buf)
		if err != ErrIsClosed || n != 0 {
			t.Fatalf("Read after immediate close = (%d, %v), want (0, %v)", n, err, ErrIsClosed)
		}

		if closeErr := r.Close(); closeErr != nil {
			t.Fatalf("Close reader = %v, want nil", closeErr)
		}
	})
}

func TestReaderSeekStart(t *testing.T) {
	runForEachBackend(t, func(t *testing.T, b *Buffer) {
		_, _ = b.Write([]byte("abcdef"))

		var r io.ReadSeekCloser
		r = mustReader(t, b)

		assertRead(t, r, 2, "ab")
		assertSeek(t, r, 1, io.SeekStart, 1, nil)
		assertRead(t, r, 3, "bcd")
	})
}

func TestReaderSeekCurrent(t *testing.T) {
	runForEachBackend(t, func(t *testing.T, b *Buffer) {
		_, _ = b.Write([]byte("abcdef"))

		var r io.ReadSeekCloser
		r = mustReader(t, b)

		assertRead(t, r, 3, "abc")
		assertSeek(t, r, -2, io.SeekCurrent, 1, nil)
		assertRead(t, r, 2, "bc")
	})
}

func TestReaderSeekCurrentClampsNegativeIndex(t *testing.T) {
	runForEachBackend(t, func(t *testing.T, b *Buffer) {
		_, _ = b.Write([]byte("abcdef"))

		var r io.ReadSeekCloser
		r = mustReader(t, b)

		assertRead(t, r, 1, "a")
		assertSeek(t, r, -10, io.SeekCurrent, 0, ErrNegativeIndex)
		assertRead(t, r, 2, "ab")
	})
}

func TestReaderSeekEndNotSupported(t *testing.T) {
	runForEachBackend(t, func(t *testing.T, b *Buffer) {
		_, _ = b.Write([]byte("abcdef"))

		var r io.ReadSeekCloser
		r = mustReader(t, b)

		assertSeek(t, r, 0, io.SeekEnd, 0, ErrSeekEndNotSupported)
		assertRead(t, r, 2, "ab")
	})
}

func TestReaderSeekInvalidWhence(t *testing.T) {
	runForEachBackend(t, func(t *testing.T, b *Buffer) {
		_, _ = b.Write([]byte("abcdef"))

		var r io.ReadSeekCloser
		r = mustReader(t, b)

		var (
			pos int64
			err error
		)

		if pos, err = r.Seek(0, -1); err == nil || pos != 0 {
			t.Fatalf("Seek(%d, %d) = (%d, %v), want (0, non-nil error)", 0, -1, pos, err)
		}

		assertRead(t, r, 2, "ab")
	})
}

func assertSeek(t *testing.T, r io.Seeker, offset int64, whence int, wantPos int64, wantErr error) {
	t.Helper()

	var (
		pos int64
		err error
	)
	pos, err = r.Seek(offset, whence)
	if pos != wantPos || err != wantErr {
		t.Fatalf("Seek(%d, %d) = (%d, %v), want (%d, %v)", offset, whence, pos, err, wantPos, wantErr)
	}
}

func assertRead(t *testing.T, r io.Reader, bufSize int, want string) {
	t.Helper()

	var buf []byte
	buf = make([]byte, bufSize)

	var (
		n   int
		err error
	)
	n, err = r.Read(buf)
	if err != nil || n != len(want) {
		t.Fatalf("Read = (%d, %v), want (%d, nil)", n, err, len(want))
	}
	if got := string(buf[:n]); got != want {
		t.Fatalf("Read data = %q, want %q", got, want)
	}
}

func mustReader(t *testing.T, b *Buffer) (out io.ReadSeekCloser) {
	t.Helper()

	var err error
	if out, err = b.Reader(); err != nil {
		t.Fatalf("Reader() = (%v, %v), want (non-nil, nil)", out, err)
	}

	return out
}
