package fetch

import (
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"

	"github.com/andybalholm/brotli"
	tls "github.com/refraction-networking/utls"
	"github.com/shiroyk/cloudcat/core"
	"github.com/shiroyk/cloudcat/fetch/http2"
	"github.com/stretchr/testify/assert"
)

func TestCharsetFromHeaders(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=iso-8859-9")
		_, _ = fmt.Fprint(w, "G\xfcltekin")
	}))
	defer ts.Close()

	req, _ := NewRequest("GET", ts.URL, nil, nil)
	res, _ := DoString(newFetcherDefault(), req)

	assert.Equal(t, "Gültekin", res)
}

func TestCharsetFromBody(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = fmt.Fprint(w, "G\xfcltekin")
	}))
	defer ts.Close()

	req, _ := NewRequest("POST", ts.URL, nil, nil)
	res, _ := DoString(newFetcherDefault(), req)

	assert.Equal(t, "Gültekin", res)
}

func TestCharsetProvidedWithRequest(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = fmt.Fprint(w, "G\xfcltekin")
	}))
	defer ts.Close()

	req, _ := NewRequest("GET", ts.URL, nil, nil)
	res, _ := DoString(newFetcherDefault(), WithRequestConfig(req, RequestConfig{Encoding: "windows-1254"}))

	assert.Equal(t, "Gültekin", res)
}

func TestRetry(t *testing.T) {
	t.Parallel()
	var times atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if times.Load() < DefaultRetryTimes {
			times.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte{226})
		}
	}))
	defer ts.Close()

	fetch := newFetcherDefault()

	for i, s := range []string{"Status code retry", "Other error retry"} {
		t.Run(s, func(t *testing.T) {
			times.Store(0)
			var req *http.Request
			if i > 0 {
				req, _ = NewRequest("GET", ts.URL, nil, map[string]string{"Location": "\x00"})
			} else {
				req, _ = NewRequest("HEAD", ts.URL, nil, nil)
			}

			res, err := fetch.Do(req)
			if err != nil {
				assert.ErrorContains(t, err, "Location")
			} else {
				assert.Equal(t, http.StatusOK, res.StatusCode)
			}
		})
	}
}

func TestDecompress(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoding := r.Header.Get("Content-Encoding")
		w.Header().Set("Content-Encoding", encoding)
		w.Header().Set("Content-Type", "text/plain")

		var bodyWriter io.WriteCloser
		switch encoding {
		case "deflate":
			bodyWriter = zlib.NewWriter(w)
		case "gzip":
			bodyWriter = gzip.NewWriter(w)
		case "br":
			bodyWriter = brotli.NewWriter(w)
		}
		defer bodyWriter.Close()

		bytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}

		_, _ = bodyWriter.Write(bytes)
	}))
	defer ts.Close()

	testCases := []struct {
		compress, want string
	}{
		{"deflate", "test1"},
		{"gzip", "test2"},
		{"br", "test3"},
	}

	fetch := newFetcherDefault()

	for _, testCase := range testCases {
		t.Run(testCase.compress, func(t *testing.T) {
			req, _ := NewRequest(http.MethodGet, ts.URL, testCase.want, map[string]string{
				"Content-Encoding": testCase.compress,
			})

			str, err := DoString(fetch, req)
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, testCase.want, str)
		})
	}
}

// newFetcherDefault creates new client with default options
func newFetcherDefault() cloudcat.Fetch {
	return NewFetcher(Options{
		MaxBodySize:    DefaultMaxBodySize,
		RetryTimes:     DefaultRetryTimes,
		RetryHTTPCodes: DefaultRetryHTTPCodes,
		Timeout:        DefaultTimeout,
		CachePolicy:    RFC2616,
	})
}

var (
	extNet = os.Getenv("EXTNET")
)

func TestFingerPrint(t *testing.T) {
	if extNet == "" {
		t.Skip("skipping external network test")
	}
	
	req, err := http.NewRequest(http.MethodGet, "https://tls.peet.ws/api/all", nil)
	assert.NoError(t, err)
	req.Header = http.Header{
		"Sec-Ch-Ua":          {`"Not.A/Brand";v="8", "Chromium";v="111", "Google Chrome";v="111"`},
		"Sec-Ch-Ua-Platform": {`"Windows"`},
		"Dnt":                {"1"},
		"User-Agent":         {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.5563.111 Safari/537.36"},
		"Accept":             {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"},
		"Sec-Fetch-Site":     {"none"},
		"Sec-Fetch-Mode":     {"navigate"},
		"Sec-Fetch-User":     {"?1"},
		"Sec-Fetch-Dest":     {"document"},
		"Accept-Encoding":    {"gzip, deflate, br"},
		"Accept-Language":    {"en,en_US;q=0.9"},
	}

	h2 := http2.ConfigureTransports(http.DefaultTransport.(*http.Transport), http2.Options{
		HeaderOrder: []string{
			"sec-ch-ua", "sec-ch-ua-platform", "dnt",
			"user-agent", "accept", "sec-fetch-site",
			"sec-fetch-mode", "sec-fetch-user", "sec-fetch-dest",
			"accept-encoding", "accept-language",
		},
		PHeaderOrder: []string{":method", ":authority", ":scheme", ":path"},
		Settings: []http2.Setting{
			{ID: http2.SettingHeaderTableSize, Val: 65536},
			{ID: http2.SettingEnablePush, Val: 0},
			{ID: http2.SettingMaxConcurrentStreams, Val: 1000},
			{ID: http2.SettingInitialWindowSize, Val: 6291456},
			{ID: http2.SettingMaxHeaderListSize, Val: 262144},
		},
		WindowSizeIncrement: 15663105,
		GetTlsClientHelloSpec: func() *tls.ClientHelloSpec {
			spec, _ := tls.UTLSIdToSpec(tls.HelloChrome_102)
			return &spec
		},
	})

	res, err := h2.RoundTrip(req)
	assert.NoError(t, err)
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	assert.NoError(t, err)
	var data map[string]any
	assert.NoError(t, json.Unmarshal(b, &data))

	assert.Equal(t, data["http_version"], http2.NextProtoTLS)

	if fp, ok := data["tls"].(map[string]any); ok {
		assert.Equal(t, fp["ja3"], "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-17513-21,29-23-24,0")
		assert.Equal(t, fp["ja3_hash"], "cd08e31494f9531f560d64c695473da9")
		assert.Equal(t, fp["peetprint"], "GREASE-772-771|2-1.1|GREASE-29-23-24|1027-2052-1025-1283-2053-1281-2054-1537|1|2|GREASE-4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53|0-10-11-13-16-17513-18-21-23-27-35-43-45-5-51-65281-GREASE-GREASE")
		assert.Equal(t, fp["peetprint_hash"], "22a4f858cc83b9144c829ca411948a88")
	} else {
		assert.False(t, ok, data)
	}
	if fp, ok := data["http2"].(map[string]any); ok {
		assert.Equal(t, fp["akamai_fingerprint"], "1:65536,2:0,3:1000,4:6291456,6:262144|15663105|0|m,a,s,p")
		assert.Equal(t, fp["akamai_fingerprint_hash"], "46cedabdca2073198a42fa10ca4494d0")
	} else {
		assert.False(t, ok, data)
	}
}
