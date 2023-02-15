package fetch

import (
	"context"
	"net/http"
	"net/url"
	"sync/atomic"

	"github.com/shiroyk/cloudcat/utils"
	"github.com/shiroyk/cloudcat/utils/logger"
)

type proxyURLKey int

var (
	cacheProxy = utils.NewLRUCache[string, roundRobinProxy](64)
)

type roundRobinProxy struct {
	proxyURLs []*url.URL
	index     uint32
}

// getProxy returns a proxy URL for the given http.Request
func (r *roundRobinProxy) getProxy(pr *http.Request) (*url.URL, error) {
	index := atomic.AddUint32(&r.index, 1) - 1
	u := r.proxyURLs[index%uint32(len(r.proxyURLs))]
	// Set proxy url to context
	ctx := context.WithValue(pr.Context(), proxyURLKey(0), u.String())
	*pr = *pr.WithContext(ctx)
	return u, nil
}

// RoundRobinCacheProxy creates a cache proxy switcher function which rotates
// ProxyURLs on specific request.
// The proxy type is determined by the URL scheme. "http", "https"
// and "socks5" are supported. If the scheme is empty,
// "http" is assumed.
func RoundRobinCacheProxy(u string, proxyURLs ...string) func(*http.Request) (*url.URL, error) {
	if len(proxyURLs) < 1 {
		return http.ProxyFromEnvironment
	}

	var p roundRobinProxy
	var ok bool
	p, ok = cacheProxy.Get(u)
	if ok {
		return p.getProxy
	}

	parsedProxyURLs := make([]*url.URL, len(proxyURLs))
	for i, pu := range proxyURLs {
		parsedURL, err := url.Parse(pu)
		if err != nil {
			logger.Errorf("proxy url error %s", err)
			return nil
		}
		parsedProxyURLs[i] = parsedURL
	}

	p = roundRobinProxy{parsedProxyURLs, 0}
	cacheProxy.Add(u, p)
	return p.getProxy
}
