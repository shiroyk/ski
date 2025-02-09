package http

import "github.com/grafana/sobek"

func reject(rt *sobek.Runtime, err any) sobek.Value {
	promise, _, rejectFn := rt.NewPromise()
	_ = rejectFn(err)
	return rt.ToValue(promise)
}
