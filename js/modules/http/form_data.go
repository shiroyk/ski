package http

import (
	"fmt"
	"slices"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski"
	"github.com/shiroyk/ski/js"
)

// fileData wraps the file data and filename
type fileData struct {
	data     []byte
	filename string
}

// formData provides a way to construct a set of key/value pairs representing form fields and their values.
// which can be sent using the http() method and encoding type were set to "multipart/form-data".
// Implement the https://developer.mozilla.org/en-US/docs/Web/API/FormData
type formData struct {
	keys []string
	data map[string][]any
}

// FormData Constructor
type FormData struct{}

// Instantiate returns module instance
func (*FormData) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	return rt.ToValue(func(call sobek.ConstructorCall) *sobek.Object {
		params := call.Argument(0)

		var ret formData
		if sobek.IsUndefined(params) {
			ret.data = make(map[string][]any)
			return ret.object(rt)
		}

		object := params.ToObject(rt)
		keys := object.Keys()
		ret.keys = make([]string, 0, len(keys))
		ret.data = make(map[string][]any, len(keys))

		for _, key := range keys {
			value, _ := js.Unwrap(object.Get(key))
			switch ve := value.(type) {
			case []byte:
				// Default filename "blob".
				ret.data[key] = []any{fileData{
					data:     ve,
					filename: "blob",
				}}
			case sobek.ArrayBuffer:
				// Default filename "blob".
				ret.data[key] = []any{fileData{
					data:     ve.Bytes(),
					filename: "blob",
				}}
			case []any:
				ret.data[key] = ve
			case nil:
				ret.data[key] = nil
			default:
				ret.data[key] = []any{fmt.Sprintf("%s", ve)}
			}
			ret.keys = append(ret.keys, key)
		}

		return ret.object(rt)
	}), nil
}

// Global it is a global module
func (*FormData) Global() {}

func (f *formData) object(rt *sobek.Runtime) *sobek.Object {
	obj := rt.ToValue(f).ToObject(rt)

	_ = obj.SetSymbol(sobek.SymIterator, func(sobek.ConstructorCall) *sobek.Object {
		var i int
		it := rt.NewObject()
		_ = it.Set("next", func(sobek.FunctionCall) sobek.Value {
			if i < len(f.keys) {
				key := f.keys[i]
				i++
				return rt.ToValue(iter{Value: rt.ToValue([2]any{key, f.data[key]})})
			}
			return rt.ToValue(iter{Done: true})
		})
		return it
	})
	return obj
}

// Append method of the formData interface appends a new value onto an existing key inside a formData object,
// or adds the key if it does not already exist.
func (f *formData) Append(name string, value any, filename string) sobek.Value {
	if filename == "" {
		// Default filename "blob".
		filename = "blob"
	}

	ele, ok := f.data[name]
	if !ok {
		f.keys = append(f.keys, name)
		ele = make([]any, 0)
	}

	switch v := value.(type) {
	case []byte:
		ele = append(ele, fileData{
			data:     v,
			filename: filename,
		})
	case sobek.ArrayBuffer:
		ele = append(ele, fileData{
			data:     v.Bytes(),
			filename: filename,
		})
	default:
		ele = append(ele, fmt.Sprintf("%v", v))
	}

	f.data[name] = ele

	return sobek.Undefined()
}

// Delete method of the formData interface deletes a key and its value(s) from a formData object.
func (f *formData) Delete(name string) {
	f.keys = slices.DeleteFunc(f.keys, func(k string) bool { return k == name })
	delete(f.data, name)
}

// Entries method returns an iterator which iterates through all key/value pairs contained in the formData.
func (f *formData) Entries() any {
	entries := make([][2]any, 0, len(f.keys))
	for _, key := range f.keys {
		entries = append(entries, [2]any{key, f.data[key]})
	}
	return entries
}

// Get method of the formData interface returns the first value associated
// with a given key from within a formData object.
// If you expect multiple values and want all of them, use the getAll() method instead.
func (f *formData) Get(name string) any {
	if v, ok := f.data[name]; ok {
		return v[0]
	}
	return nil
}

// GetAll method of the formData interface returns all the values associated
// with a given key from within a formData object.
func (f *formData) GetAll(name string) any {
	v, ok := f.data[name]
	if ok {
		return v
	}
	return [0]any{}
}

// Has method of the formData interface returns whether a formData object contains a certain key.
func (f *formData) Has(name string) bool {
	_, ok := f.data[name]
	return ok
}

// Keys method returns an iterator which iterates through all keys contained in the formData.
// The keys are strings.
func (f *formData) Keys() any { return f.keys }

// Set method of the formData interface sets a new value for an existing key inside a formData object,
// or adds the key/value if it does not already exist.
func (f *formData) Set(name string, value any, filename string) {
	if filename == "" {
		filename = "blob"
	}

	if _, ok := f.data[name]; !ok {
		f.keys = append(f.keys, name)
	}

	switch v := value.(type) {
	case sobek.ArrayBuffer:
		f.data[name] = []any{
			fileData{
				data:     v.Bytes(),
				filename: filename,
			},
		}
	default:
		f.data[name] = []any{fmt.Sprintf("%v", v)}
	}
}

// Values method returns an iterator which iterates through all values contained in the formData.
func (f *formData) Values() any { return ski.MapValues(f.data) }
