package http

import (
	"errors"
	"net/http"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski"
	"github.com/shiroyk/ski/js"
)

// FetchModule the global Fetch() method starts the process of
// fetching a resource from the network, returning a promise
// which is fulfilled once the response is available.
// https://developer.mozilla.org/en-US/docs/Web/API/fetch
type FetchModule struct{ ski.Fetch }

func (fetch *FetchModule) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	if fetch.Fetch == nil {
		return nil, errors.New("Fetch can not nil")
	}
	return rt.ToValue(func(call sobek.FunctionCall, vm *sobek.Runtime) sobek.Value {
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

func (*FetchModule) Global() {}
