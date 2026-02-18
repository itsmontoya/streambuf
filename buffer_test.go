package streambuf

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
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

		if err := b.CloseAndWait(nil); err != ErrIsClosed {
			t.Fatalf("CloseAndWait with closed waiter = %v, want %v", err, ErrIsClosed)
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
