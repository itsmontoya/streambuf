package streambuf

import "testing"

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
