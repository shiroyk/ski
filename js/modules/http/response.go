package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/dop251/goja"
	"github.com/shiroyk/ski/js"
)

var errBodyAlreadyRead = errors.New("body stream already read")

func defineGetter(r *goja.Runtime, o *goja.Object, name string, v func() any) {
	_ = o.DefineAccessorProperty(name, r.ToValue(func(goja.FunctionCall) goja.Value {
		return r.ToValue(v())
	}), nil, goja.FLAG_FALSE, goja.FLAG_TRUE)
}

// NewResponse returns a new Response
func NewResponse(rt *goja.Runtime, res *http.Response) goja.Value {
	var bodyUsed bool
	js.OnDone(rt, func() {
		if !bodyUsed {
			res.Body.Close()
		}
	})
	readBody := func() []byte {
		if bodyUsed {
			js.Throw(rt, errBodyAlreadyRead)
		}
		bodyUsed = true
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		if err != nil {
			js.Throw(rt, err)
		}
		return data
	}

	object := rt.NewObject()
	defineGetter(rt, object, "body", func() any { return rt.NewArrayBuffer(readBody()) })
	defineGetter(rt, object, "bodyUsed", func() any { return bodyUsed })
	defineGetter(rt, object, "headers", func() any { return joinHeader(res.Header) })
	defineGetter(rt, object, "status", func() any { return res.StatusCode })
	defineGetter(rt, object, "statusText", func() any { return res.Status })
	defineGetter(rt, object, "ok", func() any {
		return res.StatusCode >= 200 && res.StatusCode < 300
	})
	_ = object.Set("text", func(goja.FunctionCall) goja.Value { return rt.ToValue(string(readBody())) })
	_ = object.Set("json", func(call goja.FunctionCall) goja.Value {
		var data any
		if err := json.Unmarshal(readBody(), &data); err != nil {
			js.Throw(rt, err)
		}
		return rt.ToValue(data)
	})
	_ = object.Set("arrayBuffer", func(goja.FunctionCall) goja.Value { return rt.ToValue(rt.NewArrayBuffer(readBody())) })
	return object
}

// NewAsyncResponse returns a new async Response
func NewAsyncResponse(rt *goja.Runtime, res *http.Response) goja.Value {
	var bodyUsed bool
	js.OnDone(rt, func() {
		if !bodyUsed {
			res.Body.Close()
		}
	})

	object := rt.NewObject()
	readBody := func() ([]byte, error) {
		if bodyUsed {
			return nil, errBodyAlreadyRead
		}
		bodyUsed = true
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		return data, nil
	}

	defineGetter(rt, object, "body", func() any {
		if bodyUsed {
			js.Throw(rt, errBodyAlreadyRead)
		}
		return NewReadableStream(res.Body, rt, &bodyUsed)
	})
	defineGetter(rt, object, "bodyUsed", func() any { return bodyUsed })
	defineGetter(rt, object, "headers", func() any { return joinHeader(res.Header) })
	defineGetter(rt, object, "status", func() any { return res.StatusCode })
	defineGetter(rt, object, "statusText", func() any { return res.Status })
	defineGetter(rt, object, "ok", func() any {
		return res.StatusCode >= 200 && res.StatusCode < 300
	})
	_ = object.Set("text", func(goja.FunctionCall) goja.Value {
		return rt.ToValue(js.NewPromise(rt, func() (any, error) {
			data, err := readBody()
			if err != nil {
				return nil, err
			}
			return string(data), nil
		}))
	})
	_ = object.Set("json", func(goja.FunctionCall) goja.Value {
		return rt.ToValue(js.NewPromise(rt, func() (any, error) {
			data, err := readBody()
			if err != nil {
				return nil, err
			}
			var j any
			if err = json.Unmarshal(data, &j); err != nil {
				return nil, err
			}
			return j, err
		}))
	})
	_ = object.Set("arrayBuffer", func(goja.FunctionCall) goja.Value {
		return rt.ToValue(js.NewPromise(rt, func() (any, error) {
			data, err := readBody()
			if err != nil {
				return nil, err
			}
			return rt.NewArrayBuffer(data), nil
		}))
	})
	return object
}

func joinHeader(header http.Header) map[string]string {
	h := make(map[string]string, len(header))
	for k, vs := range header {
		h[k] = strings.Join(vs, ", ")
	}
	return h
}

// NewReadableStream ReadableStream API
// https://developer.mozilla.org/en-US/docs/Web/API/ReadableStream
func NewReadableStream(body io.ReadCloser, vm *goja.Runtime, bodyUsed *bool) *goja.Object {
	var lock bool
	object := vm.NewObject()
	_ = object.Set("cancel", func() {
		if err := body.Close(); err != nil {
			js.Throw(vm, err)
		}
	})
	_ = object.Set("getReader", func(call goja.FunctionCall) goja.Value {
		if *bodyUsed {
			js.Throw(vm, errBodyAlreadyRead)
		}
		*bodyUsed = true
		if lock {
			js.Throw(vm, errors.New("ReadableStream locked"))
		}
		lock = true
		return NewReadableStreamDefaultReader(body, vm, &lock)
	})

	// not implement
	_ = object.Set("pipeThrough", func() {})
	_ = object.Set("pipeTo", func() {})
	_ = object.Set("tee", func() {})
	return object
}

type iter struct {
	Value goja.Value
	Done  bool
}

// NewReadableStreamDefaultReader Streams API
// https://developer.mozilla.org/en-US/docs/Web/API/ReadableStreamDefaultReader
// https://developer.mozilla.org/en-US/docs/Web/API/ReadableStreamBYOBReader
func NewReadableStreamDefaultReader(body io.ReadCloser, vm *goja.Runtime, lock *bool) *goja.Object {
	object := vm.NewObject()
	defineGetter(vm, object, "locked", func() any { return &lock })
	_ = object.Set("cancel", func() {
		if err := body.Close(); err != nil {
			js.Throw(vm, err)
		}
	})

	_ = object.Set("read", func(call goja.FunctionCall) goja.Value {
		var buffer []byte
		if goja.IsUndefined(call.Argument(0)) {
			buffer = make([]byte, 1024)
		} else {
			var view bool
			buffer, view = call.Argument(0).Export().([]byte)
			if !view {
				js.Throw(vm, errors.New("read view is not TypedArray"))
			}
		}

		return vm.ToValue(js.NewPromise(vm,
			func() (int, error) { return body.Read(buffer) },
			func(n int, err error) (any, error) {
				if err != nil {
					if errors.Is(err, io.EOF) {
						return iter{goja.Undefined(), true}, nil
					}
					return nil, err
				}
				value, err := vm.New(vm.Get("Uint8Array"), vm.ToValue(vm.NewArrayBuffer(buffer[:n])))
				if err != nil {
					js.Throw(vm, err)
				}
				return iter{value, false}, nil
			}))
	})

	_ = object.Set("releaseLock", func() { *lock = false })
	return object
}
