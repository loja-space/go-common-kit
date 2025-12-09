package dispatcher

import (
	"context"
	"sync"
)

// ------------------------------------
// ResponseFuture
// ------------------------------------

// ResponseFuture behaves like a simple "future" or "promise".
// It is completed exactly once by the dispatcher when processing finishes.
// Users can wait synchronously using Wait() or register asynchronous callbacks.
type ResponseFuture struct {
	ch   chan struct{} // Closed when response is ready
	resp *Response

	once sync.Once
	mu   sync.Mutex
}

// newResponseFuture allocates a new future.
// The channel is unbuffered and closed when completed.
func newResponseFuture() *ResponseFuture {
	return &ResponseFuture{
		ch: make(chan struct{}),
	}
}

// setResponse completes the future exactly once.
// Any duplicate calls or late updates are ignored.
// Closing the channel signals to all waiters that the response is available.
func (f *ResponseFuture) setResponse(resp *Response) {
	f.once.Do(func() {
		f.mu.Lock()
		f.resp = resp
		f.mu.Unlock()
		close(f.ch)
	})
}

// Done returns a read-only channel that is closed when the response is ready.
// This allows integration with select-based asynchronous waiting.
func (f *ResponseFuture) Done() <-chan struct{} {
	return f.ch
}

// Wait blocks until the future is completed or the context is cancelled.
// If ctx is done first, it returns ctx.Err().
func (f *ResponseFuture) Wait(ctx context.Context) (*Response, error) {
	select {
	case <-f.ch:
		f.mu.Lock()
		r := f.resp
		f.mu.Unlock()
		return r, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Result returns the response and a boolean indicating whether the future is done.
// If not completed, it returns (nil, false).
func (f *ResponseFuture) Result() (*Response, bool) {
	select {
	case <-f.ch:
		f.mu.Lock()
		r := f.resp
		f.mu.Unlock()
		return r, true
	default:
		return nil, false
	}
}

// OnDone registers a callback executed asynchronously when the future completes.
// If the future is already completed, the callback runs immediately in a goroutine.
// The callback receives the final Response object.
func (f *ResponseFuture) OnDone(cb func(*Response)) {
	go func() {
		<-f.ch
		f.mu.Lock()
		r := f.resp
		f.mu.Unlock()
		cb(r)
	}()
}
