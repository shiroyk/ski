package fetcher

import (
	"context"
	"net/http"
	"net/url"
	"sync/atomic"

	"github.com/labstack/gommon/log"
	"github.com/shiroyk/cloudcat/utils"
)

type ProxyURLKey int

var (
	cacheProxy = utils.NewLRUCache[string, roundRobinProxy](64)
)

type roundRobinProxy struct {
	proxyURLs []*url.URL
	index     uint32
}

func (r *roundRobinProxy) GetProxy(pr *http.Request) (*url.URL, error) {
	index := atomic.AddUint32(&r.index, 1) - 1
	u := r.proxyURLs[index%uint32(len(r.proxyURLs))]
	// Set proxy url to context
	ctx := context.WithValue(pr.Context(), ProxyURLKey(0), u.String())
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

	if p, ok := cacheProxy.Get(u); ok {
		return p.GetProxy
	} else {
		parsedProxyURLs := make([]*url.URL, len(proxyURLs))
		for i, pu := range proxyURLs {
			parsedURL, err := url.Parse(pu)
			if err != nil {
				log.Infof("proxy url parse: %v", err)
				return nil
			}
			parsedProxyURLs[i] = parsedURL
		}

		p = roundRobinProxy{parsedProxyURLs, 0}
		cacheProxy.Add(u, p)
		return p.GetProxy
	}
}
