package streambuf

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func Test_reader_Read(t *testing.T) {
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

				if f, err = os.CreateTemp(t.TempDir(), "reader-read-*"); err != nil {
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
			name: "read only memory with preloaded bytes",
			init: func(t *testing.T) (b *Buffer, err error) {
				t.Helper()

				return NewReadOnlyMemory(append([]byte(nil), testInput...)), nil
			},
		},
		{
			name: "read only file with preloaded bytes",
			init: func(t *testing.T) (b *Buffer, err error) {
				var f *os.File

				t.Helper()

				if f, err = os.CreateTemp(t.TempDir(), "reader-read-only-*"); err != nil {
					return nil, err
				}

				if _, err = f.Write(testInput); err != nil {
					_ = f.Close()
					return nil, err
				}

				if err = f.Close(); err != nil {
					return nil, err
				}

				if b, err = NewReadOnly(f.Name()); err != nil {
					return nil, err
				}

				t.Cleanup(func() {
					_ = b.Close()
				})

				return b, nil
			},
		},
		{
			name: "closed buffer before read",
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
				b   *Buffer
				err error
			)

			if b, err = tt.init(t); err != nil {
				t.Fatal(err)
			}

			b.Write(testInput)

			r := newReader(b)
			bs := make([]byte, len(testInput))
			gotN, gotErr := r.Read(bs)
			if !isEqualErrors(gotErr, tt.wantErr) {
				t.Fatalf("Read() invalid error, expected <%v> and received <%v>", tt.wantErr, gotErr)
			}

			if gotErr != nil {
				return
			}

			if gotN != len(testInput) {
				t.Fatalf("Read() invalid n, expected <%v> and received <%v>", len(testInput), gotN)
			}

			if !bytes.Equal(bs, testInput) {
				t.Fatalf("Read() invalid read value, expected <%v> and received <%v>", string(testInput), string(bs))
			}
		})
	}
}

func Test_reader_Read_zero_length_input(t *testing.T) {
	var (
		b      *Buffer
		r      *reader
		bs     []byte
		gotN   int
		gotErr error
	)

	// Zero-length reads return before interacting with the backend, so this
	// behavior is backend-independent and only needs one simple backend setup.
	b = NewMemory()
	r = newReader(b)
	bs = make([]byte, 0)
	gotN, gotErr = r.Read(bs)

	if gotErr != nil {
		t.Fatalf("Read() invalid error, expected <nil> and received <%v>", gotErr)
	}

	if gotN != 0 {
		t.Fatalf("Read() invalid n, expected <0> and received <%v>", gotN)
	}
}

func Test_reader_Read_closed_reader_while_waiting(t *testing.T) {
	type readResult struct {
		n   int
		err error
	}

	type testcase struct {
		name string // description of this test case

		init func(t *testing.T) (b *Buffer, err error)

		wantErr error
	}

	tests := []testcase{
		{
			name: "memory",
			init: func(t *testing.T) (b *Buffer, err error) {
				t.Helper()
				return NewMemory(), nil
			},
			wantErr: ErrIsClosed,
		},
		{
			name: "file",
			init: func(t *testing.T) (b *Buffer, err error) {
				var f *os.File

				t.Helper()

				if f, err = os.CreateTemp(t.TempDir(), "reader-read-close-wait-*"); err != nil {
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
			wantErr: ErrIsClosed,
		},
	}

	// Read-only backends are not included here because an empty read on them
	// returns ErrIsClosed from the backend directly instead of blocking in Read().
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				b       *Buffer
				err     error
				r       *reader
				bs      []byte
				results chan readResult
				got     readResult
			)

			if b, err = tt.init(t); err != nil {
				t.Fatal(err)
			}

			r = newReader(b)
			b.wg.Add(1)

			bs = make([]byte, 1)
			results = make(chan readResult, 1)

			go func() {
				var out readResult
				out.n, out.err = r.Read(bs)
				results <- out
			}()

			if err = r.Close(); err != nil {
				t.Fatalf("Close() unexpected error: %v", err)
			}

			got = <-results
			if !isEqualErrors(got.err, tt.wantErr) {
				t.Fatalf("Read() invalid error, expected <%v> and received <%v>", tt.wantErr, got.err)
			}

			if got.n != 0 {
				t.Fatalf("Read() invalid n, expected <0> and received <%v>", got.n)
			}
		})
	}
}

