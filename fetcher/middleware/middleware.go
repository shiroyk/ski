package middleware

import "github.com/shiroyk/cloudcat/fetcher"

// RequestResponseProcessor interface is for middlewares that needs to process both requests and responses
type RequestResponseProcessor interface {
	RequestProcessor
	ResponseProcessor
}

// RequestProcessor called before requests made.
// Set request.Cancelled = true to cancel request
type RequestProcessor interface {
	ProcessRequest(r *fetcher.Request)
}

// ResponseProcessor called after request response receive
type ResponseProcessor interface {
	ProcessResponse(r *fetcher.Response)
}
