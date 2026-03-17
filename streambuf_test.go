package streambuf

import (
	"context"
	"io"
	"log"
)

var exampleBuffer *Buffer
var exampleStream *Stream

func ExampleNew() {
	var err error
	if exampleBuffer, err = New("path/to/file"); err != nil {
		log.Fatal(err)
	}
}

func ExampleNewStream() {
	var err error
	// NewStream constructs a read-only file-backed stream.
	if exampleStream, err = NewStream("path/to/file"); err != nil {
		log.Fatal(err)
	}
}

func ExampleNewMemory() {
	exampleBuffer = NewMemory()
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

func ExampleStream_Reader() {
	var (
		r1  io.ReadSeekCloser
		r2  io.ReadSeekCloser
		r3  io.ReadSeekCloser
		err error
	)

	if r1, err = exampleStream.Reader(); err != nil {
		log.Fatal(err)
	}
	defer r1.Close()

	if r2, err = exampleStream.Reader(); err != nil {
		log.Fatal(err)
	}
	defer r2.Close()

	if r3, err = exampleStream.Reader(); err != nil {
		log.Fatal(err)
	}
	defer r3.Close()

	// Each reader is independent and maintains its own read offset.
	// Reads or seeks on r1 do not affect r2 or r3.
}

func ExampleStream_Close() {
	// Close closes the readable backend immediately and does not wait for readers.
	if err := exampleStream.Close(); err != nil {
		log.Fatal(err)
	}
}

func ExampleStream_CloseAndWait() {
	// CloseAndWait blocks until the readable backend is closed and all readers are
	// closed, or until the provided context is done.
	if err := exampleStream.CloseAndWait(context.Background()); err != nil {
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
