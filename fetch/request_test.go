package fetch

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"text/template"

	"github.com/shiroyk/cloudcat/cache"
	"github.com/shiroyk/cloudcat/cache/memory"
	"github.com/stretchr/testify/assert"
)

func TestNewRequest(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=iso-8859-9")
		switch r.Method {
		case http.MethodPut:
			if token := r.Header.Get("Authorization"); token != "1919810" {
				t.Errorf("unexpected token %s", token)
			}
		case http.MethodGet:
			_, err := fmt.Fprint(w, "114514")
			if err != nil {
				t.Error(err)
			}
			return
		}

		if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
			file, _, err := r.FormFile("file")
			if err != nil {
				t.Error(err)
			}

			body, err := io.ReadAll(file)
			if err != nil {
				t.Error(err)
			}

			_, err = fmt.Fprint(w, string(body))
			if err != nil {
				t.Error(err)
			}
		} else {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
			}

			_, err = fmt.Fprint(w, string(body))
			if err != nil {
				t.Error(err)
			}
		}
	})

	fetch := newTestFetcher()

	mpBytes, mpwHeader := createMultiPart(t, map[string]any{
		"key":  "foo",
		"file": []byte{226, 153, 130, 239, 184, 142},
	})

	jsonData := struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}{Key: "foo", Value: "bar"}

	token := map[string]string{"Authorization": "1919810"}

	testCase := []struct {
		method string
		body   any
		header map[string]string
		want   string
	}{
		{http.MethodGet, nil, nil, "114514"},
		{http.MethodPost, url.Values{"key": {"holy"}}.Encode(),
			map[string]string{"Content-Type": "application/x-www-form-url"}, "key=holy"},
		{http.MethodPost, []byte{226, 153, 130, 239, 184, 142}, nil, "♂︎"},
		{http.MethodPost, strings.NewReader("fa"), nil, "fa"},
		{http.MethodPost, bytes.NewBuffer(mpBytes), mpwHeader, "♂︎"},
		{http.MethodPost, bytes.NewReader(mpBytes), mpwHeader, "♂︎"},
		{http.MethodPost, jsonData, nil, `{"key":"foo","value":"bar"}`},
		{http.MethodPut, jsonData, token, `{"key":"foo","value":"bar"}`},
	}

	for _, useTLS := range []bool{false, true} {
		var ts *httptest.Server
		if useTLS {
			ts = httptest.NewTLSServer(h)
			fetch.Client = ts.Client()
		} else {
			ts = httptest.NewServer(h)
		}

		t.Run(fmt.Sprintf("useTLS=%v", useTLS), func(t *testing.T) {
			defer ts.Close()
			for _, r := range testCase {
				switch b := r.body.(type) {
				// rewrite bytes
				case *bytes.Buffer:
					b.Write(mpBytes)
				case *bytes.Reader:
					b.Reset(mpBytes)
				case *strings.Reader:
					b.Reset("fa")
				}
				req, err := NewRequest(r.method, ts.URL, r.body, r.header)
				if err != nil {
					t.Error(err)
				}

				res, err := fetch.DoRequest(req)
				if err != nil {
					t.Error(err)
					continue
				}
				assert.Equal(t, r.want, res.String())
			}
		})
	}
}

var templateTestCase = []struct{ template, want string }{
	{`CONNECT {{.url}}`, ""},
	{`GET {{.url}} HTTP/0.0`, ""},
	{`{{.url}}?page=1`, "page=1"},
	{`{{.url}}{{if gt .page 1}}?page={{.page}}{{end}}`, "page=2"},
	{`{{.url}}?key={{.data.key}}`, "key=foo"},
	{`POST {{.url}}
Content-Type: application/json

{{ get "json" }}`, `{"key":"foo"}`},
	{`POST {{.url}}
Content-Type: application/x-www-form-urlencoded

{{ get "form" }}`, `foo`},
	{`POST {{.url}} HTTP/2.0
Pragma: no-cache
Content-Type: application/octet-stream
Connection: close

{{ get "image" }}`, "image/png"},
	{`POST {{.url}} HTTP/1.0
Content-Type: multipart/form-data; boundary=X-1145141919810

--X-1145141919810
Content-Disposition: form-data; name="key"

foo
--X-1145141919810
Content-Disposition: form-data; name="file"; filename="test.png"
Content-Type: image/png

{{ get "image" }}
--X-1145141919810--`, "foo-test.png-image/png"},
}

