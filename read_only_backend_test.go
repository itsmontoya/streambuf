package streambuf

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

type backendContract struct {
	name                     string
	new                      func(t *testing.T) backend
	readOnly                 bool
	wantWriteN               int
	wantWriteErr             error
	wantFirstCloseWriterErr  error
	wantSecondCloseWriterErr error
}

func runForEachBackendContract(t *testing.T, fn func(t *testing.T, tc backendContract, b backend)) {
	t.Helper()

	var tests []backendContract
	tests = []backendContract{
		{
			name: "memory",
			new: func(t *testing.T) (out backend) {
				t.Helper()

				var buf *Buffer
				buf = NewMemory()

				var (
					n   int
					err error
				)
				n, err = buf.b.Write([]byte("hello world"))
				if err != nil || n != len("hello world") {
					t.Fatalf("memory backend seed Write = (%d, %v), want (%d, nil)", n, err, len("hello world"))
				}

				return buf.b
			},
			wantWriteN:               len("x"),
			wantWriteErr:             nil,
			wantFirstCloseWriterErr:  nil,
			wantSecondCloseWriterErr: ErrIsClosed,
		},
		{
			name: "file",
			new: func(t *testing.T) (out backend) {
				t.Helper()

				var path string
				path = filepath.Join(t.TempDir(), "streambuf.backend.test")

				var (
					buf *Buffer
					err error
				)
				if buf, err = New(path); err != nil {
					t.Fatalf("New(%q) = %v, want nil", path, err)
				}

				var n int
				n, err = buf.b.Write([]byte("hello world"))
				if err != nil || n != len("hello world") {
					t.Fatalf("file backend seed Write = (%d, %v), want (%d, nil)", n, err, len("hello world"))
				}

				return buf.b
			},
			wantWriteN:               len("x"),
			wantWriteErr:             nil,
			wantFirstCloseWriterErr:  nil,
			wantSecondCloseWriterErr: ErrIsClosed,
		},
		{
			name: "read_only_file",
			new: func(t *testing.T) (out backend) {
				t.Helper()

				var path string
				path = filepath.Join(t.TempDir(), "streambuf.read_only.test")

				var err error
				if err = os.WriteFile(path, []byte("hello world"), 0644); err != nil {
					t.Fatalf("WriteFile(%q) = %v, want nil", path, err)
				}

				var buf *Buffer
				if buf, err = NewReadOnly(path); err != nil {
					t.Fatalf("NewReadOnly(%q) = %v, want nil", path, err)
				}

				return buf.b
			},
			readOnly:                 true,
			wantWriteN:               0,
			wantWriteErr:             ErrCannotWriteToReadOnly,
			wantFirstCloseWriterErr:  nil,
			wantSecondCloseWriterErr: nil,
		},
	}

	for _, tt := range tests {
		var tc backendContract
		tc = tt

		t.Run(tc.name, func(t *testing.T) {
			var b backend
			b = tc.new(t)
			fn(t, tc, b)
		})
	}
}

func runForEachReadOnlyBackend(t *testing.T, fn func(t *testing.T, tc backendContract, b backend)) {
	t.Helper()

	runForEachBackendContract(t, func(t *testing.T, tc backendContract, b backend) {
		if !tc.readOnly {
			return
		}

		fn(t, tc, b)
	})
}

func TestBackendContractWrite(t *testing.T) {
	runForEachBackendContract(t, func(t *testing.T, tc backendContract, b backend) {
		var (
			n   int
			err error
		)
		n, err = b.Write([]byte("x"))
		if err != tc.wantWriteErr || n != tc.wantWriteN {
			t.Fatalf("Write = (%d, %v), want (%d, %v)", n, err, tc.wantWriteN, tc.wantWriteErr)
		}
	})
}

func TestBackendContractCloseWriter(t *testing.T) {
	runForEachBackendContract(t, func(t *testing.T, tc backendContract, b backend) {
		var err error
		err = b.CloseWriter()
		if err != tc.wantFirstCloseWriterErr {
			t.Fatalf("CloseWriter first call = %v, want %v", err, tc.wantFirstCloseWriterErr)
		}

		err = b.CloseWriter()
		if err != tc.wantSecondCloseWriterErr {
			t.Fatalf("CloseWriter second call = %v, want %v", err, tc.wantSecondCloseWriterErr)
		}
	})
}

func TestBackendContractReadAndCloseReader(t *testing.T) {
	runForEachBackendContract(t, func(t *testing.T, tc backendContract, b backend) {
		var (
			n   int
			err error
		)

		var in []byte
		in = make([]byte, len("hello"))
		n, err = b.ReadAt(in, 0)
		if err != nil || n != len("hello") {
			t.Fatalf("ReadAt = (%d, %v), want (%d, nil)", n, err, len("hello"))
		}
		if got := string(in[:n]); got != "hello" {
			t.Fatalf("ReadAt data = %q, want %q", got, "hello")
		}

		if err = b.CloseReader(); err != nil {
			t.Fatalf("CloseReader first call = %v, want nil", err)
		}

		err = b.CloseReader()
		if err != ErrIsClosed {
			t.Fatalf("CloseReader second call = %v, want %v", err, ErrIsClosed)
		}
	})
}

func TestReadOnlyBackendContractReadAfterCloseReader(t *testing.T) {
	runForEachReadOnlyBackend(t, func(t *testing.T, tc backendContract, b backend) {
		var (
			n   int
			err error
		)

		var in []byte
		in = make([]byte, 1)
		n, err = b.ReadAt(in, 0)
		if err != nil && err != io.EOF {
			t.Fatalf("initial ReadAt = (%d, %v), want data or EOF-compatible setup", n, err)
		}

		if err = b.CloseReader(); err != nil {
			t.Fatalf("CloseReader first call = %v, want nil", err)
		}

		in = make([]byte, 1)
		n, err = b.ReadAt(in, 0)
		if err != ErrIsClosed || n != 0 {
			t.Fatalf("ReadAt after CloseReader = (%d, %v), want (0, %v)", n, err, ErrIsClosed)
		}
	})
}
