package streambuf

import (
	"bytes"
	"os"
	"testing"
)

func Test_New_invalid_filepath(t *testing.T) {
	type testcase struct {
		name string // description of this test case

		filepath string

		wantErr bool
	}

	tests := []testcase{
		{
			name:     "missing parent directory",
			filepath: t.TempDir() + "/missing-dir/buffer.tmp",
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
				b   *Buffer
				err error
			)

			if b, err = New(tt.filepath); err != nil {
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

func Test_NewReadOnly_invalid_filepath(t *testing.T) {
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
				b   *Buffer
				err error
			)

			if b, err = NewReadOnly(tt.filepath); err != nil {
				if !tt.wantErr {
					t.Fatalf("NewReadOnly() unexpected error: %v", err)
				}

				return
			}

			if tt.wantErr {
				t.Fatal("NewReadOnly() expected error, received <nil>")
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
			name: "read only memory",
			init: func(t *testing.T) (b *Buffer, err error) {
				t.Helper()
				return NewReadOnlyMemory(nil), nil
			},
			wantErr: ErrCannotWriteToReadOnly,
		},
		{
			name: "read only file",
			init: func(t *testing.T) (b *Buffer, err error) {
				var f *os.File

				t.Helper()

				if f, err = os.CreateTemp(t.TempDir(), "buffer-write-read-only-*"); err != nil {
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
			wantErr: ErrCannotWriteToReadOnly,
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

			r = newReader(b)
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
