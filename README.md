# streambuf &emsp; ![Coverage] [![Go Report Card][Report Card Badge]][Report Card URL] [![MIT licensed][License Badge]][License URL]

[Coverage]: https://img.shields.io/badge/coverage-100%25-brightgreen
[License Badge]: https://img.shields.io/badge/license-MIT-blue.svg
[License URL]: https://github.com/itsmontoya/streambuf/blob/main/LICENSE
[Report Card Badge]: https://goreportcard.com/badge/github.com/itsmontoya/streambuf
[Report Card URL]: https://goreportcard.com/report/github.com/itsmontoya/streambuf

![banner](https://github.com/itsmontoya/streambuf/blob/main/banner_1920.webp?raw=true "Streambuf banner")

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

### Pluggable storage

`streambuf` supports multiple backing implementations:

- **Memory-backed** (`[]byte`)
- **File-backed** (using a shared file descriptor)

Both implementations expose the same behavior and API.

## Example

```go
buf := streambuf.NewMemory()

r1 := buf.NewReader()
r2 := buf.NewReader()

go func() {
    buf.Append([]byte("hello\n"))
    buf.Append([]byte("world\n"))
}()

io.Copy(os.Stdout, r1)
io.Copy(os.Stdout, r2)
