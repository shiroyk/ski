package http

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
)

// Headers allows you to perform various actions on HTTP request and response headers.
// These actions include retrieving, setting, adding to, and removing headers from the
// list of the request's headers.
// https://developer.mozilla.org/en-US/docs/Web/API/Headers
type Headers struct{}

func (h *Headers) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()
	_ = p.Set("append", h.append)
	_ = p.Set("delete", h.delete)
	_ = p.Set("get", h.get)
	_ = p.Set("has", h.has)
	_ = p.Set("set", h.set)
	_ = p.Set("forEach", h.forEach)
	_ = p.Set("entries", h.entries)
	_ = p.Set("keys", h.keys)
	_ = p.Set("values", h.values)
	_ = p.SetSymbol(sobek.SymIterator, h.entries)
	_ = p.SetSymbol(sobek.SymToStringTag, func(sobek.FunctionCall) sobek.Value { return rt.ToValue("Headers") })
	return p
}

func (h *Headers) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	header := make(headers)
	if init := call.Argument(0); !sobek.IsUndefined(init) {
		switch {
		case init.ExportType() == typeHeaders:
			other := init.Export().(headers)
			header = headers(http.Header.Clone(http.Header(other)))
		default:
			obj := init.ToObject(rt)
			for _, key := range obj.Keys() {
				value := obj.Get(key)
				if !sobek.IsUndefined(value) {
					header[normalizeHeaderName(key)] = []string{value.String()}
				}
			}
		}
	}

	obj := rt.ToValue(header).(*sobek.Object)
	_ = obj.SetPrototype(call.This.Prototype())
	return obj
}

func (*Headers) append(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toHeaders(rt, call.This)
	name := normalizeHeaderName(call.Argument(0).String())
	value := call.Argument(1).String()

	this[name] = append(this[name], value)
	return sobek.Undefined()
}

func (*Headers) delete(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toHeaders(rt, call.This)
	name := normalizeHeaderName(call.Argument(0).String())
	delete(this, name)
	return sobek.Undefined()
}

func (*Headers) get(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toHeaders(rt, call.This)
	name := normalizeHeaderName(call.Argument(0).String())
	if values := this[name]; len(values) > 0 {
		return rt.ToValue(strings.Join(values, ", "))
	}
	return sobek.Null()
}

func (*Headers) has(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toHeaders(rt, call.This)
	name := normalizeHeaderName(call.Argument(0).String())
	_, ok := this[name]
	return rt.ToValue(ok)
}

func (*Headers) set(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toHeaders(rt, call.This)
	name := normalizeHeaderName(call.Argument(0).String())
	value := call.Argument(1).String()
	this[name] = []string{value}
	return sobek.Undefined()
}

func (*Headers) forEach(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toHeaders(rt, call.This)
	callback, ok := sobek.AssertFunction(call.Argument(0))
	if !ok {
		panic(rt.NewTypeError("callback is not a function"))
	}

	for name, values := range this {
		value := strings.Join(values, ", ")
		_, err := callback(call.Argument(0), rt.ToValue(value), rt.ToValue(name), call.This)
		if err != nil {
			js.Throw(rt, err)
		}
	}
	return sobek.Undefined()
}

func (*Headers) entries(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toHeaders(rt, call.This)
	return js.Iterator(rt, func(yield func(any) bool) {
		for key, value := range this {
			if !yield(rt.NewArray(key, strings.Join(value, ", "))) {
				return
			}
		}
	})
}

func (*Headers) keys(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toHeaders(rt, call.This)
	return js.Iterator(rt, func(yield func(any) bool) {
		for key := range this {
			if !yield(key) {
				return
			}
		}
	})
}

func (*Headers) values(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toHeaders(rt, call.This)
	return js.Iterator(rt, func(yield func(any) bool) {
		for _, value := range this {
			if !yield(value) {
				return
			}
		}
	})
}

func (h *Headers) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	proto := h.prototype(rt)
	ctor := rt.ToValue(h.constructor).(*sobek.Object)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.Set("prototype", proto)
	_ = ctor.SetPrototype(proto)
	return ctor, nil
}

func (*Headers) Global() {}

type headers map[string][]string

func normalizeHeaderName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func toHeaders(rt *sobek.Runtime, value sobek.Value) headers {
	if value.ExportType() == typeHeaders {
		return value.Export().(headers)
	}
	panic(rt.NewTypeError(`Value of "this" must be of type Headers`))
}

var typeHeaders = reflect.TypeOf((headers)(nil))
