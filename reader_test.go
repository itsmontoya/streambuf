package streambuf

import (
	"bytes"
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