func Test_reader_Seek(t *testing.T) {
	type testcase struct {
		name string // description of this test case

		init func(t *testing.T) (b *Buffer, err error)

		setup func(t *testing.T, r *reader)

		offset int64
		whence int

		wantPos int64
		wantErr error
	}

	tests := []testcase{
		{
			name: "memory seek start",
			init: func(t *testing.T) (b *Buffer, err error) {
				t.Helper()
				return NewMemory(), nil
			},
			offset:  5,
			whence:  io.SeekStart,
			wantPos: 5,
		},
		{
			name: "file seek current forward",
			init: func(t *testing.T) (b *Buffer, err error) {
				var f *os.File

				t.Helper()

				if f, err = os.CreateTemp(t.TempDir(), "reader-seek-*"); err != nil {
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
			setup: func(t *testing.T, r *reader) {
				var (
					pos int64
					err error
				)

				t.Helper()

				if pos, err = r.Seek(2, io.SeekStart); err != nil {
					t.Fatalf("setup Seek() unexpected error: %v", err)
				}

				if pos != 2 {
					t.Fatalf("setup Seek() invalid pos, expected <2> and received <%v>", pos)
				}
			},
			offset:  3,
			whence:  io.SeekCurrent,
			wantPos: 5,
		},
		{
			name: "read only memory seek current backward clamped to zero",
			init: func(t *testing.T) (b *Buffer, err error) {
				t.Helper()
				return NewReadOnlyMemory(nil), nil
			},
			setup: func(t *testing.T, r *reader) {
				var (
					pos int64
					err error
				)

				t.Helper()

				if pos, err = r.Seek(3, io.SeekStart); err != nil {
					t.Fatalf("setup Seek() unexpected error: %v", err)
				}

				if pos != 3 {
					t.Fatalf("setup Seek() invalid pos, expected <3> and received <%v>", pos)
				}
			},
			offset:  -10,
			whence:  io.SeekCurrent,
			wantPos: 0,
			wantErr: ErrNegativeIndex,
		},
		{
			name: "read only file seek start negative clamped to zero",
			init: func(t *testing.T) (b *Buffer, err error) {
				var f *os.File

				t.Helper()

				if f, err = os.CreateTemp(t.TempDir(), "reader-seek-read-only-*"); err != nil {
					return nil, err
				}

				if err = f.Close(); err != nil {
					return nil, err
				}

				if b, err = NewReadOnly(f.Name()); err != nil {
					return nil, err
				}

				t.Cleanup(func() {
					_ = b.Close()
				})

				return b, nil
			},
			offset:  -1,
			whence:  io.SeekStart,
			wantPos: 0,
			wantErr: ErrNegativeIndex,
		},
		{
			name: "seek end not supported",
			init: func(t *testing.T) (b *Buffer, err error) {
				t.Helper()
				return NewMemory(), nil
			},
			offset:  0,
			whence:  io.SeekEnd,
			wantPos: 0,
			wantErr: ErrSeekEndNotSupported,
		},
		{
			name: "invalid whence",
			init: func(t *testing.T) (b *Buffer, err error) {
				t.Helper()
				return NewMemory(), nil
			},
			offset:  0,
			whence:  999,
			wantPos: 0,
			wantErr: ErrInvalidWhence,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				b      *Buffer
				err    error
				r      *reader
				gotPos int64
				gotErr error
			)

			if b, err = tt.init(t); err != nil {
				t.Fatal(err)
			}

			r = newReader(b)
			if tt.setup != nil {
				tt.setup(t, r)
			}

			gotPos, gotErr = r.Seek(tt.offset, tt.whence)
			if !isEqualErrors(gotErr, tt.wantErr) {
				t.Fatalf("Seek() invalid error, expected <%v> and received <%v>", tt.wantErr, gotErr)
			}

			if gotPos != tt.wantPos {
				t.Fatalf("Seek() invalid pos, expected <%v> and received <%v>", tt.wantPos, gotPos)
			}
		})
	}
}

func Test_reader_Close(t *testing.T) {
	type testcase struct {
		name string // description of this test case

		init func(t *testing.T) (b *Buffer, err error)

		setup func(t *testing.T, r *reader)

		wantErr error
	}

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

				if f, err = os.CreateTemp(t.TempDir(), "reader-close-*"); err != nil {
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
			name: "read only memory",
			init: func(t *testing.T) (b *Buffer, err error) {
				t.Helper()
				return NewReadOnlyMemory(nil), nil
			},
		},
		{
			name: "read only file",
			init: func(t *testing.T) (b *Buffer, err error) {
				var f *os.File

				t.Helper()

				if f, err = os.CreateTemp(t.TempDir(), "reader-close-read-only-*"); err != nil {
					return nil, err
				}

				if err = f.Close(); err != nil {
					return nil, err
				}

				if b, err = NewReadOnly(f.Name()); err != nil {
					return nil, err
				}

				t.Cleanup(func() {
					_ = b.Close()
				})

				return b, nil
			},
		},
		{
			name: "already closed reader",
			init: func(t *testing.T) (b *Buffer, err error) {
				t.Helper()
				return NewMemory(), nil
			},
			setup: func(t *testing.T, r *reader) {
				var err error

				t.Helper()

				if err = r.Close(); err != nil {
					t.Fatalf("setup Close() unexpected error: %v", err)
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
				r      *reader
				gotErr error
			)

			if b, err = tt.init(t); err != nil {
				t.Fatal(err)
			}

			r = newReader(b)
			b.wg.Add(1)

			if tt.setup != nil {
				tt.setup(t, r)
			}

			gotErr = r.Close()
			if !isEqualErrors(gotErr, tt.wantErr) {
				t.Fatalf("Close() invalid error, expected <%v> and received <%v>", tt.wantErr, gotErr)
			}
		})
	}
}
