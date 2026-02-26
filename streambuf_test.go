package streambuf

import (
	"context"
	"io"
	"log"
)

var exampleBuffer *Buffer

func ExampleNew() {
	var err error
	if exampleBuffer, err = New("path/to/file"); err != nil {
		log.Fatal(err)
	}
}

func ExampleNewReadOnly() {
	var err error
	if exampleBuffer, err = NewReadOnly("path/to/file"); err != nil {
		log.Fatal(err)
	}
}

func ExampleNewMemory() {
	exampleBuffer = NewMemory()
}

func ExampleNewReadOnlyMemory() {
	exampleBuffer = NewReadOnlyMemory([]byte("hello world"))
}

func ExampleBuffer_Write() {
	if _, err := exampleBuffer.Write([]byte("hello world")); err != nil {
		log.Fatal(err)
	}
}

func ExampleBuffer_Reader() {
	var err error
	if _, err = exampleBuffer.Write([]byte("hello world")); err != nil {
		log.Fatal(err)
	}

	var (
		r1 io.ReadSeekCloser
		r2 io.ReadSeekCloser
		r3 io.ReadSeekCloser
	)

	if r1, err = exampleBuffer.Reader(); err != nil {
		log.Fatal(err)
	}
	defer r1.Close()

	if r2, err = exampleBuffer.Reader(); err != nil {
		log.Fatal(err)
	}
	defer r2.Close()

	if r3, err = exampleBuffer.Reader(); err != nil {
		log.Fatal(err)
	}
	defer r3.Close()

	// Each reader is independent and maintains its own read offset.
	// Reads or seeks on r1 do not affect r2 or r3.
}

func ExampleBuffer_Close() {
	// Close closes the backend immediately and does not wait for readers to finish.
	if err := exampleBuffer.Close(); err != nil {
		log.Fatal(err)
	}
}

func ExampleBuffer_CloseAndWait() {
	// CloseAndWait blocks until the backend is closed and all readers are closed,
	// or until the provided context is done.
	if err := exampleBuffer.CloseAndWait(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func isEqualErrors(a, b error) (isEqual bool) {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil && b != nil:
		return false
	case a != nil && b == nil:
		return false
	default:
		return a.Error() == b.Error()
	}
}
