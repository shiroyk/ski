package buffer

import (
	"bytes"
	"io"
	"reflect"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/promise"
)

var (
	TypeBlob        = reflect.TypeOf((*blob)(nil))
	TypeBytes       = reflect.TypeOf(([]byte)(nil))
	TypeArrayBuffer = reflect.TypeOf(sobek.ArrayBuffer{})
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
	_ = p.SetSymbol(sobek.SymToStringTag, func(sobek.FunctionCall) sobek.Value { return rt.ToValue("Blob") })
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

	var err error
	rt.ForOf(blobParts, func(part sobek.Value) bool {
		if buffer, ok := GetBuffer(rt, part); ok {
			_, err = buf.Write(buffer)
		} else {
			_, err = buf.WriteString(part.String())
		}
		if err != nil {
			js.Throw(rt, err)
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
	return promise.New(rt, func(callback promise.Callback) {
		data, err := this.read()
		callback(func() (any, error) {
			if err != nil {
				return nil, err
			}
			return rt.NewArrayBuffer(data), nil
		})
	})
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
	return promise.New(rt, func(callback promise.Callback) {
		data, err := this.read()
		callback(func() (any, error) {
			if err != nil {
				return nil, err
			}
			return string(data), nil
		})
	})
}

type blob struct {
	data  Reader
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

type Reader interface {
	io.Reader
	io.ReaderAt
	io.Seeker
}

func (b *Blob) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	proto := b.prototype(rt)
	ctor := rt.ToValue(b.constructor).(*sobek.Object)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.Set("prototype", proto)
	_ = ctor.SetPrototype(proto)
	return ctor, nil
}

func toBlob(rt *sobek.Runtime, value sobek.Value) *blob {
	switch value.ExportType() {
	case TypeBlob:
		return value.Export().(*blob)
	case TypeFile:
		return value.Export().(*file).blob
	}
	panic(rt.NewTypeError(`Value of "this" must be of type Blob`))
}
