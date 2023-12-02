package cloudcat

import (
	"context"
	"net/http"
	"net/url"

	"github.com/shiroyk/cloudcat/plugin"
)

// Fetch http client interface
type Fetch interface {
	// Do sends an HTTP request and returns an HTTP response, following
	// policy (such as redirects, cookies, auth) as configured on the
	// client.
	Do(*http.Request) (*http.Response, error)
}

type requestProxyKey struct{}

// WithProxyURL returns a copy of parent context in which the proxy associated with context.
func WithProxyURL(ctx context.Context, proxy *url.URL) context.Context {
	if proxy == nil {
		return ctx
	}
	if c, ok := ctx.(*plugin.Context); ok {
		c.SetValue(requestProxyKey{}, proxy)
		return ctx
	}
	return context.WithValue(ctx, requestProxyKey{}, proxy)
}

// ProxyFromContext returns a proxy URL on context.
func ProxyFromContext(ctx context.Context) *url.URL {
	if proxy := ctx.Value(requestProxyKey{}); proxy != nil {
		return proxy.(*url.URL)
	}
	return nil
}

// ProxyFromRequest returns a proxy URL on request context.
func ProxyFromRequest(req *http.Request) (*url.URL, error) {
	return ProxyFromContext(req.Context()), nil
}
