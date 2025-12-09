package dispatcher

// Response wraps the result produced by the dispatcher handler.
// Every Response always contains the corresponding Request.
// Result and Error are mutually exclusive (Error indicates failure).
type Response struct {
	Req    *Request
	Result any
	Error  error
}
