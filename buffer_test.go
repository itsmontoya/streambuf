package streambuf

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
)

func Test_New_invalid_filepath(t *testing.T) {
	type testcase struct {
		name string // description of this test case

		init func(t *testing.T) (filepath string, err error)

		wantErr bool
	}

	tests := []testcase{
		{
			name: "missing parent directory",
			init: func(t *testing.T) (filepath string, err error) {
				t.Helper()
				return t.TempDir() + "/missing-dir/buffer.tmp", nil
			},
			wantErr: true,
		},
		{
			name: "empty filepath",
			init: func(t *testing.T) (filepath string, err error) {
				t.Helper()
				return "", nil
			},
			wantErr: true,
		},
		{
			name: "writer opens but reader open fails",
			init: func(t *testing.T) (filepath string, err error) {
				var f *os.File

				t.Helper()

				if f, err = os.CreateTemp(t.TempDir(), "new-invalid-*"); err != nil {
					return "", err
				}

				filepath = f.Name()

				if err = f.Close(); err != nil {
					return "", err
				}

				// Write-only file permissions allow newFile() to open the writer
				// handle while causing the subsequent reader open to fail.
				if err = os.Chmod(filepath, 0200); err != nil {
					return "", err
				}

				return filepath, nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				filepath string
				b        *Buffer
				err      error
			)

			if filepath, err = tt.init(t); err != nil {
				t.Fatal(err)
			}

			if b, err = New(filepath); err != nil {
				if !tt.wantErr {
					t.Fatalf("New() unexpected error: %v", err)
				}

				return
			}

			if tt.wantErr {
				t.Fatal("New() expected error, received <nil>")
			}

			t.Cleanup(func() {
				_ = b.Close()
			})
		})
	}
}

func Test_Buffer_Write(t *testing.T) {
	type testcase struct {
		name string // description of this test case

		init func(t *testing.T) (b *Buffer, err error)

		wantErr error
	}

	testInput := []byte("This is our test input!")
	tests := []testcase{
		{
			name: "memory",
			init: func(t *testing.T) (b *Buffer, err error) {
				t.Helper()
				return NewMemory(), nil
			},
		},
		{
			name: "file",
			init: func(t *testing.T) (b *Buffer, err error) {
				var f *os.File

				t.Helper()

				if f, err = os.CreateTemp(t.TempDir(), "buffer-write-*"); err != nil {
					return nil, err
				}

				if err = f.Close(); err != nil {
					return nil, err
				}

				if b, err = New(f.Name()); err != nil {
					return nil, err
				}

				t.Cleanup(func() {
					_ = b.Close()
				})

				return b, nil
			},
		},
		{
			name: "closed buffer",
			init: func(t *testing.T) (b *Buffer, err error) {
				t.Helper()

				b = NewMemory()
				if err = b.Close(); err != nil {
					return nil, err
				}

				return b, nil
			},
			wantErr: ErrIsClosed,
		},
		{
			name: "closed waiter",
			init: func(t *testing.T) (b *Buffer, err error) {
				t.Helper()

				b = NewMemory()
				if err = b.waiter.Close(); err != nil {
					return nil, err
				}

				return b, nil
			},
			wantErr: ErrIsClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				b      *Buffer
				err    error
				r      *reader
				bs     []byte
				gotN   int
				gotErr error
			)

			if b, err = tt.init(t); err != nil {
				t.Fatal(err)
			}

			gotN, gotErr = b.Write(testInput)
			if !isEqualErrors(gotErr, tt.wantErr) {
				t.Fatalf("Write() invalid error, expected <%v> and received <%v>", tt.wantErr, gotErr)
			}

			if gotErr != nil {
				return
			}

			if gotN != len(testInput) {
				t.Fatalf("Write() invalid n, expected <%v> and received <%v>", len(testInput), gotN)
			}

			r = newReader(b.stream)
			bs = make([]byte, len(testInput))
			gotN, gotErr = r.Read(bs)
			if gotErr != nil {
				t.Fatalf("Read() unexpected error after Write(): %v", gotErr)
			}

			if gotN != len(testInput) {
				t.Fatalf("Read() invalid n after Write(), expected <%v> and received <%v>", len(testInput), gotN)
			}

			if !bytes.Equal(bs, testInput) {
				t.Fatalf("Read() invalid read value after Write(), expected <%v> and received <%v>", string(testInput), string(bs))
			}
		})
	}
}

