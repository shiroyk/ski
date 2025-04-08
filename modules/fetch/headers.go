package fetch

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/types"
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
	var header Header
	if init := call.Argument(0); !sobek.IsUndefined(init) {
		switch init.ExportType() {
		case typeHeaders:
			h2 := init.Export().(Header)
			header = make(Header, len(h2))
			for k, v := range h2 {
				name := NormalizeHeaderName(k)
				for _, vv := range v {
					header[name] = append(header[name], NormalizeHeaderValue(vv))
				}
			}
		default:
			obj := init.ToObject(rt)
			if obj.GetSymbol(sobek.SymIterator) != nil {
				if v := obj.Get("length"); v != nil {
					header = make(Header, v.ToInteger())
				} else {
					header = make(Header)
				}
				rt.ForOf(obj, func(v sobek.Value) bool {
					item := v.ToObject(rt)
					if length := item.Get("length"); length == nil || length.ToInteger() != 2 {
						panic(rt.NewTypeError(" The provided value cannot be converted to a sequence"))
					}
					key := item.Get("0").String()
					value := item.Get("1").String()
					name := NormalizeHeaderName(key)
					value = NormalizeHeaderValue(value)
					header[name] = append(header[name], value)
					return true
				})
			} else {
				if obj.ExportType().Kind() != reflect.Map {
					panic(rt.NewTypeError("The provided value is not an object"))
				}
				keys := obj.Keys()
				header = make(Header, len(keys))
				for _, key := range keys {
					name := NormalizeHeaderName(key)
					value := NormalizeHeaderValue(obj.Get(key).String())
					header[name] = append(header[name], value)
				}
			}
		}
	} else {
		header = make(Header)
	}

	obj := rt.ToValue(header).(*sobek.Object)
	_ = obj.SetPrototype(call.This.Prototype())
	return obj
}

// append appends a header.
func (*Headers) append(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if len(call.Arguments) < 2 {
		panic(rt.NewTypeError("Failed to execute 'append' on 'Headers': 2 arguments required, but only %d present", len(call.Arguments)))
	}

	this := toHeaders(rt, call.This)
	name := NormalizeHeaderName(call.Argument(0).String())
	value := NormalizeHeaderValue(call.Argument(1).String())
	this[name] = append(this[name], value)
	return sobek.Undefined()
}

// delete deletes a header.
func (*Headers) delete(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toHeaders(rt, call.This)
	name := NormalizeHeaderName(call.Argument(0).String())
	delete(this, name)
	return sobek.Undefined()
}

// get gets a header.
func (*Headers) get(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toHeaders(rt, call.This)
	name := NormalizeHeaderName(call.Argument(0).String())
	if values := this[name]; len(values) > 0 {
		return rt.ToValue(strings.Join(values, ", "))
	}
	return sobek.Null()
}

// has checks if a header exists.
func (*Headers) has(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toHeaders(rt, call.This)
	name := NormalizeHeaderName(call.Argument(0).String())
	_, ok := this[name]
	return rt.ToValue(ok)
}

// set sets a header.
func (*Headers) set(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if len(call.Arguments) < 2 {
		panic(rt.NewTypeError("Failed to execute 'set' on 'Headers': 2 arguments required, but only %d present", len(call.Arguments)))
	}

	this := toHeaders(rt, call.This)
	name := NormalizeHeaderName(call.Argument(0).String())
	value := NormalizeHeaderValue(call.Argument(1).String())
	this[name] = []string{value}
	return sobek.Undefined()
}

// forEach calls a callback for each key, value in the Headers.
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

// entries returns an iterator of key, value in the Headers.
func (*Headers) entries(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toHeaders(rt, call.This)
	return types.Iterator(rt, func(yield func(any) bool) {
		for key, values := range this {
			if key == "set-cookie" {
				for _, value := range values {
					if !yield(rt.NewArray(key, value)) {
						return
					}
				}
			} else {
				if !yield(rt.NewArray(key, strings.Join(values, ", "))) {
					return
				}
			}
		}
	})
}

// keys returns an iterator of keys in the Headers.
func (*Headers) keys(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toHeaders(rt, call.This)
	return types.Iterator(rt, func(yield func(any) bool) {
		for key := range this {
			if !yield(key) {
				return
			}
		}
	})
}

// values returns an iterator of values in the Headers.
func (*Headers) values(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toHeaders(rt, call.This)
	return types.Iterator(rt, func(yield func(any) bool) {
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

type Header map[string][]string

// NormalizeHeaderName normalizes a header name.
func NormalizeHeaderName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// NormalizeHeaderValue LF(0x0A) CR(0x0D) TAB(0x09) SPACE(0x20)
func NormalizeHeaderValue(value string) string {
	return strings.TrimFunc(value, func(r rune) bool {
		switch r {
		case 0x0A, 0x0D, 0x09, 0x20:
			return true
		default:
			return false
		}
	})
}

// toHeaders converts a value to a Headers.
func toHeaders(rt *sobek.Runtime, value sobek.Value) Header {
	if value.ExportType() == typeHeaders {
		return value.Export().(Header)
	}
	panic(rt.NewTypeError(`Value of "this" must be of type Headers`))
}

var typeHeaders = reflect.TypeOf((Header)(nil))

// getContentType returns the Content-Type header.
func getContentType(value sobek.Value) string {
	h, _ := value.Export().(Header)
	var contentType string
	if v := h["content-type"]; len(v) > 0 {
		contentType = v[0]
	}
	return contentType
}

// setContentType sets the Content-Type header if not set.
func setContentType(value sobek.Value, contentType string) {
	h, _ := value.Export().(Header)
	if _, ok := h["content-type"]; !ok {
		h["content-type"] = []string{contentType}
	}
}

// NewHeader creates a new Headers.
func NewHeader(rt *sobek.Runtime, src http.Header) sobek.Value {
	return types.New(rt, "Headers", rt.ToValue(Header(src)))
}

// ToHeader converts a value to a http.Header.
func ToHeader(value sobek.Value) (http.Header, bool) {
	if value.ExportType() == typeHeaders {
		return http.Header(value.Export().(Header)), true
	}
	return nil, false
}
