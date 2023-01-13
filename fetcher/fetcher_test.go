package fetcher

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/shiroyk/cloudcat/cache/memory"
)

func TestCharsetFromHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=iso-8859-9")
		_, _ = fmt.Fprint(w, "G\xfcltekin")
	}))
	defer ts.Close()

	res, _ := newFetcherDefault().Get(ts.URL, nil)

	if res.String() != "Gültekin" {
		t.Fatal(res.String())
	}
}

func TestCharsetFromBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = fmt.Fprint(w, "G\xfcltekin")
	}))
	defer ts.Close()

	res, _ := newFetcherDefault().Post(ts.URL, nil, nil)

	if res.String() != "Gültekin" {
		t.Fatal(res.String())
	}
}

func TestCharsetProvidedWithRequest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = fmt.Fprint(w, "G\xfcltekin")
	}))
	defer ts.Close()

	req, _ := NewRequest("GET", ts.URL, nil, nil)
	req.Encoding = "windows-1254"
	res, _ := newFetcherDefault().DoRequest(req)

	if res.String() != "Gültekin" {
		t.Fatal(res.String())
	}
}

func TestRetry(t *testing.T) {
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
	fetch.opt.CharsetDetectDisabled = true

	for i, s := range []string{"Status code retry", "Other error retry"} {
		t.Run(s, func(t *testing.T) {
			var res *Response
			var err error
			if i > 0 {
				res, err = fetch.Get(ts.URL, map[string]string{"Location": "\x00"})
			} else {
				res, err = fetch.Head(ts.URL, nil)
			}

			if err != nil {
				if !strings.Contains(err.Error(), "Location") {
					t.Fatal(err)
				}
			} else {
				if res.StatusCode != http.StatusOK {
					t.Fatalf("unexpected response status %v", res.StatusCode)
				}
			}
		})
	}
}

func TestCancel(t *testing.T) {
	fetch := newFetcherDefault()

	req, err := NewRequest(http.MethodGet, "", nil, nil)
	if err != nil {
		t.Error(err)
	}

	req.Cancel()

	_, err = fetch.DoRequest(req)
	if err != RequestCancel {
		t.Fatal(err)
	}
}

// newFetcherDefault creates new client with default options
func newFetcherDefault() *Fetcher {
	return NewFetcher(&Options{
		Cookie:         memory.NewCookie(),
		Cache:          memory.NewCache(),
		MaxBodySize:    DefaultMaxBodySize,
		RetryTimes:     DefaultRetryTimes,
		RetryHTTPCodes: DefaultRetryHTTPCodes,
		Timeout:        DefaultTimeout,
	})
}
