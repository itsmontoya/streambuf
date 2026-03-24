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

In practice, this is useful when you want more than "a mutex around an `io.Writer`". A plain writer gives you serialized writes, but it does not give each consumer its own read cursor, late-joining readers, optional follow-style blocking reads, or a shared file-backed stream that avoids opening separate descriptors per reader.

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

## Why This Exists

It is reasonable to ask: why not just protect an `io.Writer` with a mutex?

That solves a different problem.

A mutex around a writer helps multiple goroutines write safely, but it does not provide:

- Independent readers with their own offsets
- Readers that can join after data has already been written
- Optional reads that block until more data arrives
- A single shared file-backed source for many readers

`streambuf` is for cases where one side is continuously appending data and many readers need to observe the same ordered byte stream without consuming it from each other.

## Example Use Case

Imagine a service that receives a live byte stream from one upstream connection and needs to expose it to several downstream consumers:

- One consumer forwards the stream to an HTTP client
- One consumer writes it to disk for later replay
- One consumer parses it for metrics or events

With a normal `io.Writer`, you still need to build the fan-out, track read positions, and coordinate EOF-vs-follow reader behavior yourself.

With `streambuf`, the producer writes once, each consumer gets its own reader, and a file-backed buffer can keep everything on a single shared file descriptor instead of opening one per consumer.

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
	// Reader returns EOF when the current end is reached.
}
```

### Buffer.StreamingReader
```go
func ExampleBuffer_StreamingReader() {
	var (
		r   io.ReadSeekCloser
		err error
	)

	if r, err = exampleBuffer.StreamingReader(); err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	// StreamingReader waits for future writes when no bytes are currently
	// available at the reader offset.
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
	// Reader returns EOF when the current end is reached.
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

### Reader modes

- `Reader()` returns EOF when the current end is reached.
- `StreamingReader()` (Buffer only) waits for future writes when the current end is reached.

Use `Reader()` for finite/snapshot-style consumption and `StreamingReader()` for follow/tail-style consumption.

### Shutdown behavior

- `Close()` closes immediately. Existing unread bytes may no longer be available to readers.
- `CloseAndWait(ctx)` closes writes and waits for readers until `ctx` is canceled.
- `ctx` can be a timeout/deadline context to bound how long shutdown waits.
- For `StreamingReader()`, terminal reads after buffer close or reader close return `ErrIsClosed`.
- For `Reader()`, reaching the current end returns EOF.
- To preserve reader drain behavior, finish reading first, then call `CloseAndWait` (or coordinate with reader `Close` calls and context cancellation).
- If `ctx` is canceled before readers close, `CloseAndWait` still returns and the buffer stays closed; close outstanding readers afterward to finish internal wait cleanup.

### Pluggable storage

`streambuf` supports multiple backing implementations:

- **Memory-backed** (`[]byte`)
- **File-backed** (using a shared file descriptor)
- **Read-only file-backed stream** (existing file opened read-only)

`Buffer` and `Stream` both expose `Reader()` with EOF-at-end semantics. `Buffer`
adds `Write` and `StreamingReader()` for follow-style reads, while `Stream` is read-only.

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
