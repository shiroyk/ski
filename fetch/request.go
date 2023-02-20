package fetch

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"text/template"

	"github.com/shiroyk/cloudcat/cache"
	"github.com/shiroyk/cloudcat/di"
	"golang.org/x/net/http/httpguts"
)

// Request is a small wrapper around *http.Request
type Request struct {
	*http.Request

	// Proxy on this Request
	Proxy []string

	// Optional response body encoding. Leave empty for automatic detection.
	// If you're having issues with auto-detection, set this.
	Encoding string

	// Set this true to cancel Request. Should be used on middlewares.
	Cancelled bool

	retryCounter int
}

// WithContext returns a shallow copy of r with its context changed
// to ctx. The provided ctx must be non-nil.
//
// For outgoing client request, the context controls the entire
// lifetime of a request and its response: obtaining a connection,
// sending the request, and reading the response headers and body.
//
// To create a new request with a context, use NewRequestWithContext.
// To change the context of a request, such as an incoming request you
// want to modify before sending back out, use Request.Clone. Between
// those two uses, it's rare to need WithContext.
func (r *Request) WithContext(ctx context.Context) *Request {
	r.Request = r.Request.WithContext(ctx)
	return r
}

// Cancel request.
func (r *Request) Cancel() {
	r.Cancelled = true
}

// NewRequest returns a new Request given a method, URL, optional body, optional headers.
func NewRequest(method, u string, body any, headers map[string]string) (*Request, error) {
	var reqBody io.Reader = http.NoBody
	if body != nil {
		// Convert body to io.Reader
		switch data := body.(type) {
		default:
			kind := reflect.ValueOf(body).Kind()
			if kind != reflect.Struct && kind != reflect.Map {
				break
			}

			j, err := json.Marshal(body)
			if err != nil {
				return nil, err
			}
			if headers == nil {
				headers = make(map[string]string)
			}
			if _, ok := headers["Content-Type"]; !ok {
				headers["Content-Type"] = "application/json"
			}
			reqBody = bytes.NewReader(j)
		case *bytes.Buffer:
			reqBody = data
		case *bytes.Reader:
			reqBody = data
		case *strings.Reader:
			reqBody = data
		case string:
			reqBody = bytes.NewBufferString(data)
		case []byte:
			reqBody = bytes.NewBuffer(data)
		}
	}

	req, err := http.NewRequest(method, u, reqBody)
	if err != nil {
		return nil, err
	}

	if headers != nil {
		// set headers
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}
	setDefaultHeader(req.Header)

	return &Request{Request: req}, nil
}

// NewTemplateRequest returns a new Request given a http template with argument.
func NewTemplateRequest(funcs template.FuncMap, tpl string, arg any) (*Request, error) {
	tmp, err := template.New("url").Funcs(funcs).Parse(tpl)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	if err := tmp.Execute(buf, arg); err != nil {
		return nil, err
	}

	tp := newTextprotoReader(bufio.NewReader(buf))

	// First line: GET /index.html HTTP/1.0
	var s string
	if s, err = tp.ReadLine(); err != nil {
		return nil, err
	}
	defer func() {
		putTextprotoReader(tp)
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
	}()

	req := new(http.Request)
	var rawURI string

	req.Method, rawURI, req.Proto = parseRequestLine(s)
	if !validMethod(req.Method) {
		return nil, fmt.Errorf("invalid method %s", req.Method)
	}
	var ok bool
	if req.ProtoMajor, req.ProtoMinor, ok = http.ParseHTTPVersion(req.Proto); !ok {
		return nil, fmt.Errorf("malformed HTTP version %s", req.Proto)
	}

	if req.URL, err = url.ParseRequestURI(rawURI); err != nil {
		return nil, err
	}

	// Subsequent lines: Key: value.
	mimeHeader, err := tp.ReadMIMEHeader()
	if err != nil && err != io.EOF {
		return nil, err
	}
	req.Header = http.Header(mimeHeader)
	if len(req.Header["Host"]) > 1 {
		return nil, fmt.Errorf("too many Host headers")
	}

	// RFC 7230, section 5.3: Must treat
	//	GET /index.html HTTP/1.1
	//	Host: www.google.com
	// and
	//	GET http://www.google.com/index.html HTTP/1.1
	//	Host: doesntmatter
	// the same. In the second case, any Host line is ignored.
	req.Host = req.URL.Host

	fixPragmaCacheControl(req.Header)

	req.Close = shouldClose(req.ProtoMajor, req.ProtoMinor, req.Header)

	if req.Method != http.MethodHead || req.Body == nil {
		// Read body and fix content-length
		body := new(bytes.Buffer)
		if _, err = tp.R.WriteTo(body); err != nil {
			return nil, err
		}
		if body.Len() == 0 {
			req.Body = http.NoBody
		} else {
			req.ContentLength = int64(body.Len())
			req.Body = io.NopCloser(body)
		}
	}

	setDefaultHeader(req.Header)

	return &Request{Request: req}, nil
}

