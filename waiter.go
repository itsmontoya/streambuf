package streambuf

import "sync"

func newWaiter() (out *waiter) {
	var w waiter
	w.c = make(chan struct{})
	return &w
}

// waiter coordinates one-to-many notifications by rotating channels on refresh.
type waiter struct {
	mux sync.RWMutex

	c chan struct{}

	closed bool
}

// Wait returns the current notification channel.
func (w *waiter) Wait() (out <-chan struct{}) {
	w.mux.RLock()
	defer w.mux.RUnlock()
	return w.c
}

// Refresh closes the current notification channel and creates a new one.
func (w *waiter) Refresh() {
	w.mux.Lock()
	defer w.mux.Unlock()
	close(w.c)
	w.c = make(chan struct{})
}

// Close closes the waiter and its notification channel.
func (w *waiter) Close() (err error) {
	w.mux.Lock()
	defer w.mux.Unlock()
	if w.closed {
		return ErrIsClosed
	}

	w.closed = true
	close(w.c)
	return nil
}
