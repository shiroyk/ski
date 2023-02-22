package http

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"text/template"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/fetch"
	"github.com/shiroyk/cloudcat/js/common"
	"github.com/shiroyk/cloudcat/js/modules"
	"github.com/spf13/cast"
)

// Module js module
type Module struct{}

// Exports returns module instance
func (*Module) Exports() any {
	return &Http{di.MustResolve[fetch.Fetch]()}
}

func init() {
	modules.Register("http", &Module{})
	modules.Register("FormData", &NativeFormData{})
	modules.Register("URLSearchParams", &NativeURLSearchParams{})
}

// Http module for fetching resources (including across the network).
type Http struct { //nolint:var-naming
	fetch fetch.Fetch
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
	case []byte, map[string]any, string, nil:
		return body, nil
	default:
		return nil, fmt.Errorf("unsupported request body type %v", body)
	}
}

// Get Make a GET request with URL and optional headers.
func (h *Http) Get(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	header := cast.ToStringMapString(call.Argument(1).Export())

	res, err := h.fetch.Get(call.Argument(0).String(), header)
	if err != nil {
		common.Throw(vm, err)
	}

	return vm.ToValue(NewResponse(res))
}

// Post Make a POST request with URL, optional body, optional headers.
// Send POST with multipart:
// http.post(url, new FormData({'bytes': new Uint8Array([0]).buffer}))
// Send POST with x-www-form-urlencoded:
// http.post(url, new URLSearchParams({'key': 'foo', 'value': 'bar'}))
// Send POST with json:
// http.post(url, {'key': 'foo'})
func (h *Http) Post(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	u := call.Argument(0).String()
	body, _ := common.Unwrap(call.Argument(1))
	header := cast.ToStringMapString(call.Argument(2).Export())

	var err error
	body, err = handleBody(body, header)
	if err != nil {
		common.Throw(vm, err)
	}

	res, err := h.fetch.Post(u, body, header)
	if err != nil {
		common.Throw(vm, err)
	}

	return vm.ToValue(NewResponse(res))
}

// Head Make a HEAD request with URL and optional headers.
func (h *Http) Head(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	header := cast.ToStringMapString(call.Argument(1).Export())

	res, err := h.fetch.Head(call.Argument(0).String(), header)
	if err != nil {
		common.Throw(vm, err)
	}

	return vm.ToValue(NewResponse(res))
}

// Request Make a request with method and URL, optional body, optional headers.
func (h *Http) Request(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	method := call.Argument(0).String()
	u := call.Argument(1).String()
	body, _ := common.Unwrap(call.Argument(2))
	header := cast.ToStringMapString(call.Argument(3).Export())

	var err error
	body, err = handleBody(body, header)
	if err != nil {
		common.Throw(vm, err)
	}

	res, err := h.fetch.Request(method, u, body, header)
	if err != nil {
		common.Throw(vm, err)
	}

	return vm.ToValue(NewResponse(res))
}

// Template Make a request with an HTTP template, template argument.
func (h *Http) Template(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	tpl := call.Argument(0).String()
	arg := cast.ToStringMap(call.Argument(1).Export())
	funcs, _ := di.Resolve[template.FuncMap]()

	req, err := fetch.NewTemplateRequest(funcs, tpl, arg)
	if err != nil {
		common.Throw(vm, err)
	}

	res, err := h.fetch.DoRequest(req)
	if err != nil {
		common.Throw(vm, err)
	}

	return vm.ToValue(NewResponse(res))
}
