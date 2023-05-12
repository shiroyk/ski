// Package http the http JS implementation
package http

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
	"text/template"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/core"
	"github.com/shiroyk/cloudcat/core/js"
	"github.com/shiroyk/cloudcat/fetch"
	"github.com/shiroyk/cloudcat/plugin/jsmodule"
	"github.com/spf13/cast"
)

// Module js module
type Module struct{}

// Exports returns module instance
func (*Module) Exports() any {
	return &Http{cloudcat.MustResolve[cloudcat.Fetch]()}
}

func init() {
	jsmodule.Register("http", &Module{})
	jsmodule.Register("FormData", &NativeFormData{})
	jsmodule.Register("URLSearchParams", &NativeURLSearchParams{})
}

// Http module for fetching resources (including across the network).
type Http struct { //nolint
	fetch cloudcat.Fetch
}

// handleBody process the send request body and set the content-type
func handleBody(body any, header map[string]string) (any, error) {
	switch data := body.(type) {
	case FormData:
		buf := &bytes.Buffer{}
		mpw := multipart.NewWriter(buf)
		for k, v := range data.data {
			for _, ve := range v {
				if f, ok := ve.(FileData); ok {
					// Creates a new form-data header with the provided field name and file name.
					fw, err := mpw.CreateFormFile(k, f.Filename)
					if err != nil {
						return nil, err
					}
					// Write bytes to the part
					if _, err := fw.Write(f.Data); err != nil {
						return nil, err
					}
				} else {
					// Write string value
					if err := mpw.WriteField(k, fmt.Sprintf("%v", v)); err != nil {
						return nil, err
					}
				}
			}
		}
		header["Content-Type"] = mpw.FormDataContentType()
		if err := mpw.Close(); err != nil {
			return nil, err
		}
		return buf, nil
	case URLSearchParams:
		header["Content-Type"] = "application/x-www-form-url"
		return data.encode(), nil
	case goja.ArrayBuffer:
		return data.Bytes(), nil
	case []byte, map[string]any, string, nil:
		return body, nil
	default:
		return nil, fmt.Errorf("unsupported request body type %v", body)
	}
}

// Get Make a GET request with URL and optional headers.
func (h *Http) Get(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	u := call.Argument(0).String()
	header := cast.ToStringMapString(call.Argument(1).Export())
	return h.doRequest(http.MethodGet, u, nil, header, vm)
}

// Post Make a POST request with URL, optional body, optional headers.
// Send POST with multipart:
// http.post(url, new FormData({'bytes': new Uint8Array([0]).buffer}))
// Send POST with x-www-form-urlencoded:
// http.post(url, new URLSearchParams({'key': 'foo', 'value': 'bar'}))
// Send POST with json:
// http.post(url, {'key': 'foo'})
func (h *Http) Post(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	u := call.Argument(0).String()
	body := call.Argument(1)
	header := cast.ToStringMapString(call.Argument(2).Export())
	return h.doRequest(http.MethodPost, u, body, header, vm)
}

// Head Make a HEAD request with URL and optional headers.
func (h *Http) Head(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	u := call.Argument(0).String()
	header := cast.ToStringMapString(call.Argument(1).Export())
	return h.doRequest(http.MethodGet, u, nil, header, vm)
}

// Request Make a request with method and URL, optional body, optional headers.
func (h *Http) Request(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	method := call.Argument(0).String()
	u := call.Argument(1).String()
	body := call.Argument(2)
	header := cast.ToStringMapString(call.Argument(3).Export())
	return h.doRequest(strings.ToLower(method), u, body, header, vm)
}

// Template Make a request with an HTTP template, template argument.
func (h *Http) Template(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	funcs, _ := cloudcat.Resolve[template.FuncMap]()
	tpl := call.Argument(0).String()
	arg := cast.ToStringMap(call.Argument(1).Export())

	req, err := fetch.NewTemplateRequest(funcs, tpl, arg)
	if err != nil {
		js.Throw(vm, err)
	}

	res, err := h.fetch.Do(req.WithContext(js.VMContext(vm)))
	if err != nil {
		js.Throw(vm, err)
	}

	return NewResponse(vm, res)
}

func (h *Http) doRequest(
	method, u string,
	reqBody goja.Value,
	header map[string]string,
	vm *goja.Runtime,
) goja.Value {
	var body any
	var err error

	if reqBody != nil && !goja.IsUndefined(reqBody) {
		if header == nil {
			header = make(map[string]string)
		}
		body, err = js.Unwrap(reqBody)
		if err != nil {
			js.Throw(vm, err)
		}
		body, err = handleBody(body, header)
		if err != nil {
			js.Throw(vm, err)
		}
	}

	req, err := fetch.NewRequest(method, u, body, header)
	if err != nil {
		js.Throw(vm, err)
	}

	res, err := h.fetch.Do(req.WithContext(js.VMContext(vm)))
	if err != nil {
		js.Throw(vm, err)
	}

	return NewResponse(vm, res)
}
