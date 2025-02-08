package http

import (
	"errors"
	"io"
	"net/http"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski"
	"github.com/shiroyk/ski/js"
)

// Fetch the global fetch() method starts the process of
// fetching a resource from the network, returning a promise
// which is fulfilled once the response is available.
// https://developer.mozilla.org/en-US/docs/Web/API/fetch
type Fetch struct{ ski.Fetch }

func (fetch *Fetch) fetch(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
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
			method: "GET",
			cache:  "default",
			url:    resource.String(),
			body:   io.NopCloser(http.NoBody),
		}
		initRequest(rt, call.Argument(1), req)
	}

	return rt.ToValue(js.NewPromise(rt, func() (*http.Response, error) {
		defer req.cancel()
		return fetch.Do(req.toRequest(rt))
	}, func(res *http.Response, err error) (any, error) {
		if err != nil {
			return nil, err
		}
		return NewResponse(rt, res, true), nil
	}))
}

func (fetch *Fetch) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	if fetch.Fetch == nil {
		return nil, errors.New("fetch can not be nil")
	}
	proto := rt.NewObject()
	_ = proto.SetSymbol(sobek.SymToStringTag, func(sobek.FunctionCall) sobek.Value { return rt.ToValue("fetch") })
	object := rt.ToValue(fetch.fetch).ToObject(rt)
	_ = object.Set("prototype", proto)
	return object, nil
}

func (*Fetch) Global() {}
