package url

import (
	"reflect"
	"slices"
	"strings"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/types"
)

// URLSearchParams defines utility methods to work with the query string of a URL,
// which can be sent using the http() method and encoding type were set to "application/x-www-form-url".
// https://developer.mozilla.org/en-US/docs/Web/API/URLSearchParams
type URLSearchParams struct{}

func (u *URLSearchParams) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()
	_ = p.DefineAccessorProperty("size", rt.ToValue(u.size), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)

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
	return p
}

func (u *URLSearchParams) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	params := call.Argument(0)

	var ret urlSearchParams
	ret.data = make(map[string][]string)

	if sobek.IsUndefined(params) {
		goto RET
	}

	switch params.ExportType() {
	case types.TypeString:
		// "foo=1&bar=2"
		ret.fromString(params.String())

	case TypeURLSearchParams:
		other := params.Export().(*urlSearchParams)
		ret.keys = make([]string, len(other.keys))
		copy(ret.keys, other.keys)
		for k, v := range other.data {
			values := make([]string, len(v))
			copy(values, v)
			ret.data[k] = values
		}

	default:
		object := params.ToObject(rt)
		if object.GetSymbol(sobek.SymIterator) != nil {
			// [["foo", "1"], ["bar", "2"]]
			rt.ForOf(object, func(v sobek.Value) bool {
				item := v.ToObject(rt)
				length := item.Get("length")
				if length == nil || length.ToInteger() != 2 {
					panic(rt.NewTypeError("The provided value cannot be converted to a sequence."))
				}

				key := item.Get("0").String()
				value := item.Get("1").String()
				if _, ok := ret.data[key]; !ok {
					ret.keys = append(ret.keys, key)
				}
				ret.data[key] = append(ret.data[key], value)
				return true
			})
		} else {
			// {foo: "1", bar: ["2", "3"]}
			for _, key := range object.Keys() {
				value := object.Get(key)
				ret.keys = append(ret.keys, key)
				if value == nil {
					continue
				}
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

func toURLSearchParams(rt *sobek.Runtime, value sobek.Value) *urlSearchParams {
	if value.ExportType() == TypeURLSearchParams {
		return value.Export().(*urlSearchParams)
	}
	panic(rt.NewTypeError(`Value of "this" must be of type URLSearchParams`))
}

type urlSearchParams struct {
	keys []string
	data map[string][]string
}

func (u *urlSearchParams) String() string {
	if len(u.data) == 0 {
		return ""
	}
	var buf strings.Builder
	for _, key := range u.keys {
		vs := u.data[key]
		keyEscaped := queryEscape(key)
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(keyEscaped)
			buf.WriteByte('=')
			buf.WriteString(queryEscape(v))
		}
	}
	return buf.String()
}

func (u *urlSearchParams) fromString(str string) {
	str = strings.TrimPrefix(str, "?")
	kvs := strings.Split(str, "&")
	u.keys = make([]string, 0, len(kvs))
	u.data = make(map[string][]string, len(kvs))
	for _, kv := range kvs {
		if kv == "" {
			continue
		}
		k, v, _ := strings.Cut(kv, "=")
		k = queryUnescape(k)
		values, ok := u.data[k]
		if !ok {
			u.keys = append(u.keys, k)
		}
		u.data[k] = append(values, queryUnescape(v))
	}
}

func (*URLSearchParams) size(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURLSearchParams(rt, call.This)
	size := 0
	for _, v := range this.data {
		size += len(v)
	}
	return rt.ToValue(size)
}

func (*URLSearchParams) append(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURLSearchParams(rt, call.This)
	name := call.Argument(0).String()
	value := call.Argument(1).String()

	ele, ok := this.data[name]
	if !ok {
		this.keys = append(this.keys, name)
		ele = make([]string, 0, 1)
	}
	this.data[name] = append(ele, value)
	return sobek.Undefined()
}

func (*URLSearchParams) delete(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURLSearchParams(rt, call.This)
	name := call.Argument(0).String()
	if v := call.Argument(1); !sobek.IsUndefined(v) {
		elem := slices.DeleteFunc(this.data[name], func(s string) bool { return s == v.String() })
		if len(elem) > 0 {
			this.data[name] = elem
			return sobek.Undefined()
		}
	}
	this.keys = slices.DeleteFunc(this.keys, func(k string) bool { return k == name })
	delete(this.data, name)
	return sobek.Undefined()
}

func (*URLSearchParams) forEach(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURLSearchParams(rt, call.This)
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
	this := toURLSearchParams(rt, call.This)
	name := call.Argument(0).String()
	if v, ok := this.data[name]; ok {
		if len(v) > 0 {
			return rt.ToValue(v[0])
		}
	}
	return sobek.Null()
}

func (*URLSearchParams) getAll(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURLSearchParams(rt, call.This)
	name := call.Argument(0).String()
	v, ok := this.data[name]
	if ok {
		return rt.ToValue(v)
	}
	return rt.NewArray()
}

func (*URLSearchParams) has(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURLSearchParams(rt, call.This)
	name := call.Argument(0).String()
	elem, ok := this.data[name]
	if v := call.Argument(1); !sobek.IsUndefined(v) {
		ok = slices.Contains(elem, v.String())
	}
	return rt.ToValue(ok)
}

func (*URLSearchParams) set(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURLSearchParams(rt, call.This)
	name := call.Argument(0).String()
	value := call.Argument(1).String()

	if _, ok := this.data[name]; !ok {
		this.keys = append(this.keys, name)
	}
	this.data[name] = []string{value}
	return sobek.Undefined()
}

func (*URLSearchParams) sort(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURLSearchParams(rt, call.This)
	slices.Sort(this.keys)
	return sobek.Undefined()
}

func (*URLSearchParams) keys(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURLSearchParams(rt, call.This)
	return types.Iterator(rt, func(yield func(any) bool) {
		for _, key := range this.keys {
			if !yield(key) {
				return
			}
		}
	})
}

func (*URLSearchParams) values(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toURLSearchParams(rt, call.This)
	return types.Iterator(rt, func(yield func(any) bool) {
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
	this := toURLSearchParams(rt, call.This)
	return types.Iterator(rt, func(yield func(any) bool) {
		for _, key := range this.keys {
			for _, value := range this.data[key] {
				if !yield(rt.NewArray(key, value)) {
					return
				}
			}
		}
	})
}
