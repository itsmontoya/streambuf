package streambuf

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"
)

func Test_NewStream_invalid_filepath(t *testing.T) {
	type testcase struct {
		name string // description of this test case

		filepath string

		wantErr bool
	}

	tests := []testcase{
		{
			name:     "missing file",
			filepath: t.TempDir() + "/missing-file.tmp",
			wantErr:  true,
		},
		{
			name:     "empty filepath",
			filepath: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				s   *Stream
				err error
			)

			if s, err = NewStream(tt.filepath); err != nil {
				if !tt.wantErr {
					t.Fatalf("NewStream() unexpected error: %v", err)
				}

				return
			}

			if tt.wantErr {
				t.Fatal("NewStream() expected error, received <nil>")
			}

			t.Cleanup(func() {
				_ = s.Close()
			})
		})
	}
}

func Test_Stream_Reader(t *testing.T) {
	type testcase struct {
		name string // description of this test case

		init func(t *testing.T) (s *Stream, err error)

		wantErr error
	}

	testInput := []byte("This is our test input!")
	tests := []testcase{
		{
			name: "memory with preloaded bytes",
			init: func(t *testing.T) (s *Stream, err error) {
				t.Helper()
				return NewMemoryStream(testInput), nil
			},
		},
		{
			name: "file with preloaded bytes",
			init: func(t *testing.T) (s *Stream, err error) {
				t.Helper()
				return newTestFileStream(t, "stream-reader-*", testInput)
			},
		},
		{
			name: "closed stream",
			init: func(t *testing.T) (s *Stream, err error) {
				t.Helper()

				s = NewMemoryStream(nil)
				if err = s.Close(); err != nil {
					return nil, err
				}

				return s, nil
			},
			wantErr: ErrIsClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				s      *Stream
				err    error
				r      io.ReadSeekCloser
				bs     []byte
				gotN   int
				gotErr error
			)

			if s, err = tt.init(t); err != nil {
				t.Fatal(err)
			}

			r, gotErr = s.Reader()
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

func Test_Stream_Reader_no_more_bytes_and_stream_closed(t *testing.T) {
	type testcase struct {
		name string // description of this test case

		init func(t *testing.T) (s *Stream, err error)

		wantErr error
	}

	testInput := []byte("This is our test input!")
	tests := []testcase{
		{
			name: "memory with preloaded bytes",
			init: func(t *testing.T) (s *Stream, err error) {
				t.Helper()
				return NewMemoryStream(testInput), nil
			},
			wantErr: ErrIsClosed,
		},
		{
			name: "file with preloaded bytes",
			init: func(t *testing.T) (s *Stream, err error) {
				t.Helper()
				return newTestFileStream(t, "stream-reader-eof-*", testInput)
			},
			wantErr: ErrIsClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				s          *Stream
				err        error
				r          io.ReadSeekCloser
				bs         []byte
				gotN       int
				gotErr     error
				secondRead []byte
			)

			if s, err = tt.init(t); err != nil {
				t.Fatal(err)
			}

			if r, err = s.Reader(); err != nil {
				t.Fatalf("Reader() unexpected error: %v", err)
			}

			t.Cleanup(func() {
				_ = r.Close()
			})

			bs = make([]byte, len(testInput))
			gotN, gotErr = r.Read(bs)
			if gotErr != nil {
				t.Fatalf("Read() unexpected error while consuming bytes: %v", gotErr)
			}

			if gotN != len(testInput) {
				t.Fatalf("Read() invalid n while consuming bytes, expected <%v> and received <%v>", len(testInput), gotN)
			}

			if !bytes.Equal(bs, testInput) {
				t.Fatalf("Read() invalid read value while consuming bytes, expected <%v> and received <%v>", string(testInput), string(bs))
			}

			if err = s.Close(); err != nil {
				t.Fatalf("Close() unexpected error: %v", err)
			}

			secondRead = make([]byte, 1)
			gotN, gotErr = r.Read(secondRead)
			if !isEqualErrors(gotErr, tt.wantErr) {
				t.Fatalf("Read() invalid error after stream close, expected <%v> and received <%v>", tt.wantErr, gotErr)
			}

			if gotN != 0 {
				t.Fatalf("Read() invalid n after stream close, expected <0> and received <%v>", gotN)
			}
		})
	}
}

