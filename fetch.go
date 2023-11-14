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

var requestProxyKey struct{}

// WithProxyURL returns a copy of parent context in which the proxy associated with context.
func WithProxyURL(ctx context.Context, proxy *url.URL) context.Context {
	if proxy == nil {
		return ctx
	}
	if c, ok := ctx.(*plugin.Context); ok {
		c.SetValue(&requestProxyKey, proxy)
		return ctx
	}
	return context.WithValue(ctx, &requestProxyKey, proxy)
}

// ProxyFromRequest returns a proxy URL on request context.
func ProxyFromRequest(req *http.Request) (*url.URL, error) {
	if proxy := req.Context().Value(&requestProxyKey); proxy != nil {
		return proxy.(*url.URL), nil
	}
	return nil, nil
}
