package fetcher

import (
	"net/http"
)

// Response type wraps http.Response
type Response struct {
	*http.Response

	// Response Body
	Body []byte
}

// ContentType returns Response Header Content type
func (r *Response) ContentType() string {
	return r.Header.Get("Content-Type")
}

// String returns Response string Body
func (r *Response) String() string {
	return string(r.Body)
}
