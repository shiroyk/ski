package fetch

import (
	"net"
	"net/http"
	"time"

	"github.com/shiroyk/ski/modules"
)

func init() {
	jar := NewCookieJar()
	client := NewClient()
	client.Jar = jar
	modules.Register("cookieJar", &CookieJarModule{jar})
	modules.Register("fetch", modules.Global{
		"fetch":    Fetch(client),
		"Request":  new(Request),
		"Response": new(Response),
		"Headers":  new(Headers),
		"FormData": new(FormData),
	})
}

// Client http client interface
type Client interface {
	// Do sends an HTTP request and returns an HTTP response, following
	// policy (such as redirects, cookies, auth) as configured on the
	// client.
	Do(*http.Request) (*http.Response, error)
}

// NewClient return the http.Client implementation
func NewClient() *http.Client {
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
