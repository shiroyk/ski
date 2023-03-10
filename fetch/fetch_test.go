package fetch

import (
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/shiroyk/cloudcat/cache"
	"github.com/stretchr/testify/assert"
)

func TestCharsetFromHeaders(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=iso-8859-9")
		_, _ = fmt.Fprint(w, "G\xfcltekin")
	}))
	defer ts.Close()

	res, err := newFetcherDefault().Get(ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "Gültekin", res.String())
}

func TestCharsetFromBody(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = fmt.Fprint(w, "G\xfcltekin")
	}))
	defer ts.Close()

	res, _ := newFetcherDefault().Post(ts.URL, nil, nil)

	assert.Equal(t, "Gültekin", res.String())
}

func TestCharsetProvidedWithRequest(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = fmt.Fprint(w, "G\xfcltekin")
	}))
	defer ts.Close()

	req, _ := NewRequest("GET", ts.URL, nil, nil)
	req.Encoding = "windows-1254"
	res, _ := newFetcherDefault().DoRequest(req)

	assert.Equal(t, "Gültekin", res.String())
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
			var res *Response
			var err error
			if i > 0 {
				res, err = fetch.Get(ts.URL, map[string]string{"Location": "\x00"})
			} else {
				res, err = fetch.Head(ts.URL, nil)
			}

			if err != nil {
				assert.ErrorContains(t, err, "Location")
			} else {
				assert.Equal(t, http.StatusOK, res.StatusCode)
			}
		})
	}
}

func TestCancel(t *testing.T) {
	t.Parallel()
	fetch := newFetcherDefault()

	req, err := NewRequest(http.MethodGet, "", nil, nil)
	if err != nil {
		t.Error(err)
	}

	req.Cancel()

	_, err = fetch.DoRequest(req)
	assert.ErrorIs(t, err, ErrRequestCancel)
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
			res, err := fetch.Post(ts.URL, testCase.want, map[string]string{
				"Content-Encoding": testCase.compress,
			})
			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, testCase.want, res.String())
		})
	}
}

// newFetcherDefault creates new client with default options
func newFetcherDefault() Fetch {
	return NewFetcher(Options{
		MaxBodySize:    DefaultMaxBodySize,
		RetryTimes:     DefaultRetryTimes,
		RetryHTTPCodes: DefaultRetryHTTPCodes,
		Timeout:        DefaultTimeout,
		CachePolicy:    cache.RFC2616,
	})
}
