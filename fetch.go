package ski

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
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

type _fetch string

// fetch the resource from the network, default method is GET
// Method http://example.com
func fetch(arg Arguments) (Executor, error) {
	return _fetch(arg.GetString(0)), nil
}

func (f _fetch) Exec(ctx context.Context, _ any) (any, error) {
	method, url, found := strings.Cut(string(f), " ")
	if !found {
		url = string(f)
		method = http.MethodGet
	}

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ski")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}