// DefaultTemplateFuncMap The default template function map
func DefaultTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"get": func(key string) (ret string) {
			if v, ok := di.MustResolve[cache.Cache]().Get(key); ok {
				return string(v)
			}
			return
		},
		"set": func(key string, value string) (ret string) {
			di.MustResolve[cache.Cache]().Set(key, []byte(value))
			return
		},
	}
}

func setDefaultHeader(reqHeader http.Header) {
	for k, v := range DefaultHeaders {
		if _, ok := reqHeader[k]; !ok {
			reqHeader.Set(k, v)
		}
	}
}

var textprotoReaderPool sync.Pool

func newTextprotoReader(br *bufio.Reader) *textproto.Reader {
	if v := textprotoReaderPool.Get(); v != nil {
		tr := v.(*textproto.Reader)
		tr.R = br
		return tr
	}
	return textproto.NewReader(br)
}

func putTextprotoReader(r *textproto.Reader) {
	r.R = nil
	textprotoReaderPool.Put(r)
}

// parseRequestLine parses "GET /foo HTTP/1.1" into its three parts.
// Default proto HTTP/1.1.
func parseRequestLine(line string) (method, requestURI, proto string) {
	method, rest, ok1 := strings.Cut(line, " ")
	requestURI, proto, ok2 := strings.Cut(rest, " ")
	if !ok1 {
		// default GET request
		return "GET", line, "HTTP/1.1"
	}
	if !ok2 {
		return method, requestURI, "HTTP/1.1"
	}
	return method, requestURI, proto
}

func validMethod(method string) bool {
	/*
	     Method         = "OPTIONS"                ; Section 9.2
	                    | "GET"                    ; Section 9.3
	                    | "HEAD"                   ; Section 9.4
	                    | "POST"                   ; Section 9.5
	                    | "PUT"                    ; Section 9.6
	                    | "DELETE"                 ; Section 9.7
	                    | "TRACE"                  ; Section 9.8
	                    | "CONNECT"                ; Section 9.9
	                    | extension-method
	   extension-method = token
	     token          = 1*<any CHAR except CTLs or separators>
	*/
	return len(method) > 0 && strings.IndexFunc(method, func(r rune) bool {
		return !httpguts.IsTokenRune(r)
	}) == -1
}

// Determine whether to hang up after sending a request and body, or
// receiving a response and body
// 'header' is the request headers
func shouldClose(major, minor int, header http.Header) bool {
	if major < 1 {
		return true
	}

	conv := header["Connection"]
	hasClose := httpguts.HeaderValuesContainsToken(conv, "close")
	if major == 1 && minor == 0 {
		return hasClose || !httpguts.HeaderValuesContainsToken(conv, "keep-alive")
	}

	if hasClose {
		header.Del("Connection")
	}

	return hasClose
}

// RFC 7234, section 5.4: Should treat
//
//	Pragma: no-cache
//
// like
//
//	Cache-Control: no-cache
func fixPragmaCacheControl(header http.Header) {
	if hp, ok := header["Pragma"]; ok && len(hp) > 0 && hp[0] == "no-cache" {
		if _, present := header["Cache-Control"]; !present {
			header["Cache-Control"] = []string{"no-cache"}
		}
	}
}
