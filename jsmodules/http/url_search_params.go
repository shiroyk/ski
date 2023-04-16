package http

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/core/js"
	"github.com/spf13/cast"
	"golang.org/x/exp/maps"
)

// The URLSearchParams defines utility methods to work with the query string of a URL,
// which can be sent using the http() method and encoding type were set to "application/x-www-form-url".
// Implement the https://developer.mozilla.org/en-US/docs/Web/API/URLSearchParams
type URLSearchParams struct {
	data map[string][]string
}

// NativeURLSearchParams Native module
type NativeURLSearchParams struct{}

// Exports instance URLSearchParams module
func (*NativeURLSearchParams) Exports() any {
	return func(call goja.ConstructorCall, vm *goja.Runtime) *goja.Object {
		param := call.Argument(0)

		if goja.IsUndefined(param) {
			return vm.ToValue(URLSearchParams{data: make(url.Values)}).ToObject(vm)
		}

		var pa map[string]any
		var ok bool
		pa, ok = param.Export().(map[string]any)
		if !ok {
			js.Throw(vm, fmt.Errorf("unsupported type %T", param.Export()))
		}

		data := make(map[string][]string, len(pa))
		for k, v := range pa {
			if s, ok := v.([]any); ok {
				data[k] = cast.ToStringSlice(s)
			} else {
				data[k] = []string{fmt.Sprintf("%v", v)}
			}
		}

		return vm.ToValue(URLSearchParams{data: data}).ToObject(vm)
	}
}

// Global it is a global module
func (*NativeURLSearchParams) Global() {}

// encode encodes the values into “URL encoded” form
// ("bar=baz&foo=qux") sorted by key.
func (u *URLSearchParams) encode() string {
	if u.data == nil {
		return ""
	}
	var buf strings.Builder
	keys := maps.Keys(u.data)
	sort.Strings(keys)
	for _, k := range keys {
		vs := u.data[k]
		keyEscaped := url.QueryEscape(k)
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

// Append method of the URLSearchParams interface appends a specified key/value pair as a new search parameter.
func (u *URLSearchParams) Append(name, value string) {
	u.data[name] = append(u.data[name], value)
}

// Delete method of the URLSearchParams interface deletes the given search parameter and all its associated values,
// from the list of all search parameters.
func (u *URLSearchParams) Delete(name string) {
	delete(u.data, name)
}

// Entries method of the URLSearchParams interface returns an iterator allowing iteration
// through all key/value pairs contained in this object.
// The iterator returns key/value pairs in the same order as they appear in the query string.
// The key and value of each pair are string objects.
func (u *URLSearchParams) Entries() any {
	entries := make([][2]string, 0, len(u.data))
	for k, v := range u.data {
		for _, ve := range v {
			entries = append(entries, [2]string{k, ve})
		}
	}
	return entries
}

// ForEach method of the URLSearchParams interface allows iteration
// through all values contained in this object via a callback function.
func (u *URLSearchParams) ForEach(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	arg := call.Argument(0)
	if callback, ok := goja.AssertFunction(arg); ok {
		for k, v := range u.data {
			if _, err := callback(goja.Undefined(), vm.ToValue(v), vm.ToValue(k), vm.ToValue(u)); err != nil {
				panic(vm.ToValue(err))
			}
		}
	}
	return
}

// Get method of the URLSearchParams interface returns the first value associated to the given search parameter.
func (u *URLSearchParams) Get(name string) any {
	if v, ok := u.data[name]; ok {
		return v[0]
	}
	return nil
}

// GetAll method of the URLSearchParams interface returns all the values associated
// with a given search parameter as an array.
func (u *URLSearchParams) GetAll(name string) []string {
	if v, ok := u.data[name]; ok {
		return v
	}
	return []string{}
}

// Has method of the URLSearchParams interface returns a boolean value that indicates whether
// a parameter with the specified name exists.
func (u *URLSearchParams) Has(name string) bool {
	_, ok := u.data[name]
	return ok
}

// Keys method of the URLSearchParams interface returns an iterator allowing iteration
// through all keys contained in this object. The keys are string objects.
func (u *URLSearchParams) Keys() []string {
	return maps.Keys(u.data)
}

// Set method of the URLSearchParams interface sets the value associated
// with a given search parameter to the given value.
// If there were several matching values, this method deletes the others.
// If the search parameter doesn't exist, this method creates it.
func (u *URLSearchParams) Set(name, value string) {
	u.data[name] = []string{value}
}

// Sort method sorts all key/value pairs contained in this object in place and returns undefined.
func (u *URLSearchParams) Sort() {
	// Not implemented
}

// ToString method of the URLSearchParams interface returns a query string suitable for use in a URL.
func (u *URLSearchParams) ToString() string {
	return u.encode()
}

// Values method of the URLSearchParams interface returns an iterator allowing iteration through
// all values contained in this object. The values are string objects.
func (u *URLSearchParams) Values() [][]string {
	return maps.Values(u.data)
}