func Test_Buffer_Write_underlying_writer_closed(t *testing.T) {
	type testcase struct {
		name string // description of this test case

		init func(t *testing.T) (b *Buffer, err error)

		setup func(t *testing.T, b *Buffer)

		wantErr error
	}

	testInput := []byte("This is our test input!")
	tests := []testcase{
		{
			name: "memory",
			init: func(t *testing.T) (b *Buffer, err error) {
				t.Helper()
				return NewMemory(), nil
			},
			setup: func(t *testing.T, b *Buffer) {
				var err error

				t.Helper()

				if err = b.w.Close(); err != nil {
					t.Fatalf("setup CloseWriter() unexpected error: %v", err)
				}
			},
			wantErr: ErrIsClosed,
		},
		{
			name: "file",
			init: func(t *testing.T) (b *Buffer, err error) {
				var f *os.File

				t.Helper()

				if f, err = os.CreateTemp(t.TempDir(), "buffer-write-writer-closed-*"); err != nil {
					return nil, err
				}

				if err = f.Close(); err != nil {
					return nil, err
				}

				if b, err = New(f.Name()); err != nil {
					return nil, err
				}

				// If setup closes the raw writer handle, Buffer.Close() may return
				// early without closing the reader handle. Clean up the reader here.
				t.Cleanup(func() {
					_ = b.r.Close()
				})

				return b, nil
			},
			setup: func(t *testing.T, b *Buffer) {
				var err error

				t.Helper()

				if err = b.w.Close(); err != nil {
					t.Fatalf("setup CloseWriter() unexpected error: %v", err)
				}
			},
			wantErr: ErrIsClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				b      *Buffer
				err    error
				gotN   int
				gotErr error
			)

			if b, err = tt.init(t); err != nil {
				t.Fatal(err)
			}

			if tt.setup != nil {
				tt.setup(t, b)
			}

			gotN, gotErr = b.Write(testInput)
			if !isEqualErrors(gotErr, tt.wantErr) {
				t.Fatalf("Write() invalid error, expected <%v> and received <%v>", tt.wantErr, gotErr)
			}

			if gotErr != nil {
				if gotN != 0 {
					t.Fatalf("Write() invalid n, expected <0> and received <%v>", gotN)
				}

				return
			}

			t.Fatal("Write() expected error, received <nil>")
		})
	}
}

