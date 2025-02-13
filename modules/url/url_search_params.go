package url

import (
	"fmt"
	pkgurl "net/url"
	"reflect"
	"slices"
	"strings"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
)

// URLSearchParams defines utility methods to work with the query string of a URL,
// which can be sent using the http() method and encoding type were set to "application/x-www-form-url".
// https://developer.mozilla.org/en-US/docs/Web/API/URLSearchParams
type URLSearchParams struct{}

func (u *URLSearchParams) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()
	_ = p.Set("append", u.append)
	_ = p.Set("delete", u.delete)
	_ = p.Set("forEach", u.forEach)
	_ = p.Set("get", u.get)
	_ = p.Set("getAll", u.getAll)
	_ = p.Set("has", u.has)
	_ = p.Set("set", u.set)
	_ = p.Set("sort", u.sort)
	_ = p.Set("keys", u.keys)
	_ = p.Set("values", u.values)
	_ = p.Set("entries", u.entries)
	_ = p.SetSymbol(sobek.SymIterator, u.entries)
	_ = p.SetSymbol(sobek.SymToStringTag, func(sobek.FunctionCall) sobek.Value { return rt.ToValue("URLSearchParams") })
	_ = p.SetSymbol(sobek.SymHasInstance, func(call sobek.FunctionCall) sobek.Value {
		return rt.ToValue(call.Argument(0).ExportType() == TypeURLSearchParams)
	})
	return p
}

func (u *URLSearchParams) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	params := call.Argument(0)

	var ret urlSearchParams
	ret.data = make(map[string][]string)

	if sobek.IsUndefined(params) {
		goto RET
	}

	switch {
	case params.ExportType().Kind() == reflect.String:
		// "foo=1&bar=2"
		str := strings.TrimPrefix(params.String(), "?")
		for _, kv := range strings.Split(str, "&") {
			if kv == "" {
				continue
			}
			k, v, _ := strings.Cut(kv, "=")
			key, err := pkgurl.QueryUnescape(k)
			if err != nil {
				js.Throw(rt, fmt.Errorf("invalid key '%s': %s", k, err))
			}
			if key == "" {
				continue
			}
			value, err := pkgurl.QueryUnescape(v)
			if err != nil {
				js.Throw(rt, fmt.Errorf("invalid value '%s': %s", k, err))
			}
			values, ok := ret.data[key]
			if !ok {
				ret.keys = append(ret.keys, key)
			}
			ret.data[key] = append(values, value)
		}

	case params.ExportType() == TypeURLSearchParams:
		other := params.Export().(*urlSearchParams)
		ret.keys = make([]string, len(other.keys))
		copy(ret.keys, other.keys)
		for k, v := range other.data {
			values := make([]string, len(v))
			copy(values, v)
			ret.data[k] = values
		}

	default:
		// {foo: "1", bar: ["2", "3"]}
		object := params.ToObject(rt)
		for _, key := range object.Keys() {
			value := object.Get(key)
			if value.ExportType().Kind() == reflect.Array || value.ExportType().Kind() == reflect.Slice {
				arr := value.ToObject(rt)
				length := arr.Get("length").ToInteger()
				values := make([]string, 0, length)
				rt.ForOf(value, func(v sobek.Value) (ok bool) {
					values = append(values, v.String())
					return true
				})
				if len(values) > 0 {
					ret.keys = append(ret.keys, key)
					ret.data[key] = values
				}
			} else {
				ret.keys = append(ret.keys, key)
				ret.data[key] = []string{value.String()}
			}
		}
	}

RET:
	obj := rt.ToValue(&ret).ToObject(rt)
	_ = obj.SetPrototype(call.This.Prototype())
	return obj
}

func (u *URLSearchParams) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	proto := u.prototype(rt)
	ctor := rt.ToValue(u.constructor).(*sobek.Object)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.Set("prototype", proto)
	return ctor, nil
}

var (
	TypeURLSearchParams = reflect.TypeOf((*urlSearchParams)(nil))
)