func Test_Stream_Reader_no_bytes_available_not_closed(t *testing.T) {
	type testcase struct {
		name string // description of this test case

		init func(t *testing.T) (s *Stream, err error)
	}

	tests := []testcase{
		{
			name: "empty memory stream",
			init: func(t *testing.T) (s *Stream, err error) {
				t.Helper()
				return NewMemoryStream(nil), nil
			},
		},
		{
			name: "empty file stream",
			init: func(t *testing.T) (s *Stream, err error) {
				t.Helper()
				return newTestFileStream(t, "stream-reader-wait-*", nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				s      *Stream
				err    error
				r      io.ReadSeekCloser
				bs     []byte
				gotN   int
				gotErr error
			)

			if s, err = tt.init(t); err != nil {
				t.Fatal(err)
			}

			if r, err = s.Reader(); err != nil {
				t.Fatalf("Reader() unexpected error: %v", err)
			}

			t.Cleanup(func() {
				_ = r.Close()
			})

			bs = make([]byte, 1)
			gotN, gotErr = r.Read(bs)
			if !errors.Is(gotErr, io.EOF) {
				t.Fatalf("Read() invalid error when no bytes are available, expected error wrapping <%v> and received <%v>", io.EOF, gotErr)
			}

			if gotN != 0 {
				t.Fatalf("Read() invalid n when no bytes are available, expected <0> and received <%v>", gotN)
			}
		})
	}
}

