package buffer

import (
	"reflect"
	"time"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
)

// File object is a specific kind of Blob, and can be used in any context that a Blob can.
// https://developer.mozilla.org/en-US/docs/Web/API/File
type File struct{}

func (f *File) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()
	b := rt.Get("Blob")
	if b == nil {
		panic(rt.NewTypeError("Blob is undefined"))
	}
	proto := b.ToObject(rt).Get("prototype").ToObject(rt)
	_ = p.SetPrototype(proto)
	_ = p.DefineAccessorProperty("name", rt.ToValue(f.name), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("lastModified", rt.ToValue(f.lastModified), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("webkitRelativePath", rt.ToValue(f.webkitRelativePath), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.SetSymbol(sobek.SymToStringTag, func(sobek.FunctionCall) sobek.Value { return rt.ToValue("File") })
	return p
}

func (f *File) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	if len(call.Arguments) < 2 {
		panic(rt.NewTypeError("File constructor requires at least 2 arguments"))
	}

	options := call.Argument(2)
	b := js.New(rt, "Blob", call.Argument(0), options)

	filename := call.Argument(1).String()

	lastModified := time.Now().UnixMilli()
	if !sobek.IsUndefined(options) {
		if lm := options.ToObject(rt).Get("lastModified"); lm != nil {
			lastModified = lm.ToInteger()
		}
	}

	obj := rt.ToValue(&file{
		blob:         b.Export().(*blob),
		name:         filename,
		lastModified: lastModified,
	}).(*sobek.Object)
	_ = obj.SetPrototype(call.This.Prototype())
	return obj
}

func (*File) name(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toFile(rt, call.This)
	return rt.ToValue(this.name)
}

func (*File) lastModified(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toFile(rt, call.This)
	return rt.ToValue(this.lastModified)
}

func (*File) webkitRelativePath(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toFile(rt, call.This)
	return rt.ToValue(this.webkitRelativePath)
}

type file struct {
	*blob
	name               string
	webkitRelativePath string
	lastModified       int64
}

var TypeFile = reflect.TypeOf((*file)(nil))

func toFile(rt *sobek.Runtime, value sobek.Value) *file {
	if value.ExportType() == TypeFile {
		return value.Export().(*file)
	}
	panic(rt.NewTypeError(`Value of "this" must be of type File`))
}

func (f *File) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	proto := f.prototype(rt)
	ctor := rt.ToValue(f.constructor).(*sobek.Object)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.Set("prototype", proto)
	return ctor, nil
}
