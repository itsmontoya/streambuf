# streambuf &emsp; [![GoDoc][GoDoc Badge]][GoDoc URL] ![Coverage] [![Go Report Card][Report Card Badge]][Report Card URL] [![MIT licensed][License Badge]][License URL]

[GoDoc Badge]: https://pkg.go.dev/badge/github.com/itsmontoya/streambuf
[GoDoc URL]: https://pkg.go.dev/github.com/itsmontoya/streambuf
[Coverage]: https://img.shields.io/badge/coverage-100%25-brightgreen
[License Badge]: https://img.shields.io/badge/license-MIT-blue.svg
[License URL]: https://github.com/itsmontoya/streambuf/blob/main/LICENSE
[Report Card Badge]: https://goreportcard.com/badge/github.com/itsmontoya/streambuf
[Report Card URL]: https://goreportcard.com/report/github.com/itsmontoya/streambuf

![banner](https://res.cloudinary.com/dryepxxoy/image/upload/v1771535319/streambuf_banner_with_name_1920_xvev2l.webp "Streambuf banner")

`streambuf` is a Go library that provides an **append-only buffer with multiple independent readers**.

It allows a single writer to continuously append bytes to a buffer, while any number of readers consume the data at their own pace, without interfering with each other.

The buffer can be backed by memory or by a file, making it suitable for both lightweight in-memory streaming and durable, disk-backed use cases.

## Motivation

Go’s standard library provides excellent primitives for streaming (`io.Reader`, `io.Writer`, `bufio`, channels), but it lacks a native abstraction for:

- Append-only data
- Multiple independent readers
- Late-joining readers
- Sequential, ordered reads
- Optional file-backed persistence

`streambuf` fills this gap by behaving like a shared, growing stream where readers maintain their own cursor.

This pattern shows up frequently in systems programming, including:

- Chat and messaging services
- Log streaming
- Fan-out pipelines
- Event feeds
- Streaming ingestion systems
- Testing and replay of streamed data

## Examples

Below are quick API examples. For runnable end-to-end examples, see `examples/`.

### New
```go
func ExampleNew() {
	var err error
	if exampleBuffer, err = New("path/to/file"); err != nil {
		log.Fatal(err)
	}
}
```

### NewStream
```go
func ExampleNewStream() {
	var err error
	// NewStream constructs a read-only file-backed stream.
	if exampleStream, err = NewStream("path/to/file"); err != nil {
		log.Fatal(err)
	}
}
```

### NewMemory
```go
func ExampleNewMemory() {
	exampleBuffer = NewMemory()
}
```

### NewMemoryStream
```go
func ExampleNewMemoryStream() {
	bs := []byte("hello world")
	exampleStream = NewMemoryStream(bs)
}
```

### Buffer.Write
```go
func ExampleBuffer_Write() {
	if _, err := exampleBuffer.Write([]byte("hello world")); err != nil {
		log.Fatal(err)
	}
}
```

### Buffer.Reader
```go
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
```

### Buffer.Close
```go
func ExampleBuffer_Close() {
	// Close closes the backend immediately and does not wait for readers to finish.
	if err := exampleBuffer.Close(); err != nil {
		log.Fatal(err)
	}
}
```

### Buffer.CloseAndWait
```go
func ExampleBuffer_CloseAndWait() {
	// CloseAndWait blocks until the backend is closed and all readers are closed,
	// or until the provided context is done.
	if err := exampleBuffer.CloseAndWait(context.Background()); err != nil {
		log.Fatal(err)
	}
}
```

### Stream.Reader
```go
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
```

### Stream.Close
```go
func ExampleStream_Close() {
	// Close closes the readable backend immediately and does not wait for readers.
	if err := exampleStream.Close(); err != nil {
		log.Fatal(err)
	}
}
```

### Stream.CloseAndWait
```go
func ExampleStream_CloseAndWait() {
	// CloseAndWait blocks until the readable backend is closed and all readers are
	// closed, or until the provided context is done.
	if err := exampleStream.CloseAndWait(context.Background()); err != nil {
		log.Fatal(err)
	}
}
```

## Core Concepts

### Append-only buffer

Data is written once and never modified in place.

Writes always append to the end of the buffer.

### Independent readers

Each reader maintains its own read position. Readers do not block or consume data from each other.

Readers may:

- Start from the beginning
- Start from the current end
- Join after data has already been written

### Blocking reads

Readers block when no data is available and resume automatically when new data is appended.

For streams (`NewStream`), this means reaching the current end of the readable
file will also block until the stream is closed or the reader is closed.

If you are treating a stream as a finite snapshot, call `Close()` (or
`CloseAndWait(...)`) on the stream after readers finish consuming data, or close
the reader directly, to unblock waiting reads and complete shutdown cleanly.

### Shutdown behavior

- `Close()` closes immediately. Existing unread bytes may no longer be available to readers.
- `CloseAndWait(ctx)` closes writes and waits for readers until `ctx` is canceled.
- `ctx` can be a timeout/deadline context to bound how long shutdown waits.
- Terminal reads after either buffer close or reader close return `ErrIsClosed`.
- To preserve reader drain behavior, finish reading first, then call `CloseAndWait` (or coordinate with reader `Close` calls and context cancellation).
- If `ctx` is canceled before readers close, `CloseAndWait` still returns and the buffer stays closed; close outstanding readers afterward to finish internal wait cleanup.

### Pluggable storage

`streambuf` supports multiple backing implementations:

- **Memory-backed** (`[]byte`)
- **File-backed** (using a shared file descriptor)
- **Read-only file-backed stream** (existing file opened read-only)

`Buffer` and `Stream` share the same reader behavior. `Buffer` adds `Write`,
while `Stream` is read-only.

## AI Usage and Authorship

This project is intentionally **human-authored** for all logic.

To be explicit:

- AI does **not** write or modify non-test code in this repository.
- AI does **not** make architectural or behavioral decisions.
- AI may assist with documentation, comments, and test scaffolding only.
- All implementation logic is written and reviewed by human maintainers.

These boundaries are enforced in `AGENTS.md` and are part of this repository's contribution discipline.

## Contributors

- Human maintainers: library design, implementation, and behavior decisions.
- ChatGPT Codex: documentation, test coverage support, and comments.
- Google Gemini: README artwork generation.

![banner](https://res.cloudinary.com/dryepxxoy/image/upload/v1771535291/streambuf_footer_1920_qhttyv.webp "Streambuf footer")
