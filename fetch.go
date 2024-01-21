package ski

import (
	"context"
	"net"
	"net/http"
	"net/url"
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
			Proxy: ProxyFromRequest,
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
		Jar: NewCookieJar(),
	}
}

var requestProxyKey byte

// WithProxyURL returns a copy of parent context in which the proxy associated with context.
func WithProxyURL(ctx context.Context, proxy *url.URL) context.Context {
	if proxy == nil {
		return ctx
	}
	if c, ok := ctx.(Context); ok {
		c.SetValue(&requestProxyKey, proxy)
		return c
	}
	return context.WithValue(ctx, &requestProxyKey, proxy)
}

// ProxyFromContext returns a proxy URL on context.
func ProxyFromContext(ctx context.Context) *url.URL {
	if proxy := ctx.Value(&requestProxyKey); proxy != nil {
		return proxy.(*url.URL)
	}
	return nil
}

// ProxyFromRequest returns a proxy URL on request context.
func ProxyFromRequest(req *http.Request) (*url.URL, error) {
	return ProxyFromContext(req.Context()), nil
}
