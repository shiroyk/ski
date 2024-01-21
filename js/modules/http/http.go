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
	urlpkg "net/url"
	"strings"

	"github.com/dop251/goja"
	"github.com/shiroyk/ski"
	"github.com/shiroyk/ski/js"
	"github.com/spf13/cast"
)

func init() {
	jar := ski.NewCookieJar()
	fetch := ski.NewFetch().(*http.Client)
	fetch.Jar = jar
	js.Register("cookieJar", &CookieJar{jar})
	js.Register("http", &Http{fetch})
	js.Register("fetch", &Fetch{fetch})
	js.Register("FormData", new(FormData))
	js.Register("URLSearchParams", new(URLSearchParams))
	js.Register("AbortController", new(AbortController))
	js.Register("AbortSignal", new(AbortSignal))
}

// Fetch the global Fetch() method starts the process of
// fetching a resource from the network, returning a promise
// which is fulfilled once the response is available.
// https://developer.mozilla.org/en-US/docs/Web/API/fetch
type Fetch struct{ ski.Fetch }

func (fetch *Fetch) Instantiate(rt *goja.Runtime) (goja.Value, error) {
	if fetch.Fetch == nil {
		return nil, errors.New("Fetch can not nil")
	}
	return rt.ToValue(func(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
		req, signal := buildRequest(http.MethodGet, call, vm)
		return vm.ToValue(js.NewPromise(vm,
			func() (*http.Response, error) {
				if signal != nil {
					defer signal.abort() // release resources
				}
				return fetch.Do(req)
			},
			func(res *http.Response, err error) (any, error) {
				if err != nil {
					return nil, err
				}
				return NewAsyncResponse(vm, res), nil
			}))
	}), nil
}

func (*Fetch) Global() {}

// Http module for fetching resources (including across the network).
type Http struct{ ski.Fetch }

func (h *Http) Instantiate(rt *goja.Runtime) (goja.Value, error) {
	if h.Fetch == nil {
		return nil, errors.New("Fetch can not nil")
	}
	return rt.ToValue(map[string]func(call goja.FunctionCall, vm *goja.Runtime) goja.Value{
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
func (h *Http) Get(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return h.do(call, vm, http.MethodGet)
}

// Post Make a HTTP POST.
// Send POST with multipart:
// http.post(url, { body: new FormData({'bytes': new Uint8Array([0]).buffer}) })
// Send POST with x-www-form-urlencoded:
// http.post(url, { body: new URLSearchParams({'key': 'foo', 'value': 'bar'}) })
// Send POST with json:
// http.post(url, { body: {'key': 'foo'} })
func (h *Http) Post(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return h.do(call, vm, http.MethodPost)
}

// Put Make a HTTP PUT request.
func (h *Http) Put(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return h.do(call, vm, http.MethodPut)
}

// Delete Make a HTTP DELETE request.
func (h *Http) Delete(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return h.do(call, vm, http.MethodDelete)
}

// Patch Make a HTTP PATCH request.
func (h *Http) Patch(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return h.do(call, vm, http.MethodPatch)
}

// Request Make a HTTP request.
func (h *Http) Request(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return h.do(call, vm, http.MethodGet)
}

// Head Make a HTTP HEAD request.
func (h *Http) Head(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return h.do(call, vm, http.MethodHead)
}

func (h *Http) do(call goja.FunctionCall, vm *goja.Runtime, method string) goja.Value {
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
	call goja.FunctionCall,
	vm *goja.Runtime,
) (req *http.Request, signal *abortSignal) {
	var (
		ctx     = context.Background()
		url     = call.Argument(0).String()
		options = call.Argument(1)
		opt     *goja.Object
		body    io.Reader
		headers = make(map[string]string)
		err     error
	)

	if goja.IsUndefined(options) || goja.IsNull(options) {
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
	if v := opt.Get("proxy"); v != nil {
		proxy, err := urlpkg.Parse(v.String())
		if err != nil {
			js.Throw(vm, fmt.Errorf("options proxy is invalid URL, %s", err))
		}
		ctx = ski.WithProxyURL(ctx, proxy)
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
	case goja.ArrayBuffer:
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
