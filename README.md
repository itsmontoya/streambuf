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

Quick start example:

```go
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/itsmontoya/streambuf"
)

func main() {
	var (
		buf *streambuf.Buffer
		err error
	)

	if buf, err = streambuf.New("./stream.log"); err != nil {
		log.Fatal(err)
	}
	defer os.Remove("./stream.log")

	var fast io.ReadCloser
	if fast, err = buf.Reader(); err != nil {
		log.Fatal(err)
	}

	var slow io.ReadCloser
	if slow, err = buf.Reader(); err != nil {
		log.Fatal(err)
	}

	var fastBS []byte
	go func() {
		fastBS, _ = io.ReadAll(fast)
		defer fast.Close()
	}()

	var slowBS []byte
	go func() {
		time.Sleep(time.Second)
		slowBS, _ = io.ReadAll(slow)
		defer slow.Close()
	}()

	if _, err = buf.Write([]byte("hello file backend")); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if err = buf.CloseAndWait(ctx); err != nil {
		log.Fatal(err)
	}

	// Fast reader has all contents
	fmt.Printf("fast reader: %s\n", string(fastBS))
	// Slow reader has all contents
	fmt.Printf("slow reader: %s\n", string(slowBS))
}
```

Additional runnable examples live in `examples/`:

- `examples/basic/main.go`: demonstrates immediate `Close()` behavior.
- `examples/basic_with_wait/main.go`: demonstrates `CloseAndWait(ctx)` with timeout-based cancellation.

Run them from the repository root:

```bash
go run ./examples/basic
go run ./examples/basic_with_wait
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

Both implementations expose the same behavior and API.

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
