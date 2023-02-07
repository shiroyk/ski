package http

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/js/common"
	"golang.org/x/exp/maps"
)

// FileData wraps the file data and filename
type FileData struct {
	Data     []byte
	Filename string
}

// FormData provides a way to construct a set of key/value pairs representing form fields and their values.
// which can be sent using the http() method and encoding type were set to "multipart/form-data".
type FormData struct {
	data map[string][]any
}

type NativeFormData struct{}

func (*NativeFormData) New() any {
	return func(call goja.ConstructorCall, vm *goja.Runtime) *goja.Object {
		param := call.Argument(0)

		if goja.IsUndefined(param) {
			return vm.ToValue(FormData{make(map[string][]any)}).ToObject(vm)
		}

		var pa map[string]any
		var ok bool
		pa, ok = param.Export().(map[string]any)
		if !ok {
			common.Throw(vm, fmt.Errorf("unsupported type %T", param.Export()))
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

// Append method of the FormData interface appends a new value onto an existing key inside a FormData object,
// or adds the key if it does not already exist.
func (f *FormData) Append(call goja.FunctionCall) (ret goja.Value) {
	name := call.Argument(0).String()
	value := call.Argument(1).Export()
	var filename string

	if goja.IsUndefined(call.Argument(2)) {
		// Default filename "blob".
		filename = "blob"
	} else {
		filename = call.Argument(2).String()
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
func (f *FormData) Delete(call goja.FunctionCall) (ret goja.Value) {
	delete(f.data, call.Argument(0).String())
	return
}

// Entries method returns an iterator which iterates through all key/value pairs contained in the FormData.
func (f *FormData) Entries(_ goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	entries := make([][2]any, 0, len(f.data))
	for k, v := range f.data {
		entries = append(entries, [2]any{k, v})
	}
	return vm.ToValue(entries)
}

// Get method of the FormData interface returns the first value associated
// with a given key from within a FormData object.
// If you expect multiple values and want all of them, use the getAll() method instead.
func (f *FormData) Get(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	if v, ok := f.data[call.Argument(0).String()]; ok {
		return vm.ToValue(v[0])
	}
	return
}

// GetAll method of the FormData interface returns all the values associated
// with a given key from within a FormData object.
func (f *FormData) GetAll(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	if v, ok := f.data[call.Argument(0).String()]; ok {
		return vm.ToValue(v)
	}
	return vm.ToValue([0]any{})
}

// Has method of the FormData interface returns whether a FormData object contains a certain key.
func (f *FormData) Has(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	if _, ok := f.data[call.Argument(0).String()]; ok {
		return vm.ToValue(true)
	}
	return vm.ToValue(false)
}

// Keys method returns an iterator which iterates through all keys contained in the FormData.
// The keys are strings.
func (f *FormData) Keys(_ goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	return vm.ToValue(maps.Keys(f.data))
}

// Set method of the FormData interface sets a new value for an existing key inside a FormData object,
// or adds the key/value if it does not already exist.
func (f *FormData) Set(call goja.FunctionCall) (ret goja.Value) {
	name := call.Argument(0).String()
	value := call.Argument(1).Export()

	var filename string
	if goja.IsUndefined(call.Argument(2)) {
		filename = "blob"
	} else {
		filename = call.Argument(2).String()
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

	return
}

// Values method returns an iterator which iterates through all values contained in the FormData.
func (f *FormData) Values(_ goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	return vm.ToValue(maps.Values(f.data))
}