func Test_Buffer_Reader(t *testing.T) {
	type testcase struct {
		name string // description of this test case

		init func(t *testing.T) (b *Buffer, err error)

		wantErr error
	}

	testInput := []byte("This is our test input!")
	tests := []testcase{
		{
			name: "memory",
			init: func(t *testing.T) (b *Buffer, err error) {
				t.Helper()
				return NewMemory(), nil
			},
		},
		{
			name: "file",
			init: func(t *testing.T) (b *Buffer, err error) {
				var f *os.File

				t.Helper()

				if f, err = os.CreateTemp(t.TempDir(), "buffer-reader-*"); err != nil {
					return nil, err
				}

				if err = f.Close(); err != nil {
					return nil, err
				}

				if b, err = New(f.Name()); err != nil {
					return nil, err
				}

				t.Cleanup(func() {
					_ = b.Close()
				})

				return b, nil
			},
		},
		{
			name: "closed buffer",
			init: func(t *testing.T) (b *Buffer, err error) {
				t.Helper()

				b = NewMemory()
				if err = b.Close(); err != nil {
					return nil, err
				}

				return b, nil
			},
			wantErr: ErrIsClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				b      *Buffer
				err    error
				r      io.ReadSeekCloser
				bs     []byte
				gotN   int
				gotErr error
			)

			if b, err = tt.init(t); err != nil {
				t.Fatal(err)
			}

			b.Write(testInput)

			r, gotErr = b.Reader()
			if !isEqualErrors(gotErr, tt.wantErr) {
				t.Fatalf("Reader() invalid error, expected <%v> and received <%v>", tt.wantErr, gotErr)
			}

			if gotErr != nil {
				return
			}

			t.Cleanup(func() {
				_ = r.Close()
			})

			bs = make([]byte, len(testInput))
			gotN, gotErr = r.Read(bs)
			if gotErr != nil {
				t.Fatalf("Read() unexpected error from Reader(): %v", gotErr)
			}

			if gotN != len(testInput) {
				t.Fatalf("Read() invalid n from Reader(), expected <%v> and received <%v>", len(testInput), gotN)
			}

			if !bytes.Equal(bs, testInput) {
				t.Fatalf("Read() invalid read value from Reader(), expected <%v> and received <%v>", string(testInput), string(bs))
			}
		})
	}
}

func Test_Buffer_CloseAndWait(t *testing.T) {
	type testcase struct {
		name string // description of this test case

		init func(t *testing.T) (b *Buffer, err error)

		setup func(t *testing.T, b *Buffer)

		ctx context.Context

		wantErr error
	}

	tests := []testcase{
		{
			name: "memory",
			init: func(t *testing.T) (b *Buffer, err error) {
				t.Helper()
				return NewMemory(), nil
			},
			ctx: context.Background(),
		},
		{
			name: "file",
			init: func(t *testing.T) (b *Buffer, err error) {
				var f *os.File

				t.Helper()

				if f, err = os.CreateTemp(t.TempDir(), "buffer-close-and-wait-*"); err != nil {
					return nil, err
				}

				if err = f.Close(); err != nil {
					return nil, err
				}

				if b, err = New(f.Name()); err != nil {
					return nil, err
				}

				return b, nil
			},
			ctx: context.Background(),
		},
		{
			name: "closed buffer",
			init: func(t *testing.T) (b *Buffer, err error) {
				t.Helper()

				b = NewMemory()
				if err = b.Close(); err != nil {
					return nil, err
				}

				return b, nil
			},
			ctx:     context.Background(),
			wantErr: ErrIsClosed,
		},
		{
			name: "closed waiter",
			init: func(t *testing.T) (b *Buffer, err error) {
				t.Helper()

				b = NewMemory()
				if err = b.waiter.Close(); err != nil {
					return nil, err
				}

				return b, nil
			},
			ctx:     context.Background(),
			wantErr: ErrIsClosed,
		},
		{
			name: "reader open and canceled context",
			init: func(t *testing.T) (b *Buffer, err error) {
				t.Helper()
				return NewMemory(), nil
			},
			setup: func(t *testing.T, b *Buffer) {
				var (
					r   io.ReadSeekCloser
					err error
				)

				t.Helper()

				if r, err = b.Reader(); err != nil {
					t.Fatalf("setup Reader() unexpected error: %v", err)
				}

				t.Cleanup(func() {
					_ = r.Close()
				})
			},
			ctx: expiredContext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				b      *Buffer
				err    error
				gotErr error
			)

			if b, err = tt.init(t); err != nil {
				t.Fatal(err)
			}

			if tt.setup != nil {
				tt.setup(t, b)
			}

			gotErr = b.CloseAndWait(tt.ctx)
			if !isEqualErrors(gotErr, tt.wantErr) {
				t.Fatalf("CloseAndWait() invalid error, expected <%v> and received <%v>", tt.wantErr, gotErr)
			}
		})
	}
}