func Test_Stream_Reader_seek(t *testing.T) {
	type testcase struct {
		name string // description of this test case

		init func(t *testing.T) (s *Stream, err error)

		setup func(t *testing.T, r io.ReadSeekCloser)

		offset int64
		whence int

		wantPos int64
		wantErr error
	}

	tests := []testcase{
		{
			name: "memory seek current backward clamped to zero",
			init: func(t *testing.T) (s *Stream, err error) {
				t.Helper()
				return NewMemoryStream(nil), nil
			},
			setup: func(t *testing.T, r io.ReadSeekCloser) {
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
			name: "file seek start negative clamped to zero",
			init: func(t *testing.T) (s *Stream, err error) {
				t.Helper()
				return newTestFileStream(t, "stream-seek-*", nil)
			},
			offset:  -1,
			whence:  io.SeekStart,
			wantPos: 0,
			wantErr: ErrNegativeIndex,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				s      *Stream
				err    error
				r      io.ReadSeekCloser
				gotPos int64
				gotErr error
			)

			if s, err = tt.init(t); err != nil {
				t.Fatal(err)
			}

			if r, err = s.Reader(); err != nil {
				t.Fatalf("Reader() unexpected error: %v", err)
			}

			t.Cleanup(func() {
				_ = r.Close()
			})

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

func Test_Stream_Reader_close(t *testing.T) {
	type testcase struct {
		name string // description of this test case

		init func(t *testing.T) (s *Stream, err error)

		setup func(t *testing.T, r io.ReadSeekCloser)

		wantErr error
	}

	tests := []testcase{
		{
			name: "memory",
			init: func(t *testing.T) (s *Stream, err error) {
				t.Helper()
				return NewMemoryStream(nil), nil
			},
		},
		{
			name: "file",
			init: func(t *testing.T) (s *Stream, err error) {
				t.Helper()
				return newTestFileStream(t, "stream-close-*", nil)
			},
		},
		{
			name: "already closed reader",
			init: func(t *testing.T) (s *Stream, err error) {
				t.Helper()
				return NewMemoryStream(nil), nil
			},
			setup: func(t *testing.T, r io.ReadSeekCloser) {
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
				s      *Stream
				err    error
				r      io.ReadSeekCloser
				gotErr error
			)

			if s, err = tt.init(t); err != nil {
				t.Fatal(err)
			}

			if r, err = s.Reader(); err != nil {
				t.Fatalf("Reader() unexpected error: %v", err)
			}

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

func Test_Stream_CloseAndWait(t *testing.T) {
	type testcase struct {
		name string // description of this test case

		init func(t *testing.T) (s *Stream, err error)

		setup func(t *testing.T, s *Stream)

		ctx context.Context

		wantErr error
	}

	tests := []testcase{
		{
			name: "memory",
			init: func(t *testing.T) (s *Stream, err error) {
				t.Helper()
				return NewMemoryStream(nil), nil
			},
			ctx: context.Background(),
		},
		{
			name: "file",
			init: func(t *testing.T) (s *Stream, err error) {
				t.Helper()
				return newTestFileStream(t, "stream-close-and-wait-*", nil)
			},
			ctx: context.Background(),
		},
		{
			name: "closed stream",
			init: func(t *testing.T) (s *Stream, err error) {
				t.Helper()

				s = NewMemoryStream(nil)
				if err = s.Close(); err != nil {
					return nil, err
				}

				return s, nil
			},
			ctx:     context.Background(),
			wantErr: ErrIsClosed,
		},
		{
			name: "closed waiter",
			init: func(t *testing.T) (s *Stream, err error) {
				t.Helper()

				s = NewMemoryStream(nil)
				if err = s.waiter.Close(); err != nil {
					return nil, err
				}

				return s, nil
			},
			ctx:     context.Background(),
			wantErr: ErrIsClosed,
		},
		{
			name: "reader open and canceled context",
			init: func(t *testing.T) (s *Stream, err error) {
				t.Helper()
				return NewMemoryStream(nil), nil
			},
			setup: func(t *testing.T, s *Stream) {
				var (
					r   io.ReadSeekCloser
					err error
				)

				t.Helper()

				if r, err = s.Reader(); err != nil {
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
				s      *Stream
				err    error
				gotErr error
			)

			if s, err = tt.init(t); err != nil {
				t.Fatal(err)
			}

			if tt.setup != nil {
				tt.setup(t, s)
			}

			gotErr = s.CloseAndWait(tt.ctx)
			if !isEqualErrors(gotErr, tt.wantErr) {
				t.Fatalf("CloseAndWait() invalid error, expected <%v> and received <%v>", tt.wantErr, gotErr)
			}
		})
	}
}

func Test_Stream_CloseAndWait_underlying_reader_closed(t *testing.T) {
	type testcase struct {
		name string // description of this test case

		init func(t *testing.T) (s *Stream, err error)

		setup func(t *testing.T, s *Stream)

		wantErr error
	}

	tests := []testcase{
		{
			name: "memory",
			init: func(t *testing.T) (s *Stream, err error) {
				t.Helper()
				return NewMemoryStream(nil), nil
			},
			setup: func(t *testing.T, s *Stream) {
				var err error

				t.Helper()

				if err = s.r.Close(); err != nil {
					t.Fatalf("setup Close() unexpected error: %v", err)
				}
			},
			wantErr: ErrIsClosed,
		},
		{
			name: "file",
			init: func(t *testing.T) (s *Stream, err error) {
				t.Helper()
				return newTestFileStream(t, "stream-close-and-wait-reader-closed-*", nil)
			},
			setup: func(t *testing.T, s *Stream) {
				var err error

				t.Helper()

				if err = s.r.Close(); err != nil {
					t.Fatalf("setup Close() unexpected error: %v", err)
				}
			},
			wantErr: ErrIsClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				s      *Stream
				err    error
				gotErr error
			)

			if s, err = tt.init(t); err != nil {
				t.Fatal(err)
			}

			if tt.setup != nil {
				tt.setup(t, s)
			}

			gotErr = s.CloseAndWait(context.Background())
			if !isEqualErrors(gotErr, tt.wantErr) {
				t.Fatalf("CloseAndWait() invalid error, expected <%v> and received <%v>", tt.wantErr, gotErr)
			}
		})
	}
}

func newTestFileStream(t *testing.T, prefix string, in []byte) (out *Stream, err error) {
	var f *os.File

	t.Helper()

	if f, err = os.CreateTemp(t.TempDir(), prefix); err != nil {
		return nil, err
	}

	if len(in) > 0 {
		if _, err = f.Write(in); err != nil {
			_ = f.Close()
			return nil, err
		}
	}

	if err = f.Close(); err != nil {
		return nil, err
	}

	if out, err = NewStream(f.Name()); err != nil {
		return nil, err
	}

	return out, nil
}