func TestNewTemplateRequest(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		var body []byte
		contentType := r.Header.Get("Content-Type")
		switch contentType {
		case "application/octet-stream":
			b, _ := io.ReadAll(r.Body)
			body = []byte(http.DetectContentType(b))
		case "application/x-www-form-urlencoded":
			body = []byte(r.FormValue("key"))
		case "multipart/form-data; boundary=X-1145141919810":
			if err := r.ParseMultipartForm(DefaultMaxBodySize); err != nil {
				t.Fatal(err)
			}
			file, fh, err := r.FormFile("file")
			if err != nil {
				t.Fatal(err)
			}
			data, _ := io.ReadAll(file)
			body = []byte(fmt.Sprintf("%s-%s-%s", r.FormValue("key"), fh.Filename, http.DetectContentType(data)))
		default:
			if r.Method == http.MethodGet {
				if err := r.ParseForm(); err != nil {
					t.Fatal(err)
				}
				body = []byte(r.Form.Encode())
			} else {
				body, _ = io.ReadAll(r.Body)
			}
		}
		_, _ = w.Write(body)
	})

	fetch := newTestFetcher()
	tplFuncs := templateFuncs()

	for _, useTLS := range []bool{false, true} {
		var ts *httptest.Server
		if useTLS {
			ts = httptest.NewTLSServer(h)
			fetch.Client = ts.Client()
		} else {
			ts = httptest.NewServer(h)
		}
		t.Run(fmt.Sprintf("useTLS=%v", useTLS), func(t *testing.T) {
			defer ts.Close()
			for _, tpl := range templateTestCase {
				req, err := NewTemplateRequest(tplFuncs, tpl.template,
					map[string]any{
						"url":  ts.URL,
						"page": 2,
						"data": map[string]any{
							"key": "foo",
						},
					})
				if err != nil {
					t.Error(err)
				}

				res, err := fetch.DoRequest(req)
				if err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, tpl.want, res.String())
				assert.Equal(t, "text/plain", res.ContentType())
			}
		})
	}
}

func TestNewTemplateErrRequest(t *testing.T) {
	testCase := []string{
		"  ERROR",
		"ERROR ERROR ERROR",
		`/
Host: http://localhost
Host: http://localhost/home`,
		`{{}{}`,
	}
	for _, tpl := range testCase {
		_, err := NewTemplateRequest(nil, tpl, nil)
		if err == nil {
			t.Errorf("error must be nil but got %v", err)
		}
	}
}

func BenchmarkNewTemplateRequest(b *testing.B) {
	request := `POST / HTTP/1.1
Host: localhost:8080
Connection: keep-alive
Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_8_2) AppleWebKit/537.17 (KHTML, like Gecko) Chrome/24.0.1312.52 Safari/537.17
Accept-Encoding: gzip,deflate,sdch
Accept-Language: en-US,en;q=0.8
Accept-Charset: ISO-8859-1,utf-8;q=0.7,*;q=0.3
Cookie: __utma=1.1978842379.1323102373.1323102373.1323102373.1; EPi:NumberOfVisits=1,2012-02-28T13:42:18; CrmSession=5b707226b9563e1bc69084d07a107c98; plushContainerWidth=100%25; plushNoTopMenu=0; hudson_auto_refresh=false

{{ get "image" }}
`
	tplFuncs := templateFuncs()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewTemplateRequest(tplFuncs, request, nil)
		if err != nil {
			b.Fatalf("failed to read request: %v", err)
		}
	}
}

func templateFuncs() template.FuncMap {
	memCache := memory.NewCache()
	memCache.Set("json", []byte(`{"key":"foo"}`))
	memCache.Set("form", []byte(`key=foo&value=bar`))
	memCache.Set("image", []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a})
	return template.FuncMap{
		"get": func(key string) (ret string) {
			if v, ok := memCache.Get(key); ok {
				return string(v)
			}
			return
		},
		"set": func(key string, value string) (ret string) {
			memCache.Set(key, []byte(value))
			return
		},
	}
}

func createMultiPart(t *testing.T, data map[string]any) ([]byte, map[string]string) {
	buf := &bytes.Buffer{}
	mpw := multipart.NewWriter(buf)
	for k, v := range data {
		if f, ok := v.([]byte); ok {
			// Creates a new form-data header with the provided field name and file name.
			fw, err := mpw.CreateFormFile(k, "blob")
			if err != nil {
				t.Fatal(err)
			}
			// Write bytes to the part
			if _, err := fw.Write(f); err != nil {
				t.Fatal(err)
			}
		} else {
			// Write string value
			if err := mpw.WriteField(k, fmt.Sprintf("%v", v)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := mpw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes(), map[string]string{"Content-Type": mpw.FormDataContentType()}
}

func newTestFetcher() *fetcher {
	return NewFetcher(Options{
		MaxBodySize:    DefaultMaxBodySize,
		RetryTimes:     DefaultRetryTimes,
		RetryHTTPCodes: DefaultRetryHTTPCodes,
		Timeout:        DefaultTimeout,
		CachePolicy:    cache.RFC2616,
	}).(*fetcher)
}
