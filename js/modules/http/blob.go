package http

import (
	"bytes"
	"io"
	"reflect"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
)

// Blob interface represents a blob, which is a file-like object of immutable,
// raw data; they can be read as text or binary data, or converted into a ReadableStream
// so its methods can be used for processing the data.
// https://developer.mozilla.org/en-US/docs/Web/API/Blob
type Blob struct{}

func (b *Blob) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()
	_ = p.DefineAccessorProperty("size", rt.ToValue(b.size), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("type", rt.ToValue(b.type_), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.Set("arrayBuffer", b.arrayBuffer)
	_ = p.Set("slice", b.slice)
	_ = p.Set("text", b.text)
	_ = p.SetSymbol(sobek.SymToStringTag, func(sobek.ConstructorCall) sobek.Value { return rt.ToValue("Blob") })
	return p
}

func (b *Blob) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("Blob constructor requires at least 1 arguments"))
	}

	blobParts := call.Argument(0)
	if sobek.IsUndefined(blobParts) {
		panic(rt.NewTypeError("Blob must have a callable @iterator property"))
	}
	buf := new(bytes.Buffer)

	rt.ForOf(blobParts, func(part sobek.Value) (ok bool) {
		switch part.ExportType() {
		case typeBlob:
			blob := part.Export().(*blob)
			if _, err := io.Copy(buf, blob.data); err != nil {
				js.Throw(rt, err)
			}
		case typeBytes:
			if _, err := buf.Write(part.Export().([]byte)); err != nil {
				js.Throw(rt, err)
			}
		case typeArrayBuffer:
			if _, err := buf.Write(part.Export().(sobek.ArrayBuffer).Bytes()); err != nil {
				js.Throw(rt, err)
			}
		default:
			if _, err := buf.WriteString(part.String()); err != nil {
				js.Throw(rt, err)
			}
		}
		return true
	})

	blob := &blob{
		data: bytes.NewReader(buf.Bytes()),
		size: int64(buf.Len()),
	}

	if opts := call.Argument(1); !sobek.IsUndefined(opts) {
		options := opts.ToObject(rt)
		if t := options.Get("type"); !sobek.IsUndefined(t) {
			blob.type_ = t.String()
		}
	}

	obj := rt.ToValue(blob).(*sobek.Object)
	_ = obj.SetPrototype(call.This.Prototype())
	return obj
}

func (*Blob) size(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toBlob(rt, call.This).size)
}

func (*Blob) type_(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toBlob(rt, call.This).type_)
}

func (*Blob) arrayBuffer(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBlob(rt, call.This)
	return rt.ToValue(js.NewPromise(rt, this.read, func(data []byte, err error) (any, error) {
		if err != nil {
			return nil, err
		}
		return rt.NewArrayBuffer(data), nil
	}))
}

func (*Blob) slice(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBlob(rt, call.This)
	start := 0
	size := int(this.size)
	end := size
	contentType := this.type_

	if v := call.Argument(0); !sobek.IsUndefined(v) {
		start = int(v.ToInteger())
	}
	if v := call.Argument(1); !sobek.IsUndefined(v) {
		end = int(v.ToInteger())
	}
	if v := call.Argument(2); !sobek.IsUndefined(v) {
		contentType = v.String()
	}

	if start < 0 {
		start = size + start
	}
	if end < 0 {
		end = size + end
	}

	if start < 0 {
		start = 0
	}
	if end > size {
		end = size
	}
	if start > end {
		start = end
	}

	data := make([]byte, end-start)
	_, err := this.data.ReadAt(data, int64(start))
	if err != nil {
		js.Throw(rt, err)
	}

	obj := rt.ToValue(&blob{
		data:  bytes.NewReader(data),
		type_: contentType,
	}).(*sobek.Object)
	_ = obj.SetPrototype(call.This.ToObject(rt).Prototype())
	return obj
}

func (*Blob) text(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBlob(rt, call.This)
	return rt.ToValue(js.NewPromise(rt, this.read, func(data []byte, err error) (any, error) {
		if err != nil {
			return nil, err
		}
		return string(data), nil
	}))
}

type blob struct {
	data  blobData
	size  int64
	type_ string
}

func (b *blob) read() ([]byte, error) {
	err := b.reset()
	if err != nil {
		return nil, err
	}
	return io.ReadAll(b.data)
}

func (b *blob) reset() error {
	_, err := b.data.Seek(0, io.SeekStart)
	return err
}

type blobData interface {
	io.Reader
	io.ReaderAt
	io.Seeker
}

func toBlob(rt *sobek.Runtime, value sobek.Value) *blob {
	switch value.ExportType() {
	case typeBlob:
		return value.Export().(*blob)
	case typeFile:
		return toBlob(rt, value.Export().(*file).blob)
	}
	panic(rt.NewTypeError(`Value of "this" must be of type Blob`))
}

var (
	typeBlob        = reflect.TypeOf((*blob)(nil))
	typeBytes       = reflect.TypeOf(([]byte)(nil))
	typeArrayBuffer = reflect.TypeOf(sobek.ArrayBuffer{})
)

func (b *Blob) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	proto := b.prototype(rt)
	ctor := rt.ToValue(b.constructor).(*sobek.Object)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.Set("prototype", proto)
	_ = ctor.SetPrototype(proto)
	return ctor, nil
}

func (*Blob) Global() {}
