package dispatcher

import "time"

// ------------------------------------
// Request and Response Types
// ------------------------------------

// Request represents a message submitted to the dispatcher.
// The caller must provide a unique RequestID.
//
// Payload, Meta, FromApp, and Timestamp are optional fields describing
// request content and metadata.
type Request struct {
	RequestID int32
	Payload   any
	FromApp   string
	Timestamp time.Time
	Meta      map[string]any
}
