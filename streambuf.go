// Package streambuf provides a concurrent buffer with independent readers that
// can block until more data is written or the buffer is closed.
package streambuf

import "context"

var expiredContext context.Context

func init() {
	var cancel func()
	expiredContext, cancel = context.WithCancel(context.Background())
	cancel()
}
