package streambuf

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestBufferWriteAfterClose(t *testing.T) {
	runForEachBackend(t, func(t *testing.T, b *Buffer) {
		if err := b.Close(); err != nil {
			t.Fatalf("Close = %v, want nil", err)
		}

		var (
			n   int
			err error
		)
		n, err = b.Write([]byte("hello"))
		if err != ErrIsClosed || n != 0 {
			t.Fatalf("Write after close = (%d, %v), want (0, %v)", n, err, ErrIsClosed)
		}
	})
}

func TestBufferCloseAfterClose(t *testing.T) {
	runForEachBackend(t, func(t *testing.T, b *Buffer) {
		if err := b.Close(); err != nil {
			t.Fatalf("Close = %v, want nil", err)
		}

		if err := b.Close(); err != ErrIsClosed {
			t.Fatalf("Close second call = %v, want %v", err, ErrIsClosed)
		}
	})
}

func TestBufferSignalsWaitersOnWriteAndClose(t *testing.T) {
	runForEachBackend(t, func(t *testing.T, b *Buffer) {
		var first <-chan struct{}
		first = b.waiter.Wait()

		select {
		case <-first:
			t.Fatalf("initial waiter channel should be open")
		default:
		}

		if _, err := b.Write([]byte("x")); err != nil {
			t.Fatalf("Write = %v, want nil", err)
		}

		select {
		case <-first:
		default:
			t.Fatalf("waiter channel should close after write")
		}

		var second <-chan struct{}
		second = b.waiter.Wait()
		if first == second {
			t.Fatalf("waiter channel should refresh after write")
		}

		select {
		case <-second:
			t.Fatalf("refreshed waiter channel should be open before close")
		default:
		}

		if err := b.Close(); err != nil {
			t.Fatalf("Close = %v, want nil", err)
		}

		select {
		case <-second:
		default:
			t.Fatalf("waiter channel should close after buffer close")
		}
	})
}

func TestBackendCloseReaderAfterCloseReader(t *testing.T) {
	runForEachBackend(t, func(t *testing.T, b *Buffer) {
		if err := b.b.CloseReader(); err != nil {
			t.Fatalf("CloseReader = %v, want nil", err)
		}

		if err := b.b.CloseReader(); err != ErrIsClosed {
			t.Fatalf("CloseReader second call = %v, want %v", err, ErrIsClosed)
		}
	})
}

func TestBufferCloseAndWaitWhenWaiterAlreadyClosed(t *testing.T) {
	runForEachBackend(t, func(t *testing.T, b *Buffer) {
		if err := b.waiter.Close(); err != nil {
			t.Fatalf("waiter.Close = %v, want nil", err)
		}

		if err := b.CloseAndWait(context.Background()); err != ErrIsClosed {
			t.Fatalf("CloseAndWait with closed waiter = %v, want %v", err, ErrIsClosed)
		}
	})
}

func TestBufferCloseAndWaitReturnsOnContextCancelWithOpenReader(t *testing.T) {
	runForEachBackend(t, func(t *testing.T, b *Buffer) {
		var r io.ReadCloser
		var err error
		if r, err = b.Reader(); err != nil {
			t.Fatalf("Reader() = (%v, %v), want (non-nil, nil)", r, err)
		}
		defer r.Close()

		var cancel context.CancelFunc
		var ctx context.Context
		ctx, cancel = context.WithCancel(context.Background())
		cancel()

		var closeDone chan error
		closeDone = make(chan error, 1)
		go func() {
			closeDone <- b.CloseAndWait(ctx)
		}()

		select {
		case err = <-closeDone:
			if err != nil {
				t.Fatalf("CloseAndWait(cancelled) = %v, want nil", err)
			}
		case <-time.After(time.Second):
			t.Fatalf("CloseAndWait(cancelled) did not return with open reader")
		}
	})
}

