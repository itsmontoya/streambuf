// Package streambuf provides append-only buffers and read-only streams with
// independent readers that track their own offsets.
//
// Reader returns EOF when the current end is reached. Buffer also provides
// StreamingReader for reader behavior that waits for future writes.
package streambuf

import (
	"context"
	"errors"
)

var (
	// ErrSeekEndNotSupported is returned when seeking relative to the end.
	// Reader-backed seeks currently support only SeekStart and SeekCurrent.
	ErrSeekEndNotSupported = errors.New("seek end is not currently supported")
	// ErrInvalidWhence is returned when Seek receives an unsupported whence value.
	ErrInvalidWhence = errors.New("invalid seek whence")
	// ErrNegativeIndex is returned when a seek would move before byte index 0.
	// The reader position is clamped to 0 in this case.
	ErrNegativeIndex = errors.New("invalid index, cannot be less than 0")
	// ErrCannotWriteToReadOnly is returned when a write is attempted on a read-only backend.
	ErrCannotWriteToReadOnly = errors.New("cannot write to read-only backend")
	// ErrIsClosed is returned when an action is attempted on a closed instance.
	ErrIsClosed = errors.New("cannot perform action on closed instance")
)

var expiredContext context.Context

func init() {
	var cancel func()
	expiredContext, cancel = context.WithCancel(context.Background())
	cancel()
}