func Test_Buffer_CloseAndWait_underlying_writer_closed(t *testing.T) {
	type testcase struct {
		name string // description of this test case

		init func(t *testing.T) (b *Buffer, err error)

		setup func(t *testing.T, b *Buffer)

		wantErr error
	}

	tests := []testcase{
		{
			name: "memory",
			init: func(t *testing.T) (b *Buffer, err error) {
				t.Helper()
				return NewMemory(), nil
			},
			setup: func(t *testing.T, b *Buffer) {
				var err error

				t.Helper()

				if err = b.w.Close(); err != nil {
					t.Fatalf("setup CloseWriter() unexpected error: %v", err)
				}
			},
			wantErr: ErrIsClosed,
		},
		{
			name: "file",
			init: func(t *testing.T) (b *Buffer, err error) {
				var f *os.File

				t.Helper()

				if f, err = os.CreateTemp(t.TempDir(), "buffer-close-and-wait-writer-closed-*"); err != nil {
					return nil, err
				}

				if err = f.Close(); err != nil {
					return nil, err
				}

				if b, err = New(f.Name()); err != nil {
					return nil, err
				}

				// CloseAndWait() returns before closing the reader when CloseWriter()
				// fails, so close the file reader handle explicitly for this test.
				t.Cleanup(func() {
					_ = b.r.Close()
				})

				return b, nil
			},
			setup: func(t *testing.T, b *Buffer) {
				var err error

				t.Helper()

				if err = b.w.Close(); err != nil {
					t.Fatalf("setup CloseWriter() unexpected error: %v", err)
				}
			},
			wantErr: ErrIsClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				b      *Buffer
				err    error
				gotErr error
			)

			if b, err = tt.init(t); err != nil {
				t.Fatal(err)
			}

			if tt.setup != nil {
				tt.setup(t, b)
			}

			gotErr = b.CloseAndWait(context.Background())
			if !isEqualErrors(gotErr, tt.wantErr) {
				t.Fatalf("CloseAndWait() invalid error, expected <%v> and received <%v>", tt.wantErr, gotErr)
			}
		})
	}
}

func Test_Buffer_CloseAndWait_underlying_reader_closed(t *testing.T) {
	type testcase struct {
		name string // description of this test case

		init func(t *testing.T) (b *Buffer, err error)

		setup func(t *testing.T, b *Buffer)

		wantErr error
	}

	tests := []testcase{
		{
			name: "memory",
			init: func(t *testing.T) (b *Buffer, err error) {
				t.Helper()
				return NewMemory(), nil
			},
			setup: func(t *testing.T, b *Buffer) {
				var err error

				t.Helper()

				if err = b.r.Close(); err != nil {
					t.Fatalf("setup CloseReader() unexpected error: %v", err)
				}
			},
			wantErr: ErrIsClosed,
		},
		{
			name: "file",
			init: func(t *testing.T) (b *Buffer, err error) {
				var f *os.File

				t.Helper()

				if f, err = os.CreateTemp(t.TempDir(), "buffer-close-and-wait-reader-closed-*"); err != nil {
					return nil, err
				}

				if err = f.Close(); err != nil {
					return nil, err
				}

				if b, err = New(f.Name()); err != nil {
					return nil, err
				}

				return b, nil
			},
			setup: func(t *testing.T, b *Buffer) {
				var err error

				t.Helper()

				if err = b.r.Close(); err != nil {
					t.Fatalf("setup CloseReader() unexpected error: %v", err)
				}
			},
			wantErr: ErrIsClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				b      *Buffer
				err    error
				gotErr error
			)

			if b, err = tt.init(t); err != nil {
				t.Fatal(err)
			}

			if tt.setup != nil {
				tt.setup(t, b)
			}

			gotErr = b.CloseAndWait(context.Background())
			if !isEqualErrors(gotErr, tt.wantErr) {
				t.Fatalf("CloseAndWait() invalid error, expected <%v> and received <%v>", tt.wantErr, gotErr)
			}
		})
	}
}
