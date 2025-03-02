package fetch

import (
	"errors"
	"io"
	"net/http"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
)

// Http module for fetching resources (including across the network).
type Http struct{ Client }

func (h *Http) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	if h.Client == nil {
		return nil, errors.New("http client can not nil")
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
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("http requires at least 1 argument"))
	}
	resource := call.Argument(0)

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
