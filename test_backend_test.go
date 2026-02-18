package streambuf

import (
	"path/filepath"
	"testing"
)

type bufferConstructor struct {
	name string
	new  func(t *testing.T) *Buffer
}

func runForEachBackend(t *testing.T, fn func(t *testing.T, b *Buffer)) {
	t.Helper()

	var tests []bufferConstructor
	tests = []bufferConstructor{
		{
			name: "memory",
			new: func(t *testing.T) (out *Buffer) {
				t.Helper()
				return NewMemory()
			},
		},
		{
			name: "file",
			new: func(t *testing.T) (out *Buffer) {
				t.Helper()

				var path string
				path = filepath.Join(t.TempDir(), "streambuf.test")

				var (
					b   *Buffer
					err error
				)

				if b, err = New(path); err != nil {
					t.Fatalf("New(%q) = %v, want nil", path, err)
				}

				return b
			},
		},
	}

	for _, tt := range tests {
		var tc bufferConstructor
		tc = tt

		t.Run(tc.name, func(t *testing.T) {
			var b *Buffer
			b = tc.new(t)
			fn(t, b)
		})
	}
}
