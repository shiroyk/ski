// Package http the http JS implementation
package http

import (
	"errors"
	"io"
	"net/http"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/modules"
)

func init() {
	jar := NewCookieJar()
	fetch := ski.NewFetch().(*http.Client)
	fetch.Jar = jar
	modules.Register("cookieJar", &CookieJarModule{jar})
	modules.Register("http", &Http{fetch})
	modules.Register("fetch", &Fetch{fetch})
	modules.Register("Blob", new(Blob))
	modules.Register("Request", new(Request))
	modules.Register("Response", new(Response))
	modules.Register("ReadableStream", new(ReadableStream))
	modules.Register("ReadableStreamBYOBReader", new(ReadableStreamBYOBReader))
	modules.Register("ReadableStreamDefaultReader", new(ReadableStreamDefaultReader))
	modules.Register("File", new(File))
	modules.Register("URL", new(URL))
	modules.Register("Headers", new(Headers))
	modules.Register("FormData", new(FormData))
	modules.Register("URLSearchParams", new(URLSearchParams))
	modules.Register("AbortController", new(AbortController))
	modules.Register("AbortSignal", new(AbortSignal))
}

// Http module for fetching resources (including across the network).
type Http struct{ ski.Fetch }

func (h *Http) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	if h.Fetch == nil {
		return nil, errors.New("fetch can not nil")
	}
	obj := rt.NewObject()
	_ = obj.Set("get", h.get)
	_ = obj.Set("post", h.post)
	_ = obj.Set("put", h.put)
	_ = obj.Set("delete", h.delete)
	_ = obj.Set("patch", h.patch)
	_ = obj.Set("request", h.request)
	_ = obj.Set("head", h.head)
	return obj, nil
}

// get Make a HTTP GET request.
func (h *Http) get(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return h.do(call, rt, http.MethodGet)
}

// post Make a HTTP POST.
// Send POST with multipart:
// http.post(url, { body: new FormData({'bytes': new Uint8Array([0])}) })
// Send POST with x-www-form-urlencoded:
// http.post(url, { body: new URLSearchParams({'key': 'foo', 'value': 'bar'}) })
// Send POST with json:
// http.post(url, { body: {'key': 'foo'} })
func (h *Http) post(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return h.do(call, rt, http.MethodPost)
}

// put Make a HTTP PUT request.
func (h *Http) put(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return h.do(call, rt, http.MethodPut)
}

// delete Make a HTTP DELETE request.
func (h *Http) delete(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return h.do(call, rt, http.MethodDelete)
}

// patch Make a HTTP PATCH request.
func (h *Http) patch(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return h.do(call, rt, http.MethodPatch)
}

// request Make a HTTP request.
func (h *Http) request(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return h.do(call, rt, http.MethodGet)
}

// head Make a HTTP HEAD request.
func (h *Http) head(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return h.do(call, rt, http.MethodHead)
}

func (h *Http) do(call sobek.FunctionCall, rt *sobek.Runtime, method string) sobek.Value {
	if len(call.Arguments) < 1 {
		panic(rt.NewTypeError("fetch requires at least 1 argument"))
	}
	resource := call.Argument(0)
	if sobek.IsUndefined(resource) {
		panic(rt.NewTypeError("fetch requires at least 1 argument"))
	}

	var req *request
	if resource.ExportType() == typeRequest {
		req = resource.Export().(*request)
	} else {
		req = &request{
			method: method,
			cache:  "default",
			url:    resource.String(),
			body:   io.NopCloser(http.NoBody),
		}
		initRequest(rt, call.Argument(1), req)
	}

	defer req.cancel()
	res, err := h.Do(req.toRequest(rt))
	if err != nil {
		js.Throw(rt, err)
	}

	return NewResponse(rt, res, false)
}
