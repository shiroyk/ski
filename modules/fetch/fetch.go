package fetch

import (
	"io"
	"net/http"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js/promise"
	"github.com/shiroyk/ski/modules"
)

// Fetch the global fetch() method starts the process of
// fetching a resource from the network, returning a promise
// which is fulfilled once the response is available.
// https://developer.mozilla.org/en-US/docs/Web/API/fetch
func Fetch(client Client) modules.ModuleFunc {
	return func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
		if len(call.Arguments) == 0 {
			return promise.Reject(rt, rt.NewTypeError("fetch requires at least 1 argument"))
		}
		resource := call.Argument(0)

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

		return promise.New(rt, func(callback promise.Callback) {
			defer req.cancel()
			res, err := client.Do(req.toRequest(rt))
			callback(func() (any, error) {
				if err != nil {
					return nil, err
				}
				return NewResponse(rt, res, true), nil
			})
		})
	}
}
