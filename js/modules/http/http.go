// Package http the http JS implementation
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski"
	"github.com/shiroyk/ski/js"
	"github.com/spf13/cast"
)

func init() {
	jar := NewCookieJar()
	fetch := &http.Client{
		Transport: http.DefaultTransport,
		Jar:       jar,
	}
	js.Register("cookieJar", &CookieJarModule{jar})
	js.Register("http", &Http{fetch})
	js.Register("fetch", &FetchModule{fetch})
	js.Register("FormData", new(FormData))
	js.Register("URLSearchParams", new(URLSearchParams))
	js.Register("AbortController", new(AbortController))
	js.Register("AbortSignal", new(AbortSignal))
}

// Http module for fetching resources (including across the network).
type Http struct{ ski.Fetch }

func (h *Http) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	if h.Fetch == nil {
		return nil, errors.New("fetch can not nil")
	}
	return rt.ToValue(map[string]func(call sobek.FunctionCall, vm *sobek.Runtime) sobek.Value{
		"get":     h.Get,
		"post":    h.Post,
		"put":     h.Put,
		"delete":  h.Delete,
		"patch":   h.Patch,
		"request": h.Request,
		"head":    h.Head,
	}), nil
}

// Get Make a HTTP GET request.
func (h *Http) Get(call sobek.FunctionCall, vm *sobek.Runtime) sobek.Value {
	return h.do(call, vm, http.MethodGet)
}

// Post Make a HTTP POST.
// Send POST with multipart:
// http.post(url, { body: new FormData({'bytes': new Uint8Array([0]).buffer}) })
// Send POST with x-www-form-urlencoded:
// http.post(url, { body: new URLSearchParams({'key': 'foo', 'value': 'bar'}) })
// Send POST with json:
// http.post(url, { body: {'key': 'foo'} })
func (h *Http) Post(call sobek.FunctionCall, vm *sobek.Runtime) sobek.Value {
	return h.do(call, vm, http.MethodPost)
}

// Put Make a HTTP PUT request.
func (h *Http) Put(call sobek.FunctionCall, vm *sobek.Runtime) sobek.Value {
	return h.do(call, vm, http.MethodPut)
}

// Delete Make a HTTP DELETE request.
func (h *Http) Delete(call sobek.FunctionCall, vm *sobek.Runtime) sobek.Value {
	return h.do(call, vm, http.MethodDelete)
}

// Patch Make a HTTP PATCH request.
func (h *Http) Patch(call sobek.FunctionCall, vm *sobek.Runtime) sobek.Value {
	return h.do(call, vm, http.MethodPatch)
}

// Request Make a HTTP request.
func (h *Http) Request(call sobek.FunctionCall, vm *sobek.Runtime) sobek.Value {
	return h.do(call, vm, http.MethodGet)
}

// Head Make a HTTP HEAD request.
func (h *Http) Head(call sobek.FunctionCall, vm *sobek.Runtime) sobek.Value {
	return h.do(call, vm, http.MethodHead)
}

func (h *Http) do(call sobek.FunctionCall, vm *sobek.Runtime, method string) sobek.Value {
	req, signal := buildRequest(method, call, vm)
	if signal != nil {
		defer signal.abort() // release resources
	}

	res, err := h.Do(req)
	if err != nil {
		js.Throw(vm, err)
	}

	return NewResponse(vm, res)
}

func buildRequest(
	method string,
	call sobek.FunctionCall,
	vm *sobek.Runtime,
) (req *http.Request, signal *abortSignal) {
	var (
		ctx     = context.Background()
		url     = call.Argument(0).String()
		options = call.Argument(1)
		opt     *sobek.Object
		body    io.Reader
		headers = make(map[string]string)
		err     error
	)

	if sobek.IsUndefined(options) || sobek.IsNull(options) {
		ctx = js.Context(vm)
		goto NEW
	}

	opt = options.ToObject(vm)
	if v := opt.Get("method"); v != nil {
		method = strings.ToUpper(v.String())
	}
	if v := opt.Get("headers"); v != nil {
		if headers, err = cast.ToStringMapStringE(v.Export()); err != nil {
			js.Throw(vm, fmt.Errorf("options headers is invalid, %s", err))
		}
	}
	if method != http.MethodGet && method != http.MethodHead {
		if v := opt.Get("body"); v != nil {
			if body, err = processBody(v.Export(), headers); err != nil {
				js.Throw(vm, err)
			}
		}
	}
	if v := opt.Get("cache"); v != nil {
		str := v.String()
		headers["Cache-Control"] = str
		headers["Pragma"] = str
	}
	if v := opt.Get("signal"); v != nil {
		var ok bool
		if signal, ok = v.Export().(*abortSignal); !ok {
			js.Throw(vm, errors.New("options signal is not AbortSignal"))
		}
		ctx = signal.ctx
	} else {
		ctx = js.Context(vm)
	}

NEW:
	req, err = http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		js.Throw(vm, err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return
}

// processBody process the send request body and set the content-type
func processBody(body any, headers map[string]string) (io.Reader, error) {
	switch data := body.(type) {
	case *formData:
		buf := new(bytes.Buffer)
		mpw := multipart.NewWriter(buf)
		for _, key := range data.keys {
			for _, value := range data.data[key] {
				if f, ok := value.(fileData); ok {
					// Creates a new form-data header with the provided field name and file name.
					fw, err := mpw.CreateFormFile(key, f.filename)
					if err != nil {
						return nil, err
					}
					// Write bytes to the part
					if _, err = fw.Write(f.data); err != nil {
						return nil, err
					}
				} else {
					// Write string value
					if err := mpw.WriteField(key, fmt.Sprintf("%v", key)); err != nil {
						return nil, err
					}
				}
			}
		}
		headers["Content-Type"] = mpw.FormDataContentType()
		if err := mpw.Close(); err != nil {
			return nil, err
		}
		return buf, nil
	case *urlSearchParams:
		headers["Content-Type"] = "application/x-www-form-url"
		return strings.NewReader(data.encode()), nil
	case string:
		return strings.NewReader(data), nil
	case sobek.ArrayBuffer:
		return bytes.NewReader(data.Bytes()), nil
	case []byte:
		return bytes.NewReader(data), nil
	case map[string]any:
		headers["Content-Type"] = "application/json"
		marshal, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		return bytes.NewReader(marshal), nil
	case nil:
		return http.NoBody, nil
	default:
		return nil, fmt.Errorf("unsupported request body type %T", body)
	}
}
