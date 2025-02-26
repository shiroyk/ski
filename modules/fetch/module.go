package fetch

import (
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/modules"
)

func init() {
	jar := NewCookieJar()
	client := NewClient()
	client.Jar = jar
	modules.Register("http", &Http{client})
	modules.Register("cookieJar", &CookieJarModule{jar})
	modules.Register("fetch", &Module{client, jar})
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

type Module struct {
	Client
	CookieJar
}

func (m *Module) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	if m.Client == nil {
		return nil, errors.New("http client can not be nil")
	}
	if m.CookieJar == nil {
		return nil, errors.New("CookieJar can not nil")
	}
	ret := rt.NewObject()
	fetch, _ := (&Fetch{m.Client}).Instantiate(rt)
	_ = ret.Set("fetch", fetch)
	request, _ := new(Request).Instantiate(rt)
	_ = ret.Set("Request", request)
	response, _ := new(Response).Instantiate(rt)
	_ = ret.Set("Response", response)
	headers, _ := new(Headers).Instantiate(rt)
	_ = ret.Set("Headers", headers)
	formData, _ := new(FormData).Instantiate(rt)
	_ = ret.Set("FormData", formData)
	abortController, _ := new(AbortController).Instantiate(rt)
	_ = ret.Set("AbortController", abortController)
	abortSignal, _ := new(AbortSignal).Instantiate(rt)
	_ = ret.Set("AbortSignal", abortSignal)
	return ret, nil
}

func (*Module) Global() {}
