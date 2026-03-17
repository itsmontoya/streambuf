package streambuf

import (
	"errors"
	"os"
	"testing"
)

func Test_readableFile_Close_underlying_file_already_closed(t *testing.T) {
	var (
		f      *os.File
		r      *readableFile
		err    error
		gotErr error
	)

	if f, err = os.CreateTemp(t.TempDir(), "readable-file-close-*"); err != nil {
		t.Fatal(err)
	}

	if err = f.Close(); err != nil {
		t.Fatal(err)
	}

	if r, err = newReadableFile(f.Name()); err != nil {
		t.Fatal(err)
	}

	if err = r.f.Close(); err != nil {
		t.Fatalf("setup Close() unexpected error: %v", err)
	}

	gotErr = r.Close()
	if gotErr == nil {
		t.Fatal("Close() expected error, received <nil>")
	}

	if !errors.Is(gotErr, os.ErrClosed) {
		t.Fatalf("Close() invalid error, expected wrapped <%v> and received <%v>", os.ErrClosed, gotErr)
	}
}
