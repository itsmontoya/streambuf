# streambuf &emsp; [![GoDoc][GoDoc Badge]][GoDoc URL] ![Coverage] [![Go Report Card][Report Card Badge]][Report Card URL] [![MIT licensed][License Badge]][License URL]

[GoDoc Badge]: https://godoc.org/github.com/itsmontoya/streambuf?status.svg
[GoDoc URL]: https://godoc.org/github.com/itsmontoya/streambuf
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

## Examples

### File-backed buffer with independent readers (`New(filepath)`)

```go
package main

import (
	"fmt"
	"io"
	"log"
	"os"

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

	fast := buf.Reader()
	defer fast.Close()

	slow := buf.Reader()
	defer slow.Close()

	if _, err = buf.Write([]byte("hello ")); err != nil {
		log.Fatal(err)
	}

	fastFirst := make([]byte, len("hello "))
	if _, err = io.ReadFull(fast, fastFirst); err != nil {
		log.Fatal(err)
	}

	if _, err = buf.Write([]byte("file backend")); err != nil {
		log.Fatal(err)
	}

	fastRest := make([]byte, len("file backend"))
	if _, err = io.ReadFull(fast, fastRest); err != nil {
		log.Fatal(err)
	}

	slowAll := make([]byte, len("hello file backend"))
	if _, err = io.ReadFull(slow, slowAll); err != nil {
		log.Fatal(err)
	}

	if err = buf.Close(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("fast reader: %s%s\n", string(fastFirst), string(fastRest))
	fmt.Printf("slow reader: %s\n", string(slowAll))
}
```

### In-memory buffer with late-joining reader (`NewMemory()`)

```go
package main

import (
	"fmt"
	"io"
	"log"

	"github.com/itsmontoya/streambuf"
)

func main() {
	var err error
	buf := streambuf.NewMemory()
	early := buf.Reader()
	defer early.Close()

	if _, err = buf.Write([]byte("frame-1|")); err != nil {
		log.Fatal(err)
	}

	// This reader joins after data already exists but still starts at index 0.
	late := buf.Reader()
	defer late.Close()

	if _, err = buf.Write([]byte("frame-2")); err != nil {
		log.Fatal(err)
	}

	earlyAll := make([]byte, len("frame-1|frame-2"))
	if _, err = io.ReadFull(early, earlyAll); err != nil {
		log.Fatal(err)
	}

	lateAll := make([]byte, len("frame-1|frame-2"))
	if _, err = io.ReadFull(late, lateAll); err != nil {
		log.Fatal(err)
	}

	if err = buf.Close(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("early reader: %s\n", string(earlyAll))
	fmt.Printf("late reader:  %s\n", string(lateAll))
}
```

### Graceful close with timeout (`CloseAndWait(cancel)`)

```go
package main

import (
	"io"
	"log"
	"time"

	"github.com/itsmontoya/streambuf"
)

func main() {
	var err error
	buf := streambuf.NewMemory()
	r := buf.Reader()

	if _, err = buf.Write([]byte("payload")); err != nil {
		log.Fatal(err)
	}

	// Let this reader drain and then close itself.
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		in := make([]byte, len("payload"))
		if _, readErr := io.ReadFull(r, in); readErr != nil {
			return
		}
		_ = r.Close()
	}()

	// Cancel waiting after 2 seconds if readers have not closed.
	cancel := make(chan struct{})
	go func() {
		time.Sleep(2 * time.Second)
		close(cancel)
	}()

	if err = buf.CloseAndWait(cancel); err != nil {
		log.Fatal(err)
	}

	<-readerDone
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

### Shutdown behavior

- `Close()` closes immediately. Existing unread bytes may no longer be available to readers.
- `CloseAndWait(cancel)` closes writes and waits for readers only when `cancel` is non-nil and not yet closed.
- `cancel` can be a timeout channel you close after a deadline to bound how long shutdown waits.
- To preserve reader drain behavior, finish reading first, then call `CloseAndWait` (or coordinate with reader `Close` calls and a cancel channel).

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
