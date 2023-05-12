package cloudcat

import (
	"net/http"
)

// XFromCache is the header added to responses that are returned from the cache
const XFromCache = "X-From-Cache"

// Fetch http client interface
type Fetch interface {
	// Do sends an HTTP request and returns an HTTP response, following
	// policy (such as redirects, cookies, auth) as configured on the
	// client.
	Do(*http.Request) (*http.Response, error)
}

// IsFromCache returns true if the response is from cache
func IsFromCache(res *http.Response) bool { return res.Header.Get(XFromCache) == "1" }
