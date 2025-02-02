package http

import (
	"bytes"
	"io"
	"mime/multipart"
	"reflect"
	"slices"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
)

// FormData provides a way to construct a set of key/value pairs representing form fields and their values.
// which can be sent using the http() method and encoding type were set to "multipart/form-data".
// https://developer.mozilla.org/en-US/docs/Web/API/FormData
type FormData struct{}

func (f *FormData) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()
	_ = p.Set("append", f.append)
	_ = p.Set("delete", f.delete)
	_ = p.Set("forEach", f.forEach)
	_ = p.Set("get", f.get)
	_ = p.Set("getAll", f.getAll)
	_ = p.Set("has", f.has)
	_ = p.Set("set", f.set)
	_ = p.Set("keys", f.keys)
	_ = p.Set("values", f.values)
	_ = p.Set("entries", f.entries)
	_ = p.SetSymbol(sobek.SymIterator, f.entries)
	_ = p.SetSymbol(sobek.SymToStringTag, func(sobek.ConstructorCall) sobek.Value { return rt.ToValue("FormData") })
	return p
}

func (f *FormData) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	params := call.Argument(0)

	var ret formData
	ret.data = make(map[string][]sobek.Value)

	obj := rt.ToValue(&ret).ToObject(rt)
	_ = obj.SetPrototype(call.This.Prototype())

	if !sobek.IsUndefined(params) {
		callable, ok := sobek.AssertFunction(obj.Get("append"))
		if !ok {
			panic(rt.NewTypeError("invalid formData prototype"))
		}
		if params.ExportType().Kind() != reflect.Map {
			panic(rt.NewTypeError("invalid formData constructor argument"))
		}
		object := params.ToObject(rt)
		for _, key := range object.Keys() {
			_, err := callable(obj, rt.ToValue(key), object.Get(key))
			if err != nil {
				js.Throw(rt, err)
			}
		}
	}

	return obj
}

func (f *FormData) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	proto := f.prototype(rt)
	ctor := rt.ToValue(f.constructor).(*sobek.Object)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.Set("prototype", proto)
	return ctor, nil
}

func (*FormData) Global() {}

func (*FormData) append(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toFormData(rt, call.This)
	name := call.Argument(0).String()
	value := call.Argument(1)

	ele, ok := this.data[name]
	if !ok {
		this.keys = append(this.keys, name)
		ele = make([]sobek.Value, 0)
	}

	switch value.ExportType() {
	case typeArrayBuffer, typeBlob:
		filename := call.Argument(2)
		if f := call.Argument(2); sobek.IsUndefined(f) {
			// Default filename "blob".
			filename = rt.ToValue("blob")
		}

		file, err := js.New(rt, "File", rt.NewArray(value), filename)
		if err != nil {
			js.Throw(rt, err)
		}
		ele = append(ele, file)
	default:
		if !sobek.IsUndefined(value) {
			ele = append(ele, value)
		}
	}

	this.data[name] = ele

	return sobek.Undefined()
}

func (*FormData) delete(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toFormData(rt, call.This)
	name := call.Argument(0).String()
	this.keys = slices.DeleteFunc(this.keys, func(k string) bool { return k == name })
	delete(this.data, name)
	return sobek.Undefined()
}

func (*FormData) forEach(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toFormData(rt, call.This)
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

func (*FormData) get(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toFormData(rt, call.This)
	name := call.Argument(0).String()
	if v, ok := this.data[name]; ok {
		if len(v) > 0 {
			return rt.ToValue(v[0])
		}
	}
	return sobek.Undefined()
}

func (*FormData) getAll(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toFormData(rt, call.This)
	name := call.Argument(0).String()
	v, ok := this.data[name]
	if ok {
		return rt.ToValue(v)
	}
	return rt.NewArray()
}

func (*FormData) has(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toFormData(rt, call.This)
	name := call.Argument(0).String()
	_, ok := this.data[name]
	return rt.ToValue(ok)
}

func (*FormData) set(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toFormData(rt, call.This)
	name := call.Argument(0).String()
	value := call.Argument(1)
	if _, ok := this.data[name]; !ok {
		this.keys = append(this.keys, name)
	}

	switch value.ExportType() {
	case typeArrayBuffer, typeBlob:
		filename := call.Argument(2)
		if f := call.Argument(2); sobek.IsUndefined(f) {
			// Default filename "blob".
			filename = rt.ToValue("blob")
		}

		file, err := js.New(rt, "File", rt.NewArray(value), filename)
		if err != nil {
			js.Throw(rt, err)
		}
		this.data[name] = []sobek.Value{file}
	default:
		if !sobek.IsUndefined(value) {
			this.data[name] = []sobek.Value{value}
		}
	}
	return sobek.Undefined()
}

func (*FormData) keys(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toFormData(rt, call.This)
	return js.Iterator(rt, func(yield func(any) bool) {
		for _, key := range this.keys {
			if !yield(key) {
				return
			}
		}
	})
}

func (*FormData) values(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toFormData(rt, call.This)
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

func (*FormData) entries(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toFormData(rt, call.This)
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

type formData struct {
	keys []string
	data map[string][]sobek.Value
}

func (f *formData) encode() (io.Reader, string, error) {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)
	for _, key := range f.keys {
		for _, value := range f.data[key] {
			switch value.ExportType() {
			case typeFile:
				file := value.Export().(*file)
				fw, err := writer.CreateFormFile(key, file.name)
				if err != nil {
					return nil, "", err
				}
				if file.blob == nil {
					fw.Write(nil)
					continue
				}
				if _, err = io.Copy(fw, file.blob.Export().(*blob).data); err != nil {
					return nil, "", err
				}
			default:
				if err := writer.WriteField(key, value.String()); err != nil {
					return nil, "", err
				}
			}
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return buf, writer.FormDataContentType(), nil
}

var (
	typeFormData = reflect.TypeOf((*formData)(nil))
)

func toFormData(rt *sobek.Runtime, value sobek.Value) *formData {
	if value.ExportType() == typeFormData {
		return value.Export().(*formData)
	}
	panic(rt.NewTypeError(`Value of "this" must be of type FormData`))
}
