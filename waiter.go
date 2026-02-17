package streambuf

import "sync"

func newWaiter() (out *waiter) {
	var w waiter
	w.c = make(chan struct{})
	return &w
}

type waiter struct {
	mux sync.RWMutex

	c chan struct{}

	closed bool
}

func (w *waiter) Wait() (out <-chan struct{}) {
	w.mux.RLock()
	defer w.mux.RUnlock()
	return w.c
}

func (w *waiter) Refresh() {
	w.mux.Lock()
	defer w.mux.Unlock()
	close(w.c)
	w.c = make(chan struct{})
}

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
