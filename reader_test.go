package streambuf

import (
	"io"
	"testing"
	"testing/synctest"
)

func TestReaderReadExistingData(t *testing.T) {
	var b *Buffer
	b = New()

	_, _ = b.Write([]byte("hello"))
	_, _ = b.Write([]byte(" world"))

	var r io.ReadCloser
	r = b.Reader()

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
}

func TestReaderReturnsErrIsClosedWhenBufferClosedAndEmpty(t *testing.T) {
	var b *Buffer
	b = New()

	if err := b.Close(); err != nil {
		t.Fatalf("Close = %v, want nil", err)
	}

	var r io.ReadCloser
	r = b.Reader()

	var buf []byte
	buf = make([]byte, 1)

	var (
		n   int
		err error
	)
	n, err = r.Read(buf)
	if err != ErrIsClosed || n != 0 {
		t.Fatalf("Read = (%d, %v), want (0, %v)", n, err, ErrIsClosed)
	}
}

func TestReaderWaitsForData(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var b *Buffer
		b = New()

		var r io.ReadCloser
		r = b.Reader()

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
		_ = b.Close()
		synctest.Wait()

		if err != nil || n != len("late") {
			t.Fatalf("Read = (%d, %v), want (%d, nil)", n, err, len("late"))
		}
		if got := string(buf[:n]); got != "late" {
			t.Fatalf("Read data = %q, want %q", got, "late")
		}
	})
}

func TestReaderCloseUnblocks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var b *Buffer
		b = New()

		var r io.ReadCloser
		r = b.Reader()

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

		if err != io.EOF || n != 0 {
			t.Fatalf("Read after close = (%d, %v), want (0, io.EOF)", n, err)
		}
	})
}

func TestReaderMultipleReadersReceiveAllData(t *testing.T) {
	var b *Buffer
	b = New()

	var r1 io.ReadCloser
	r1 = b.Reader()

	var r2 io.ReadCloser
	r2 = b.Reader()

	_, _ = b.Write([]byte("hello "))
	_, _ = b.Write([]byte("world"))
	if err := b.Close(); err != nil {
		t.Fatalf("Close = %v, want nil", err)
	}

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
}

func TestReaderCloseIdempotent(t *testing.T) {
	var b *Buffer
	b = New()

	var r io.ReadCloser
	r = b.Reader()

	if err := r.Close(); err != nil {
		t.Fatalf("Close = %v, want nil", err)
	}

	if err := r.Close(); err != ErrIsClosed {
		t.Fatalf("Close second call = %v, want %v", err, ErrIsClosed)
	}
}

func TestReaderReadAfterClose(t *testing.T) {
	var b *Buffer
	b = New()

	var r io.ReadCloser
	r = b.Reader()

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
	if err != io.EOF || n != 0 {
		t.Fatalf("Read after close = (%d, %v), want (0, io.EOF)", n, err)
	}
}

func TestReaderReadsRemainderAfterBufferClose(t *testing.T) {
	var b *Buffer
	b = New()

	_, _ = b.Write([]byte("abcdef"))

	var r io.ReadCloser
	r = b.Reader()

	assertRead(t, r, 3, "abc")

	if err := b.Close(); err != nil {
		t.Fatalf("Close = %v, want nil", err)
	}

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
