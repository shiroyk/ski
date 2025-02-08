package ski

import (
	"net"
	"net/http"
	"time"
)

// Fetch http client interface
type Fetch interface {
	// Do sends an HTTP request and returns an HTTP response, following
	// policy (such as redirects, cookies, auth) as configured on the
	// client.
	Do(*http.Request) (*http.Response, error)
}

// NewFetch return the http.Client implementation
func NewFetch() Fetch {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}