func toUrlSearchParams(rt *sobek.Runtime, value sobek.Value) *urlSearchParams {
	if value.ExportType() == TypeURLSearchParams {
		return value.Export().(*urlSearchParams)
	}
	panic(rt.NewTypeError(`Value of "this" must be of type URLSearchParams`))
}

type urlSearchParams struct {
	keys []string
	data map[string][]string
}

func (u urlSearchParams) String() string {
	if u.data == nil {
		return ""
	}
	var buf strings.Builder
	for _, key := range u.keys {
		vs := u.data[key]
		keyEscaped := pkgurl.QueryEscape(key)
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(keyEscaped)
			buf.WriteByte('=')
			buf.WriteString(pkgurl.QueryEscape(v))
		}
	}
	return buf.String()
}

func (*URLSearchParams) append(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toUrlSearchParams(rt, call.This)
	name := call.Argument(0).String()
	value := call.Argument(1).String()

	ele, ok := this.data[name]
	if !ok {
		this.keys = append(this.keys, name)
		ele = make([]string, 0)
	}
	this.data[name] = append(ele, value)
	return sobek.Undefined()
}

func (*URLSearchParams) delete(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toUrlSearchParams(rt, call.This)
	name := call.Argument(0).String()
	this.keys = slices.DeleteFunc(this.keys, func(k string) bool { return k == name })
	delete(this.data, name)
	return sobek.Undefined()
}

func (*URLSearchParams) forEach(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toUrlSearchParams(rt, call.This)
	callback, ok := sobek.AssertFunction(call.Argument(0))
	if !ok {
		panic(rt.NewTypeError("callback is not a function"))
	}

	for _, key := range this.keys {
		for _, value := range this.data[key] {
			// forEach callback signature: (value, key, this)
			_, err := callback(call.Argument(0), rt.ToValue(value), rt.ToValue(key), call.This)
			if err != nil {
				js.Throw(rt, err)
			}
		}
	}

	return sobek.Undefined()
}

func (*URLSearchParams) get(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toUrlSearchParams(rt, call.This)
	name := call.Argument(0).String()
	if v, ok := this.data[name]; ok {
		if len(v) > 0 {
			return rt.ToValue(v[0])
		}
	}
	return rt.ToValue(nil)
}

func (*URLSearchParams) getAll(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toUrlSearchParams(rt, call.This)
	name := call.Argument(0).String()
	v, ok := this.data[name]
	if ok {
		return rt.ToValue(v)
	}
	return rt.NewArray()
}

func (*URLSearchParams) has(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toUrlSearchParams(rt, call.This)
	name := call.Argument(0).String()
	_, ok := this.data[name]
	return rt.ToValue(ok)
}

func (*URLSearchParams) set(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toUrlSearchParams(rt, call.This)
	name := call.Argument(0).String()
	value := call.Argument(1).String()

	if _, ok := this.data[name]; !ok {
		this.keys = append(this.keys, name)
	}
	this.data[name] = []string{value}
	return sobek.Undefined()
}

func (*URLSearchParams) sort(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toUrlSearchParams(rt, call.This)
	slices.Sort(this.keys)
	return sobek.Undefined()
}

func (*URLSearchParams) keys(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toUrlSearchParams(rt, call.This)
	return js.Iterator(rt, func(yield func(any) bool) {
		for _, key := range this.keys {
			if !yield(key) {
				return
			}
		}
	})
}

func (*URLSearchParams) values(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toUrlSearchParams(rt, call.This)
	return js.Iterator(rt, func(yield func(any) bool) {
		for _, key := range this.keys {
			var value any
			if values := this.data[key]; len(values) > 0 {
				value = values[0]
			}
			if !yield(value) {
				return
			}
		}
	})
}

func (*URLSearchParams) entries(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toUrlSearchParams(rt, call.This)
	return js.Iterator(rt, func(yield func(any) bool) {
		for _, key := range this.keys {
			var value any
			if values := this.data[key]; len(values) > 0 {
				value = values[0]
			}
			if !yield(rt.NewArray(key, value)) {
				return
			}
		}
	})
}