func TestBufferCloseAndWaitCancelStillClosesBufferState(t *testing.T) {
	runForEachBackend(t, func(t *testing.T, b *Buffer) {
		var r io.ReadCloser
		var err error
		if r, err = b.Reader(); err != nil {
			t.Fatalf("Reader() = (%v, %v), want (non-nil, nil)", r, err)
		}

		var cancel context.CancelFunc
		var ctx context.Context
		ctx, cancel = context.WithTimeout(context.Background(), time.Nanosecond)
		defer cancel()

		if err = b.CloseAndWait(ctx); err != nil {
			t.Fatalf("CloseAndWait(expired ctx) = %v, want nil", err)
		}

		var (
			n int
		)
		n, err = b.Write([]byte("x"))
		if err != ErrIsClosed || n != 0 {
			t.Fatalf("Write after canceled CloseAndWait = (%d, %v), want (0, %v)", n, err, ErrIsClosed)
		}

		var rs io.ReadSeekCloser
		rs, err = b.Reader()
		if err != ErrIsClosed || rs != nil {
			t.Fatalf("Reader after canceled CloseAndWait = (%v, %v), want (nil, %v)", rs, err, ErrIsClosed)
		}

		if err = r.Close(); err != nil {
			t.Fatalf("reader Close = %v, want nil", err)
		}
	})
}

func TestNewWhenFileOpenFails(t *testing.T) {
	var (
		b   *Buffer
		err error
	)

	path := filepath.Join(t.TempDir(), "missing", "streambuf.test")
	if b, err = New(path); err == nil {
		t.Fatalf("New(%q) = (%v, nil), want non-nil error", path, b)
	}

	if b != nil {
		t.Fatalf("New(%q) buffer = %v, want nil", path, b)
	}
}

func TestNewWhenFileReadOpenFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission semantics differ on windows")
	}

	path := filepath.Join(t.TempDir(), "streambuf.test")

	var err error
	if err = os.WriteFile(path, []byte("seed"), 0200); err != nil {
		t.Fatalf("WriteFile(%q) = %v, want nil", path, err)
	}

	if err = os.Chmod(path, 0200); err != nil {
		t.Fatalf("Chmod(%q) = %v, want nil", path, err)
	}

	var b *Buffer
	if b, err = New(path); err == nil {
		t.Fatalf("New(%q) = (%v, nil), want non-nil error", path, b)
	}

	if b != nil {
		t.Fatalf("New(%q) buffer = %v, want nil", path, b)
	}
}

func TestBufferCloseAndWaitCancelDoesNotLeakWaitForReadersGoroutine(t *testing.T) {
	var before int
	before = runtime.NumGoroutine()

	var i int
	for i = 0; i < 64; i++ {
		var b *Buffer
		b = NewMemory()

		var r io.ReadCloser
		var readErr error
		r, readErr = b.Reader()
		if readErr != nil {
			t.Fatalf("Reader() = (%v, %v), want (non-nil, nil)", r, readErr)
		}

		var cancel context.CancelFunc
		var ctx context.Context
		ctx, cancel = context.WithCancel(context.Background())
		cancel()

		if err := b.CloseAndWait(ctx); err != nil {
			t.Fatalf("CloseAndWait = %v, want nil", err)
		}

		if err := r.Close(); err != nil {
			t.Fatalf("reader Close = %v, want nil", err)
		}
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		runtime.GC()

		var after int
		after = runtime.NumGoroutine()
		if after <= before+8 {
			return
		}

		time.Sleep(time.Millisecond * 10)
	}

	var after int
	after = runtime.NumGoroutine()
	t.Fatalf("goroutines after cancel-path close = %d, before = %d, delta = %d", after, before, after-before)
}

func TestBufferConcurrentWriteAndCloseAndWait(t *testing.T) {
	runForEachBackend(t, func(t *testing.T, b *Buffer) {
		const (
			writerCount     = 16
			writesPerWriter = 64
		)

		var start chan struct{}
		start = make(chan struct{})

		var wg sync.WaitGroup

		var errs chan error
		errs = make(chan error, writerCount*writesPerWriter+1)

		var i int
		for i = 0; i < writerCount; i++ {
			wg.Add(1)

			go func() {
				defer wg.Done()
				<-start

				var j int
				for j = 0; j < writesPerWriter; j++ {
					if _, err := b.Write([]byte("x")); err != nil && err != ErrIsClosed {
						errs <- err
						return
					}
				}
			}()
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start

			var cancel context.CancelFunc
			var ctx context.Context
			ctx, cancel = context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			errs <- b.CloseAndWait(ctx)
		}()

		close(start)
		wg.Wait()
		close(errs)

		for err := range errs {
			if err != nil {
				t.Fatalf("concurrent Write/CloseAndWait error = %v, want nil", err)
			}
		}
	})
}
