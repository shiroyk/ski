package fetch

import (
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"

	"golang.org/x/exp/slog"
)

type roundRobinProxy struct {
	proxyURLs []*url.URL
	index     uint32
}

// getProxy returns a proxy URL for the given http.Request
func (r *roundRobinProxy) getProxy() (*url.URL, error) {
	index := atomic.AddUint32(&r.index, 1) - 1
	return r.proxyURLs[index%uint32(len(r.proxyURLs))], nil
}

// newRoundRobinProxy create the roundRobinProxy for the specified URL.
// The proxy type is determined by the URL scheme. "http", "https"
// and "socks5" are supported. If the scheme is empty,
// "http" is assumed.
func newRoundRobinProxy(proxyURLs ...string) *roundRobinProxy {
	if len(proxyURLs) == 0 {
		return nil
	}
	parsedProxyURLs := make([]*url.URL, len(proxyURLs))
	for i, pu := range proxyURLs {
		parsedURL, err := url.Parse(pu)
		if err != nil {
			slog.Error(fmt.Sprintf("proxy url %s error", pu), "error", err)
		}
		parsedProxyURLs[i] = parsedURL
	}

	return &roundRobinProxy{parsedProxyURLs, 0}
}

// RoundRobinProxy returns a proxy URL on specific request.
func RoundRobinProxy(req *http.Request) (*url.URL, error) {
	if c := GetRequestConfig(req); c.roundRobinProxy != nil {
		return c.roundRobinProxy.getProxy()
	}
	return http.ProxyFromEnvironment(req)
}
