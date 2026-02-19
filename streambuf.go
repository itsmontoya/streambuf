package streambuf

import "context"

var expiredContext context.Context

func init() {
	var cancel func()
	expiredContext, cancel = context.WithCancel(context.Background())
	cancel()
}
