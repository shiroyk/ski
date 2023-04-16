package fetch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"

	"golang.org/x/exp/slog"
)

type proxyURLKey int

var proxyMap = NewLRUCache[string, roundRobinProxy](128)

type roundRobinProxy struct {
	proxyURLs []*url.URL
	hash      string
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

// AddRoundRobinProxy add the proxy URLs for the specified URL.
// The proxy type is determined by the URL scheme. "http", "https"
// and "socks5" are supported. If the scheme is empty,
// "http" is assumed.
func AddRoundRobinProxy(u string, proxyURLs ...string) {
	if len(proxyURLs) == 0 {
		return
	}
	sum := sha256.Sum256([]byte(strings.Join(proxyURLs, "")))
	hash := hex.EncodeToString(sum[:])
	if p, ok := proxyMap.Get(u); ok {
		if p.hash == hash {
			return
		}
	}
	parsedProxyURLs := make([]*url.URL, len(proxyURLs))
	for i, pu := range proxyURLs {
		parsedURL, err := url.Parse(pu)
		if err != nil {
			slog.Error("proxy url error %s", err)
		}
		parsedProxyURLs[i] = parsedURL
	}

	proxyMap.Add(u, roundRobinProxy{parsedProxyURLs, hash, 0})
}

// RoundRobinProxy returns a proxy URL on specific request.
func RoundRobinProxy(req *http.Request) (*url.URL, error) {
	if p, ok := proxyMap.Get(req.URL.String()); ok {
		return p.getProxy(req)
	}
	return http.ProxyFromEnvironment(req)
}
