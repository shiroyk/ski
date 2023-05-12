package fetch

import (
	"compress/gzip"
	"compress/zlib"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
	utls "github.com/refraction-networking/utls"
	"github.com/shiroyk/cloudcat/core"
	"golang.org/x/exp/slices"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/transform"
)

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
	// DefaultRetryTimes fetch.RequestConfig retry times
	DefaultRetryTimes = 3
	// DefaultTimeout fetch.RequestConfig timeout
	DefaultTimeout = time.Minute
)

var (
	// DefaultRetryHTTPCodes retry fetch.RequestConfig error status code
	DefaultRetryHTTPCodes = []int{http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, //nolint:lll
		http.StatusGatewayTimeout, http.StatusRequestTimeout}
	// DefaultHeaders defaults fetch.RequestConfig headers
	DefaultHeaders = map[string]string{
		"Accept":          "*/*",
		"Accept-Encoding": "gzip, deflate, br",
		"Accept-Language": "en-US,en;",
		"User-Agent":      "cloudcat",
	}
)

// Options The Fetch instance options
type Options struct {
	CharsetDetectDisabled bool              `yaml:"charset-detect-disabled"`
	MaxBodySize           int64             `yaml:"max-body-size"`
	RetryTimes            int               `yaml:"retry-times"`
	RetryHTTPCodes        []int             `yaml:"retry-http-codes"`
	Timeout               time.Duration     `yaml:"timeout"`
	CachePolicy           Policy            `yaml:"cache-policy"`
	RoundTripper          http.RoundTripper `yaml:"-"`
}

// NewFetcher returns a new Fetch instance
func NewFetcher(opt Options) cloudcat.Fetch {
	fetch := new(fetcher)

	fetch.charsetDetectDisabled = opt.CharsetDetectDisabled
	fetch.maxBodySize = cloudcat.ZeroOr(opt.MaxBodySize, DefaultMaxBodySize)
	fetch.timeout = cloudcat.ZeroOr(opt.Timeout, DefaultTimeout)
	fetch.retryTimes = cloudcat.ZeroOr(opt.RetryTimes, DefaultRetryTimes)
	fetch.retryHTTPCodes = cloudcat.EmptyOr(opt.RetryHTTPCodes, DefaultRetryHTTPCodes)

	transport := opt.RoundTripper
	if transport == nil {
		transport = DefaultRoundTripper()
	}

	ch, _ := cloudcat.Resolve[cloudcat.Cache]()
	if ch != nil {
		transport = &CacheTransport{
			Cache:               ch,
			Policy:              cloudcat.ZeroOr(opt.CachePolicy, RFC2616),
			Transport:           transport,
			MarkCachedResponses: true,
		}
	}

	cookie, _ := cloudcat.Resolve[cloudcat.Cookie]()
	fetch.Client = &http.Client{
		Transport: transport,
		Timeout:   fetch.timeout,
		Jar:       cookie,
	}
	return fetch
}

// DefaultRoundTripper the fetch default RoundTripper
func DefaultRoundTripper() http.RoundTripper {
	return &http.Transport{
		Proxy: RoundRobinProxy,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			tcpConn, err := (new(net.Dialer)).DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			sni, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			config := utls.Config{ServerName: sni}
			tlsConn := utls.UClient(tcpConn, &config, utls.HelloRandomizedNoALPN)
			if err = tlsConn.HandshakeContext(ctx); err != nil {
				return nil, err
			}

			return tlsConn, nil
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          0,
		MaxIdleConnsPerHost:   1000,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// Do sends an HTTP request and returns an HTTP response, following
// policy (such as redirects, cookies, auth) as configured on the
// client.
func (f *fetcher) Do(req *http.Request) (*http.Response, error) {
	config := GetRequestConfig(req)
	return f.doRequestRetry(req, &config)
}

func (f *fetcher) doRequestRetry(req *http.Request, config *RequestConfig) (*http.Response, error) {
	res, err := f.doRequest(req, config)
	// Retry on Error
	if err != nil {
		if config.retryCounter < f.retryTimes {
			config.retryCounter++
			return f.doRequestRetry(req, config)
		}
		return res, err
	}

	// Retry on http status codes
	if slices.Contains(f.retryHTTPCodes, res.StatusCode) {
		if config.retryCounter < f.retryTimes {
			config.retryCounter++
			return f.doRequestRetry(req, config)
		}
	}

	return res, err
}

func (f *fetcher) doRequest(req *http.Request, config *RequestConfig) (*http.Response, error) {
	res, err := f.Client.Do(req)
	if err != nil {
		return nil, err
	}

	// Limit response body reading
	bodyReader := io.LimitReader(res.Body, f.maxBodySize)

	if res.Request.Method != http.MethodHead { //nolint:nestif
		if encoding := res.Header.Get("Content-Encoding"); encoding != "" {
			bodyReader, err = decompressedBody(encoding, bodyReader)
			if err != nil {
				return nil, err
			}
			res.Body = io.NopCloser(bodyReader)
		}

		if res.ContentLength > 0 {
			if config.Encoding != "" {
				if enc, _ := charset.Lookup(config.Encoding); enc != nil {
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
			res.Body = io.NopCloser(bodyReader)
		}
	}

	return res, nil
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
