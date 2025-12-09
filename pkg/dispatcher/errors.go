package dispatcher

import "errors"

var (
	ErrClosed     = errors.New("dispatcher closed")
	ErrQueueFull  = errors.New("dispatcher queue full")
	ErrNilRequest = errors.New("nil request")
)
