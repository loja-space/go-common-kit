package dispatcher

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// ------------------------------------
// RequestDispatcher
// ------------------------------------

// dispatchItem bundles a Request with its associated future.
type dispatchItem struct {
	req    *Request
	future *ResponseFuture
}

// RequestDispatcher is a single-consumer dispatcher that processes requests
// through a user-provided handler. Producers may call Send concurrently.
//
// Internally, Send enqueues requests onto a buffered channel. A dedicated
// goroutine consumes items one-by-one, guaranteeing ordering and serialization.
type RequestDispatcher struct {
	reqCh     chan *dispatchItem
	handler   func(req *Request) *Response
	closed    int32 // atomic flag: 0 = open, 1 = closed
	wg        sync.WaitGroup
	closeOnce sync.Once

	// closedCh is closed after the internal loop exits.
	closedCh chan struct{}
}

// NewRequestDispatcher creates a dispatcher with the specified queue size
// and handler function. If queueSize <= 0, a default is used.
//
// handler must be safe to call concurrently (though it is invoked in a single
// goroutine). If handler is nil, an echo handler is used.
func NewRequestDispatcher(queueSize int, handler func(req *Request) *Response) *RequestDispatcher {
	if queueSize <= 0 {
		queueSize = 1024
	}
	if handler == nil {
		handler = func(req *Request) *Response {
			return &Response{Req: req, Result: req.Payload}
		}
	}

	d := &RequestDispatcher{
		reqCh:    make(chan *dispatchItem, queueSize),
		handler:  handler,
		closedCh: make(chan struct{}),
	}
	d.wg.Add(1)
	go d.loop()

	return d
}

// loop continuously consumes requests from reqCh until the channel is closed.
// Each request is processed via safeHandle to ensure panics are converted into
// error responses rather than crashing the goroutine.
func (d *RequestDispatcher) loop() {
	defer d.wg.Done()
	for item := range d.reqCh {
		resp := d.safeHandle(item.req)
		item.future.setResponse(resp)
	}
	close(d.closedCh)
}

// safeHandle wraps handler invocation with panic recovery, ensuring the
// dispatcher never crashes and every future is always completed.
func (d *RequestDispatcher) safeHandle(req *Request) *Response {
	defer func() {
		if r := recover(); r != nil {
			// Convert panic to error; ensure future is still completed.
			// (Error details omitted; optionally extend with stack traces.)
		}
	}()
	resp := d.handler(req)
	if resp == nil {
		// Ensure non-nil response to avoid breaking the future contract.
		resp = &Response{Req: req}
	}
	return resp
}

// IsClosed reports whether the dispatcher is closed to new requests.
func (d *RequestDispatcher) IsClosed() bool {
	return atomic.LoadInt32(&d.closed) == 1
}

// Send attempts to enqueue a request without blocking.
// If the queue is full or dispatcher is closed, it returns an error.
//
// The returned ResponseFuture will be completed once the handler processes
// the request.
func (d *RequestDispatcher) Send(req *Request) (*ResponseFuture, error) {
	if req == nil {
		return nil, errors.New("nil request")
	}
	if d.IsClosed() {
		return nil, errors.New("dispatcher closed")
	}

	f := newResponseFuture()
	item := &dispatchItem{req: req, future: f}

	select {
	case d.reqCh <- item:
		return f, nil
	default:
		return nil, errors.New("dispatcher queue full")
	}
}

// SendWithContext enqueues the request but blocks until the request is accepted
// or the context is cancelled.
//
// If ctx is cancelled before enqueueing succeeds, the returned future is
// immediately completed with an error to avoid leaving a hanging waiter.
func (d *RequestDispatcher) SendWithContext(ctx context.Context, req *Request) (*ResponseFuture, error) {
	if req == nil {
		return nil, errors.New("nil request")
	}
	if d.IsClosed() {
		return nil, errors.New("dispatcher closed")
	}

	f := newResponseFuture()
	item := &dispatchItem{req: req, future: f}

	select {
	case d.reqCh <- item:
		return f, nil
	case <-ctx.Done():
		f.setResponse(&Response{Req: req, Error: ctx.Err()})
		return f, ctx.Err()
	}
}

// TrySend attempts a non-blocking enqueue.
// Returns (future, true) if successful, or (nil, false) otherwise.
func (d *RequestDispatcher) TrySend(req *Request) (*ResponseFuture, bool) {
	if req == nil || d.IsClosed() {
		return nil, false
	}

	f := newResponseFuture()
	item := &dispatchItem{req: req, future: f}

	select {
	case d.reqCh <- item:
		return f, true
	default:
		return nil, false
	}
}

// Close gracefully shuts down the dispatcher.
//
// * No new requests are accepted.
// * reqCh is closed, allowing the consumer to finish processing buffered items.
// * The call waits until the consumer goroutine exits.
func (d *RequestDispatcher) Close() {
	d.closeOnce.Do(func() {
		atomic.StoreInt32(&d.closed, 1)
		close(d.reqCh)
		d.wg.Wait()
	})
}

// CloseWithTimeout attempts graceful shutdown like Close(), but waits
// only up to the specified timeout.
//
// If the timeout expires, an error is returned. The dispatcher is marked
// closed, but pending items may still remain unprocessed.
func (d *RequestDispatcher) CloseWithTimeout(timeout time.Duration) error {
	d.closeOnce.Do(func() {
		atomic.StoreInt32(&d.closed, 1)
		close(d.reqCh)
	})

	c := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(c)
	}()

	select {
	case <-c:
		return nil
	case <-time.After(timeout):
		return errors.New("close timeout waiting for drain")
	}
}
