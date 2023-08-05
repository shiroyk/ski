package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/js"
)

func defineAccessorProperty(r *goja.Runtime, o *goja.Object, name string, v any) {
	_ = o.DefineAccessorProperty(name, r.ToValue(func(call goja.FunctionCall) goja.Value { return r.ToValue(v) }), nil, goja.FLAG_FALSE, goja.FLAG_FALSE)
}

// NewResponse returns a new Response
func NewResponse(vm *goja.Runtime, res *http.Response) goja.Value {
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		js.Throw(vm, err)
	}
	object := vm.NewObject()
	_ = object.DefineAccessorProperty("body", vm.ToValue(func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(vm.NewArrayBuffer(body))
	}), nil, goja.FLAG_FALSE, goja.FLAG_FALSE)
	defineAccessorProperty(vm, object, "bodyUsed", true)
	defineAccessorProperty(vm, object, "headers", joinHeader(res.Header))
	defineAccessorProperty(vm, object, "status", res.StatusCode)
	defineAccessorProperty(vm, object, "statusText", res.Status)
	defineAccessorProperty(vm, object, "ok", res.StatusCode >= 200 || res.StatusCode < 300)
	_ = object.Set("text", func(goja.FunctionCall) goja.Value { return vm.ToValue(string(body)) })
	_ = object.Set("json", func(goja.FunctionCall) goja.Value {
		j := make(map[string]any)
		if err = json.Unmarshal(body, &j); err != nil {
			js.Throw(vm, err)
		}
		return vm.ToValue(j)
	})
	_ = object.Set("arrayBuffer", func(goja.FunctionCall) goja.Value { return vm.ToValue(vm.NewArrayBuffer(body)) })
	return object
}

// NewAsyncResponse returns a new async Response
func NewAsyncResponse(vm *goja.Runtime, res *http.Response) goja.Value {
	var bodyUsed bool
	object := vm.NewObject()

	readBody := func() ([]byte, error) {
		if bodyUsed {
			return nil, errors.New("body is used")
		}
		bodyUsed = true
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		return data, nil
	}

	_ = object.DefineAccessorProperty("body", vm.ToValue(func(call goja.FunctionCall) goja.Value {
		return NewReadableStream(res.Body, vm, &bodyUsed)
	}), nil, goja.FLAG_FALSE, goja.FLAG_FALSE)
	defineAccessorProperty(vm, object, "bodyUsed", &bodyUsed)
	defineAccessorProperty(vm, object, "headers", joinHeader(res.Header))
	defineAccessorProperty(vm, object, "status", res.StatusCode)
	defineAccessorProperty(vm, object, "statusText", res.Status)
	defineAccessorProperty(vm, object, "ok", res.StatusCode >= 200 || res.StatusCode < 300)

	_ = object.Set("text", func(goja.FunctionCall) goja.Value {
		return vm.ToValue(js.NewPromise(vm, func() (any, error) {
			data, err := readBody()
			if err != nil {
				return nil, err
			}
			return string(data), nil
		}))
	})
	_ = object.Set("json", func(goja.FunctionCall) goja.Value {
		return vm.ToValue(js.NewPromise(vm, func() (any, error) {
			data, err := readBody()
			if err != nil {
				return nil, err
			}
			j := make(map[string]any)
			if err = json.Unmarshal(data, &j); err != nil {
				return nil, err
			}
			return j, err
		}))
	})
	_ = object.Set("arrayBuffer", func(goja.FunctionCall) goja.Value {
		return vm.ToValue(js.NewPromise(vm, func() (any, error) {
			data, err := readBody()
			if err != nil {
				return nil, err
			}
			return vm.NewArrayBuffer(data), nil
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
	if *bodyUsed {
		js.Throw(vm, errors.New("body is used"))
	}
	var lock bool
	object := vm.NewObject()
	_ = object.Set("cancel", func() {
		if err := body.Close(); err != nil {
			js.Throw(vm, err)
		}
	})
	_ = object.Set("getReader", func(call goja.FunctionCall) goja.Value {
		if *bodyUsed {
			js.Throw(vm, errors.New("body is used"))
		}
		*bodyUsed = true
		if lock {
			js.Throw(vm, errors.New("ReadableStream is locked"))
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

type chunk struct {
	Value goja.Value
	Done  bool
}

// NewReadableStreamDefaultReader Streams API
// https://developer.mozilla.org/en-US/docs/Web/API/ReadableStreamDefaultReader
// https://developer.mozilla.org/en-US/docs/Web/API/ReadableStreamBYOBReader
func NewReadableStreamDefaultReader(body io.ReadCloser, vm *goja.Runtime, lock *bool) *goja.Object {
	object := vm.NewObject()
	defineAccessorProperty(vm, object, "locked", &lock)
	_ = object.Set("cancel", func() {
		if err := body.Close(); err != nil {
			js.Throw(vm, err)
		}
	})

	_ = object.Set("read", func(call goja.FunctionCall) goja.Value {
		var buffer []byte
		var value *goja.Object
		var view bool
		if goja.IsUndefined(call.Argument(0)) {
			buffer = make([]byte, 1024)
		} else {
			buffer, view = call.Argument(0).Export().([]byte)
			if !view {
				js.Throw(vm, errors.New("read view is not TypedArray"))
			}
			value = call.Argument(0).ToObject(vm)
		}

		callback := js.NewEnqueueCallback(vm)
		promise, resolve, reject := vm.NewPromise()
		go func() {
			n, err := body.Read(buffer)
			callback(func() error {
				if err != nil {
					if errors.Is(err, io.EOF) {
						resolve(chunk{goja.Undefined(), true})
						return nil
					}
					reject(err)
				} else {
					if !view {
						buffer = buffer[:n]
						value, err = vm.New(vm.Get("Uint8Array"), vm.ToValue(&buffer))
						if err != nil {
							js.Throw(vm, err)
						}
					}
					resolve(chunk{value, false})
				}
				return nil
			})
		}()

		return vm.ToValue(promise)
	})

	_ = object.Set("releaseLock", func() { *lock = false })
	return object
}
