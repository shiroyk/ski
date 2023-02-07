package fetch

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/shiroyk/cloudcat/cache"
	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/utils"
	"golang.org/x/exp/slices"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/transform"
)

type Fetch interface {
	Get(url string, headers map[string]string) (*Response, error)
	Post(url string, body any, headers map[string]string) (*Response, error)
	Head(url string, headers map[string]string) (*Response, error)
	Request(method, url string, body any, headers map[string]string) (*Response, error)
	DoRequest(*Request) (*Response, error)
}

type fetcher struct {
	*http.Client
	opt Options
}

const (
	DefaultUserAgent         = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.0.0 Safari/537.36"
	DefaultMaxBodySize int64 = 1024 * 1024 * 1024
	DefaultRetryTimes        = 3
	DefaultTimeout           = time.Second * 180
)

var (
	DefaultRetryHTTPCodes = []int{http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable,
		http.StatusGatewayTimeout, http.StatusRequestTimeout}
	DefaultHeaders = map[string]string{
		"Accept":          "*/*",
		"Accept-Encoding": "gzip",
		"User-Agent":      DefaultUserAgent,
	}
	ErrRequestCancel = errors.New("request canceled")
)

type Options struct {
	CharsetDetectDisabled bool
	MaxBodySize           int64
	RetryTimes            int
	RetryHTTPCodes        []int
	Timeout               time.Duration
}

func NewFetcher(opt Options) Fetch {
	fetch := new(fetcher)

	var transport http.RoundTripper = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          0,    // Default: 100
		MaxIdleConnsPerHost:   1000, // Default: 2
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	ch, _ := di.Resolve[cache.Cache]()
	if ch != nil {
		transport = &cache.Transport{
			Cache:               ch,
			Policy:              cache.RFC2616,
			Transport:           transport,
			MarkCachedResponses: true,
		}
	}

	cookie, _ := di.Resolve[cache.Cookie]()
	fetch.Client = &http.Client{
		Transport: transport,
		Timeout:   opt.Timeout,
		Jar:       cookie,
	}
	fetch.opt = opt
	return fetch
}

func (f *fetcher) Get(url string, headers map[string]string) (*Response, error) {
	return f.Request(http.MethodGet, url, nil, headers)
}

func (f *fetcher) Post(url string, body any, headers map[string]string) (*Response, error) {
	return f.Request(http.MethodPost, url, body, headers)
}

func (f *fetcher) Head(url string, headers map[string]string) (*Response, error) {
	return f.Request(http.MethodHead, url, nil, headers)
}

func (f *fetcher) Request(method, url string, body any, headers map[string]string) (*Response, error) {
	request, err := NewRequest(method, url, body, headers)
	if err != nil {
		return nil, err
	}
	return f.DoRequest(request)
}

func (f *fetcher) DoRequest(req *Request) (*Response, error) {
	f.Transport.(*http.Transport).Proxy = RoundRobinCacheProxy(req.URL.String(), req.Proxy...)
	return f.doRequestRetry(req)
}

func (f *fetcher) doRequestRetry(req *Request) (*Response, error) {
	if req.Cancelled {
		return nil, ErrRequestCancel
	}
	res, err := f.doRequest(req)

	// Retry on Error
	if err != nil {
		if req.retryCounter < utils.ZeroOr(req.RetryTimes, f.opt.RetryTimes) {
			req.retryCounter++
			return f.doRequestRetry(req)
		}
		return res, err
	}

	// Retry on http status codes
	if slices.Contains(utils.EmptyOr(req.RetryHTTPCodes, f.opt.RetryHTTPCodes), res.StatusCode) {
		if req.retryCounter < utils.ZeroOr(req.RetryTimes, f.opt.RetryTimes) {
			req.retryCounter++
			return f.doRequestRetry(req)
		}
	}

	return res, err
}

func (f *fetcher) doRequest(req *Request) (*Response, error) {
	res, err := f.Do(req.Request)
	defer func() {
		if res != nil {
			res.Body.Close()
		}
	}()
	if err != nil {
		return nil, err
	}

	// Limit response body reading
	bodyReader := io.LimitReader(res.Body, utils.ZeroOr(f.opt.MaxBodySize, DefaultMaxBodySize))

	if res.Request.Method != http.MethodHead && res.ContentLength > 0 {
		if req.Encoding != "" {
			if enc, _ := charset.Lookup(req.Encoding); enc != nil {
				bodyReader = transform.NewReader(bodyReader, enc.NewDecoder())
			}
		} else {
			if !f.opt.CharsetDetectDisabled {
				contentType := req.Header.Get("Content-Type")
				bodyReader, err = charset.NewReader(bodyReader, contentType)
				if err != nil {
					return nil, fmt.Errorf("charset detection error on content-type %s: %w", contentType, err)
				}
			}
		}
	}

	body, err := io.ReadAll(bodyReader)
	if err != nil {
		return nil, err
	}

	return &Response{Response: res, Body: body}, nil
}
