package http

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/core/js"
	"golang.org/x/exp/maps"
)

// FileData wraps the file data and filename
type FileData struct {
	Data     []byte
	Filename string
}

// FormData provides a way to construct a set of key/value pairs representing form fields and their values.
// which can be sent using the http() method and encoding type were set to "multipart/form-data".
// Implement the https://developer.mozilla.org/en-US/docs/Web/API/FormData
type FormData struct {
	data map[string][]any
}

// FormDataConstructor FormData Constructor
type FormDataConstructor struct{}

// Exports returns module instance
func (*FormDataConstructor) Exports() any {
	return func(call goja.ConstructorCall, vm *goja.Runtime) *goja.Object {
		param := call.Argument(0)

		if goja.IsUndefined(param) {
			return vm.ToValue(FormData{make(map[string][]any)}).ToObject(vm)
		}

		var pa map[string]any
		var ok bool
		pa, ok = param.Export().(map[string]any)
		if !ok {
			js.Throw(vm, fmt.Errorf("unsupported type %T", param.Export()))
		}

		data := make(map[string][]any, len(pa))

		for k, v := range pa {
			switch ve := v.(type) {
			case goja.ArrayBuffer:
				// Default filename "blob".
				data[k] = []any{FileData{
					Data:     ve.Bytes(),
					Filename: "blob",
				}}
			case []any:
				data[k] = ve
			default:
				data[k] = []any{fmt.Sprintf("%v", ve)}
			}
		}

		return vm.ToValue(FormData{data}).ToObject(vm)
	}
}

// Global it is a global module
func (*FormDataConstructor) Global() {}

// Append method of the FormData interface appends a new value onto an existing key inside a FormData object,
// or adds the key if it does not already exist.
func (f *FormData) Append(name string, value any, filename string) (ret goja.Value) {
	if filename == "" {
		// Default filename "blob".
		filename = "blob"
	}

	var ele []any
	var ok bool
	if ele, ok = f.data[name]; !ok {
		ele = make([]any, 0)
	}

	switch v := value.(type) {
	case goja.ArrayBuffer:
		ele = append(ele, FileData{
			Data:     v.Bytes(),
			Filename: filename,
		})
	default:
		ele = append(ele, fmt.Sprintf("%v", v))
	}

	f.data[name] = ele

	return
}

// Delete method of the FormData interface deletes a key and its value(s) from a FormData object.
func (f *FormData) Delete(name string) {
	delete(f.data, name)
}

// Entries method returns an iterator which iterates through all key/value pairs contained in the FormData.
func (f *FormData) Entries() any {
	entries := make([][2]any, 0, len(f.data))
	for k, v := range f.data {
		entries = append(entries, [2]any{k, v})
	}
	return entries
}

// Get method of the FormData interface returns the first value associated
// with a given key from within a FormData object.
// If you expect multiple values and want all of them, use the getAll() method instead.
func (f *FormData) Get(name string) any {
	if v, ok := f.data[name]; ok {
		return v[0]
	}
	return nil
}

// GetAll method of the FormData interface returns all the values associated
// with a given key from within a FormData object.
func (f *FormData) GetAll(name string) any {
	v, ok := f.data[name]
	if ok {
		return v
	}
	return [0]any{}
}

// Has method of the FormData interface returns whether a FormData object contains a certain key.
func (f *FormData) Has(name string) bool {
	_, ok := f.data[name]
	return ok
}

// Keys method returns an iterator which iterates through all keys contained in the FormData.
// The keys are strings.
func (f *FormData) Keys() any {
	return maps.Keys(f.data)
}

// Set method of the FormData interface sets a new value for an existing key inside a FormData object,
// or adds the key/value if it does not already exist.
func (f *FormData) Set(name string, value any, filename string) {
	if filename == "" {
		filename = "blob"
	}

	switch v := value.(type) {
	case goja.ArrayBuffer:
		f.data[name] = []any{
			FileData{
				Data:     v.Bytes(),
				Filename: filename,
			},
		}
	default:
		f.data[name] = []any{fmt.Sprintf("%v", v)}
	}
}

// Values method returns an iterator which iterates through all values contained in the FormData.
func (f *FormData) Values() any {
	return maps.Values(f.data)
}
