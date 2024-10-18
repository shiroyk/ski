package http

import (
	"fmt"
	"net/url"
	"reflect"
	"slices"
	"strings"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/spf13/cast"
)

// The urlSearchParams defines utility methods to work with the query string of a URL,
// which can be sent using the http() method and encoding type were set to "application/x-www-form-url".
// Implement the https://developer.mozilla.org/en-US/docs/Web/API/URLSearchParams
type urlSearchParams struct {
	keys []string
	data map[string][]string
}

// URLSearchParams Constructor
type URLSearchParams struct{}

// Instantiate instance module
func (*URLSearchParams) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	return rt.ToValue(func(call sobek.ConstructorCall) *sobek.Object {
		params := call.Argument(0)

		var ret urlSearchParams
		if sobek.IsUndefined(params) {
			ret.data = make(map[string][]string)
			return ret.object(rt)
		}

		if params.ExportType().Kind() == reflect.String {
			str := strings.TrimPrefix(params.String(), "?")
			kvs := strings.Split(str, "&")
			ret.data = make(map[string][]string, len(kvs))
			for _, kv := range kvs {
				k, v, _ := strings.Cut(kv, "=")
				ret.Append(k, v)
			}
			return ret.object(rt)
		}

		object := params.ToObject(rt)
		keys := object.Keys()
		ret.keys = make([]string, 0, len(keys))
		ret.data = make(map[string][]string, len(keys))

		for _, key := range keys {
			value, _ := js.Unwrap(object.Get(key))
			if s, ok := value.([]any); ok {
				ret.data[key] = cast.ToStringSlice(s)
			} else {
				ret.data[key] = []string{fmt.Sprintf("%s", value)}
			}
			ret.keys = append(ret.keys, key)
		}

		return ret.object(rt)
	}), nil
}

// Global it is a global module
func (*URLSearchParams) Global() {}

func (u *urlSearchParams) object(rt *sobek.Runtime) *sobek.Object {
	obj := rt.ToValue(u).ToObject(rt)

	_ = obj.SetSymbol(sobek.SymIterator, func(sobek.ConstructorCall) *sobek.Object {
		var i int
		it := rt.NewObject()
		_ = it.Set("next", func(sobek.FunctionCall) sobek.Value {
			if i < len(u.keys) {
				key := u.keys[i]
				i++
				return rt.ToValue(iter{Value: rt.ToValue([2]any{key, u.data[key]})})
			}
			return rt.ToValue(iter{Done: true})
		})
		return it
	})
	return obj
}

// encode encodes the values into “URL encoded” form
// ("bar=baz&foo=qux") sorted by key.
func (u *urlSearchParams) encode() string {
	if u.data == nil {
		return ""
	}
	var buf strings.Builder
	for _, key := range u.keys {
		vs := u.data[key]
		keyEscaped := url.QueryEscape(key)
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(keyEscaped)
			buf.WriteByte('=')
			buf.WriteString(url.QueryEscape(v))
		}
	}
	return buf.String()
}

// Append method of the urlSearchParams interface appends a specified key/value pair as a new search parameter.
func (u *urlSearchParams) Append(name, value string) {
	values, ok := u.data[name]
	if !ok {
		u.keys = append(u.keys, name)
	}
	u.data[name] = append(values, value)
}

// Delete method of the urlSearchParams interface deletes the given search parameter and all its associated values,
// from the list of all search parameters.
func (u *urlSearchParams) Delete(name string) {
	u.keys = slices.DeleteFunc(u.keys, func(k string) bool { return k == name })
	delete(u.data, name)
}

// Entries method of the urlSearchParams interface returns an iterator allowing iteration
// through all key/value pairs contained in this object.
// The iterator returns key/value pairs in the same order as they appear in the query string.
// The key and value of each pair are string objects.
func (u *urlSearchParams) Entries() any {
	entries := make([][2]any, 0, len(u.keys))
	for _, key := range u.keys {
		entries = append(entries, [2]any{key, u.data[key]})
	}
	return entries
}

// ForEach method of the urlSearchParams interface allows iteration
// through all values contained in this object via a callback function.
func (u *urlSearchParams) ForEach(call sobek.FunctionCall, vm *sobek.Runtime) (ret sobek.Value) {
	arg := call.Argument(0)
	if callback, ok := sobek.AssertFunction(arg); ok {
		for _, key := range u.keys {
			if _, err := callback(sobek.Undefined(), vm.ToValue(u.data[key]), vm.ToValue(key), vm.ToValue(u)); err != nil {
				panic(vm.ToValue(err))
			}
		}
	}
	return
}

// Get method of the urlSearchParams interface returns the first value associated to the given search parameter.
func (u *urlSearchParams) Get(name string) any {
	if v, ok := u.data[name]; ok {
		return v[0]
	}
	return nil
}

// GetAll method of the urlSearchParams interface returns all the values associated
// with a given search parameter as an array.
func (u *urlSearchParams) GetAll(name string) []string {
	if v, ok := u.data[name]; ok {
		return v
	}
	return []string{}
}

// Has method of the urlSearchParams interface returns a boolean value that indicates whether
// a parameter with the specified name exists.
func (u *urlSearchParams) Has(name string) bool {
	_, ok := u.data[name]
	return ok
}

// Keys method of the urlSearchParams interface returns an iterator allowing iteration
// through all keys contained in this object. The keys are string objects.
func (u *urlSearchParams) Keys() []string { return u.keys }

// Set method of the urlSearchParams interface sets the value associated
// with a given search parameter to the given value.
// If there were several matching values, this method deletes the others.
// If the search parameter doesn't exist, this method creates it.
func (u *urlSearchParams) Set(name, value string) {
	if _, ok := u.data[name]; !ok {
		u.keys = append(u.keys, name)
	}
	u.data[name] = []string{value}
}

// Sort method sorts all key/value pairs contained in this object in place and returns undefined.
func (u *urlSearchParams) Sort() { slices.Sort(u.keys) }

// ToString method of the urlSearchParams interface returns a query string suitable for use in a URL.
func (u *urlSearchParams) ToString() string {
	return u.encode()
}

// Values method of the urlSearchParams interface returns an iterator allowing iteration through
// all values contained in this object. The values are string objects.
func (u *urlSearchParams) Values() [][]string {
	return js.MapValues(u.data)
}
