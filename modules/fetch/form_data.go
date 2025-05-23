package fetch

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/types"
	"github.com/shiroyk/ski/modules/buffer"
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
	_ = p.SetSymbol(sobek.SymToStringTag, func(sobek.FunctionCall) sobek.Value { return rt.ToValue("FormData") })
	return p
}

func (f *FormData) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	params := call.Argument(0)

	var ret formData
	ret.data = make(map[string][]sobek.Value)

	obj := rt.ToValue(&ret).ToObject(rt)
	_ = obj.SetPrototype(call.This.Prototype())

	if sobek.IsUndefined(params) {
		return obj
	}

	callable, ok := sobek.AssertFunction(obj.Get("append"))
	if !ok {
		panic(rt.NewTypeError("invalid formData prototype"))
	}
	switch params.ExportType().Kind() {
	case reflect.Map:
		object := params.ToObject(rt)
		for _, key := range object.Keys() {
			_, err := callable(obj, rt.ToValue(key), object.Get(key))
			if err != nil {
				js.Throw(rt, err)
			}
		}
	default:
		values, err := url.ParseQuery(params.String())
		if err != nil {
			js.Throw(rt, err)
		}
		for name, v := range values {
			ele, ok := ret.data[name]
			if !ok {
				ret.keys = append(ret.keys, name)
				ele = make([]sobek.Value, 0, len(v))
			}
			for _, vv := range v {
				ele = append(ele, rt.ToValue(vv))
			}
			ret.data[name] = ele
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

// append adds a new key/value pair to the FormData.
func (*FormData) append(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toFormData(rt, call.This)
	name := call.Argument(0).String()
	value := call.Argument(1)

	ele, ok := this.data[name]
	if !ok {
		this.keys = append(this.keys, name)
		ele = make([]sobek.Value, 0, 1)
	}

	switch value.ExportType() {
	case buffer.TypeBlob, buffer.TypeFile:
		filename := call.Argument(2)
		if f := call.Argument(2); sobek.IsUndefined(f) {
			// Default filename "blob".
			filename = rt.ToValue("blob")
		}

		file := types.New(rt, "File", rt.NewArray(value), filename)
		ele = append(ele, file)
	default:
		if !sobek.IsUndefined(value) {
			ele = append(ele, value)
		}
	}

	this.data[name] = ele

	return sobek.Undefined()
}

// delete removes a key/value pair from the FormData.
func (*FormData) delete(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toFormData(rt, call.This)
	name := call.Argument(0).String()
	this.keys = slices.DeleteFunc(this.keys, func(k string) bool { return k == name })
	delete(this.data, name)
	return sobek.Undefined()
}

// forEach calls a function for each key, value in the FormData.
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

// get returns the value of the first element associated with the given key.
func (*FormData) get(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toFormData(rt, call.This)
	name := call.Argument(0).String()
	if v, ok := this.data[name]; ok {
		if len(v) > 0 {
			return v[0]
		}
	}
	return sobek.Null()
}

// getAll returns an array of values associated with the given key.
func (*FormData) getAll(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toFormData(rt, call.This)
	name := call.Argument(0).String()
	v, ok := this.data[name]
	if ok {
		return rt.ToValue(v)
	}
	return rt.NewArray()
}

// has returns true if the FormData contains the given key.
func (*FormData) has(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toFormData(rt, call.This)
	name := call.Argument(0).String()
	_, ok := this.data[name]
	return rt.ToValue(ok)
}

// set adds a new key/value pair to the FormData.
func (*FormData) set(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toFormData(rt, call.This)
	name := call.Argument(0).String()
	value := call.Argument(1)
	if _, ok := this.data[name]; !ok {
		this.keys = append(this.keys, name)
	}

	switch value.ExportType() {
	case buffer.TypeBlob, buffer.TypeFile:
		filename := call.Argument(2)
		if f := call.Argument(2); sobek.IsUndefined(f) {
			// Default filename "blob".
			filename = rt.ToValue("blob")
		}

		file := types.New(rt, "File", rt.NewArray(value), filename)
		this.data[name] = []sobek.Value{file}
	default:
		if !sobek.IsUndefined(value) {
			this.data[name] = []sobek.Value{value}
		}
	}
	return sobek.Undefined()
}

// keys returns an iterator of keys in the FormData.
func (*FormData) keys(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toFormData(rt, call.This)
	return types.Iterator(rt, func(yield func(any) bool) {
		for _, key := range this.keys {
			if !yield(key) {
				return
			}
		}
	})
}

// values returns an iterator of values in the FormData.
func (*FormData) values(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toFormData(rt, call.This)
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

// entries returns an iterator of key/value pairs in the FormData.
func (*FormData) entries(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toFormData(rt, call.This)
	return types.Iterator(rt, func(yield func(any) bool) {
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

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

// encode encodes the FormData into a reader and content type.
func (f *formData) encode() (io.Reader, string, error) {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)
	for _, key := range f.keys {
		for _, value := range f.data[key] {
			switch value.ExportType() {
			case buffer.TypeFile:
				name := value.(*sobek.Object).Get("name").String()
				data, t, ok := buffer.GetReader(value)
				if !ok {
					continue
				}
				err := f.write(writer, data, key, name, t)
				if err != nil {
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

// write writes a file to the multipart writer.
func (f *formData) write(writer *multipart.Writer, data io.Reader, key, name, t string) error {
	if c, ok := data.(io.Closer); ok {
		defer c.Close()
	}
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
			quoteEscaper.Replace(key), quoteEscaper.Replace(name)))
	h.Set("Content-Type", t)
	part, err := writer.CreatePart(h)
	if err != nil {
		return err
	}
	_, err = io.Copy(part, data)
	return err
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

const (
	defaultMaxMemory = 32 << 20 // 32 MB
)

var (
	errInvalidMimeType = errors.New("Invalid MIME type")
)

// parseFromData parses the body as a FormData.
func parseFromData(body io.Reader, bodyUsed *atomic.Bool, contentType string) (*multipart.Form, error) {
	if bodyUsed.Load() {
		return nil, errBodyAlreadyRead
	}
	d, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, errInvalidMimeType
	}
	switch d {
	case "application/x-www-form-urlencoded":
		if body == nil {
			return new(multipart.Form), nil
		}
		defer func() {
			bodyUsed.Store(true)
			if c, ok := body.(io.Closer); ok {
				c.Close()
			}
		}()
		b, err := io.ReadAll(body)
		if err != nil {
			return nil, err
		}
		query, err := url.ParseQuery(string(b))
		if err != nil {
			return nil, err
		}
		return &multipart.Form{Value: query}, nil
	case "multipart/form-data", "multipart/mixed":
		if body == nil {
			return nil, errInvalidMimeType
		}
	default:
		return nil, errInvalidMimeType
	}
	boundary, ok := params["boundary"]
	if !ok {
		return nil, errInvalidMimeType
	}
	defer func() {
		bodyUsed.Store(true)
		if c, ok := body.(io.Closer); ok {
			c.Close()
		}
	}()
	form, err := multipart.NewReader(body, boundary).ReadForm(defaultMaxMemory)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return new(multipart.Form), nil
		}
		return nil, err
	}
	return form, nil
}

// newFormData creates a new FormData object.
func newFormData(rt *sobek.Runtime, form *multipart.Form) *sobek.Object {
	f := rt.Get("FormData")
	if f == nil {
		panic(rt.NewTypeError("FormData is undefined"))
	}

	var ret formData
	ret.data = make(map[string][]sobek.Value, len(form.Value)+len(form.File))

	for k, v := range form.Value {
		ele, ok := ret.data[k]
		if !ok {
			ret.keys = append(ret.keys, k)
			ele = make([]sobek.Value, 0, len(v))
		}
		for _, vv := range v {
			ele = append(ele, rt.ToValue(vv))
		}
		ret.data[k] = ele
	}

	for k, v := range form.File {
		ele, ok := ret.data[k]
		if !ok {
			ret.keys = append(ret.keys, k)
			ele = make([]sobek.Value, 0, len(v))
		}
		for _, vv := range v {
			f, err := vv.Open()
			if err != nil {
				js.Throw(rt, err)
			}
			file := buffer.NewFile(rt, f, vv.Size, vv.Header.Get("Content-Type"), vv.Filename, 0)
			ele = append(ele, file)
		}
		ret.data[k] = ele
	}

	obj := rt.ToValue(&ret).(*sobek.Object)
	_ = obj.SetPrototype(f.ToObject(rt).Get("prototype").ToObject(rt))
	return obj
}

// EncodeFormData encodes a FormData object.
func EncodeFormData(value sobek.Value) (io.Reader, string, error) {
	if value != nil && value.ExportType() == typeFormData {
		return value.Export().(*formData).encode()
	}
	return nil, "", errors.New("value is not FormData")
}
