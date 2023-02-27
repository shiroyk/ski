package fetch

import (
	"compress/gzip"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/shiroyk/cloudcat/cache"
	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/lib/consts"
	"github.com/shiroyk/cloudcat/lib/utils"
	"golang.org/x/exp/slices"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/transform"
)

// Fetch http client interface
type Fetch interface {
	// Get issues a GET to the specified URL string and optional headers.
	Get(url string, headers map[string]string) (*Response, error)
	// Post issues a POST to the specified URL string, body and optional headers.
	Post(url string, body any, headers map[string]string) (*Response, error)
	// Head issues a HEAD to the specified URL string and optional headers.
	Head(url string, headers map[string]string) (*Response, error)
	// Request sends request with specified method, url, body, headers; returns an HTTP response.
	Request(method, url string, body any, headers map[string]string) (*Response, error)
	// DoRequest sends a fetch.Request and returns an HTTP response.
	DoRequest(*Request) (*Response, error)
}

type fetcher struct {
	*http.Client
	charsetDetectDisabled bool
	maxBodySize           int64
	retryTimes            int
	retryHTTPCodes        []int
	timeout               time.Duration
}

const (
	// DefaultMaxBodySize fetch.Response default max body size
	DefaultMaxBodySize int64 = 1024 * 1024 * 1024
	// DefaultRetryTimes fetch.Request retry times
	DefaultRetryTimes = 3
	// DefaultTimeout fetch.Request timeout
	DefaultTimeout = time.Minute
)

var (
	// DefaultRetryHTTPCodes retry fetch.Request error status code
	DefaultRetryHTTPCodes = []int{http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable,
		http.StatusGatewayTimeout, http.StatusRequestTimeout}
	// DefaultHeaders defaults fetch.Request headers
	DefaultHeaders = map[string]string{
		"Accept":          "*/*",
		"Accept-Encoding": "gzip, deflate, br",
		"Accept-Language": "en-US,en;",
		"User-Agent":      fmt.Sprintf("cloudcat/%v", consts.Version),
	}
	// ErrRequestCancel fetch.Request cancel error
	ErrRequestCancel = errors.New("request canceled")
)

// Options The Fetch instance options
type Options struct {
	CharsetDetectDisabled bool                                  `yaml:"charset-detect-disabled"`
	MaxBodySize           int64                                 `yaml:"max-body-size"`
	RetryTimes            int                                   `yaml:"retry-times"`
	RetryHTTPCodes        []int                                 `yaml:"retry-http-codes"`
	Timeout               time.Duration                         `yaml:"timeout"`
	CachePolicy           cache.Policy                          `yaml:"cache-policy"`
	ProxyFunc             func(*http.Request) (*url.URL, error) `yaml:"-"`
}

// NewFetcher returns a new Fetch instance
func NewFetcher(opt Options) Fetch {
	fetch := new(fetcher)

	fetch.charsetDetectDisabled = opt.CharsetDetectDisabled
	fetch.maxBodySize = utils.ZeroOr(opt.MaxBodySize, DefaultMaxBodySize)
	fetch.timeout = utils.ZeroOr(opt.Timeout, DefaultTimeout)
	fetch.retryTimes = utils.ZeroOr(opt.RetryTimes, DefaultRetryTimes)
	fetch.retryHTTPCodes = utils.EmptyOr(opt.RetryHTTPCodes, DefaultRetryHTTPCodes)

	proxy := opt.ProxyFunc
	if proxy == nil {
		proxy = RoundRobinProxy
	}

	var transport http.RoundTripper = &http.Transport{
		Proxy: proxy,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          0,
		MaxIdleConnsPerHost:   1000,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	ch, _ := di.Resolve[cache.Cache]()
	if ch != nil {
		transport = &cache.Transport{
			Cache:               ch,
			Policy:              utils.ZeroOr(opt.CachePolicy, cache.RFC2616),
			Transport:           transport,
			MarkCachedResponses: true,
		}
	}

	cookie, _ := di.Resolve[cache.Cookie]()
	fetch.Client = &http.Client{
		Transport: transport,
		Timeout:   fetch.timeout,
		Jar:       cookie,
	}
	return fetch
}

// Get issues a GET to the specified URL string and optional headers.
func (f *fetcher) Get(url string, headers map[string]string) (*Response, error) {
	return f.Request(http.MethodGet, url, nil, headers)
}

// Post issues a POST to the specified URL string, body and optional headers.
func (f *fetcher) Post(url string, body any, headers map[string]string) (*Response, error) {
	return f.Request(http.MethodPost, url, body, headers)
}

// Head issues a HEAD to the specified URL string and optional headers.
func (f *fetcher) Head(url string, headers map[string]string) (*Response, error) {
	return f.Request(http.MethodHead, url, nil, headers)
}

// Request sends request with specified method, url, body, headers; returns an HTTP response.
func (f *fetcher) Request(method, url string, body any, headers map[string]string) (*Response, error) {
	request, err := NewRequest(method, url, body, headers)
	if err != nil {
		return nil, err
	}
	return f.DoRequest(request)
}

// DoRequest sends a fetch.Request and returns an HTTP response.
func (f *fetcher) DoRequest(req *Request) (*Response, error) {
	AddRoundRobinProxy(req.URL.String(), req.Proxy...)
	return f.doRequestRetry(req)
}

func (f *fetcher) doRequestRetry(req *Request) (*Response, error) {
	if req.Cancelled {
		return nil, ErrRequestCancel
	}
	res, err := f.doRequest(req)

	// Retry on Error
	if err != nil {
		if req.retryCounter < f.retryTimes {
			req.retryCounter++
			return f.doRequestRetry(req)
		}
		return res, err
	}

	// Retry on http status codes
	if slices.Contains(f.retryHTTPCodes, res.StatusCode) {
		if req.retryCounter < f.retryTimes {
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
	bodyReader := io.LimitReader(res.Body, f.maxBodySize)

	if encoding := res.Header.Get("Content-Encoding"); encoding != "" {
		bodyReader, err = decompressedBody(encoding, bodyReader)
		if err != nil {
			return nil, err
		}
	}

	if res.Request.Method != http.MethodHead && res.ContentLength > 0 {
		if req.Encoding != "" {
			if enc, _ := charset.Lookup(req.Encoding); enc != nil {
				bodyReader = transform.NewReader(bodyReader, enc.NewDecoder())
			}
		} else {
			if !f.charsetDetectDisabled {
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

func decompressedBody(encoding string, reader io.Reader) (bodyReader io.Reader, err error) {
	contentEncodings := strings.Split(encoding, ",")
	// In the order decompressed
	for _, encode := range contentEncodings {
		switch strings.TrimSpace(encode) {
		case "deflate":
			bodyReader, err = zlib.NewReader(reader)
		case "gzip":
			bodyReader, err = gzip.NewReader(reader)
		case "br":
			bodyReader = brotli.NewReader(reader)
		default:
			err = fmt.Errorf("unsupported compression type %s", encode)
		}
		if err != nil {
			return nil, err
		}
	}
	return
}
