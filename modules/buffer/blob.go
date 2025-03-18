package buffer

import (
	"bytes"
	"io"
	"reflect"
	"strings"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/promise"
	"github.com/shiroyk/ski/js/types"
	"github.com/shiroyk/ski/modules/stream"
)

var (
	TypeBlob = reflect.TypeOf((*blob)(nil))
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
	_ = p.Set("bytes", b.bytes)
	_ = p.Set("stream", b.stream)
	_ = p.SetSymbol(sobek.SymToStringTag, func(sobek.FunctionCall) sobek.Value { return rt.ToValue("Blob") })
	return p
}

func (b *Blob) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {

	var buf bytes.Buffer

	ret := &blob{}

	if len(call.Arguments) > 0 {
		blobParts := call.Argument(0)
		if sobek.IsUndefined(blobParts) {
			panic(rt.NewTypeError("Blob must have a callable @iterator property"))
		}
		var (
			data []byte
			err  error
		)
		rt.ForOf(blobParts, func(part sobek.Value) bool {
			if r, t, ok := GetReader(part); ok {
				data, err = ReadAll(r)
				ret.type_ = strings.ToLower(t)
			} else if v, ok := GetBuffer(rt, part); ok {
				data = v
			} else {
				data = []byte(part.String())
			}
			if err != nil {
				js.Throw(rt, err)
			}
			buf.Write(data)
			return true
		})
	}

	ret.data = bytes.NewReader(buf.Bytes())
	ret.size = int64(buf.Len())

	if opts := call.Argument(1); !sobek.IsUndefined(opts) {
		options := opts.ToObject(rt)
		if t := options.Get("type"); t != nil {
			ret.type_ = strings.ToLower(t.String())
		}
	}

	obj := rt.ToValue(ret).(*sobek.Object)
	_ = obj.SetPrototype(call.This.Prototype())
	return obj
}

// size returns the size of the blob
func (*Blob) size(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toBlob(rt, call.This).size)
}

// type returns the type of the blob
func (*Blob) type_(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toBlob(rt, call.This).type_)
}

// slice returns a new Blob object which contains data from a subset of the blob on which it's called.
func (*Blob) slice(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBlob(rt, call.This)
	start := 0
	size := int(this.size)
	end := size
	contentType := ""

	if v := call.Argument(0); !sobek.IsUndefined(v) {
		start = int(v.ToInteger())
		if start < 0 {
			start = max(size+start, 0)
		} else {
			start = min(start, size)
		}
	}
	if v := call.Argument(1); !sobek.IsUndefined(v) {
		end = int(v.ToInteger())
		if end < 0 {
			end = max(size+end, 0)
		} else {
			end = min(end, size)
		}
	}
	if v := call.Argument(2); !sobek.IsUndefined(v) {
		s := v.String()
		if !strings.ContainsFunc(s, invalidContentType) {
			contentType = strings.ToLower(s)
		}
	}

	span := max(end-start, 0)
	b := &blob{
		type_: contentType,
	}

	if span > 0 {
		data := make([]byte, span)
		_, err := this.data.ReadAt(data, int64(start))
		if err != nil && err != io.EOF {
			js.Throw(rt, err)
		}
		b.data = bytes.NewReader(data)
		b.size = int64(span)
	} else {
		b.data = bytes.NewReader(nil)
		b.type_ = contentType
	}

	obj := rt.ToValue(b).(*sobek.Object)
	_ = obj.SetPrototype(call.This.ToObject(rt).Prototype())
	return obj
}

// arrayBuffer returns a promise that resolves with the Blob as an ArrayBuffer.
func (*Blob) arrayBuffer(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBlob(rt, call.This)
	return promise.New(rt, func(callback promise.Callback) {
		data, err := ReadAll(this.data)
		callback(func() (any, error) {
			if err != nil {
				return nil, err
			}
			return rt.NewArrayBuffer(data), nil
		})
	})
}

// text returns a promise which resolves with the Blob as a string
func (*Blob) text(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBlob(rt, call.This)
	return promise.New(rt, func(callback promise.Callback) {
		data, err := ReadAll(this.data)
		callback(func() (any, error) {
			if err != nil {
				return nil, err
			}
			return string(data), nil
		})
	})
}

// bytes returns a promise which resolves with the Blob as a Uint8Array.
func (*Blob) bytes(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBlob(rt, call.This)
	return promise.New(rt, func(callback promise.Callback) {
		data, err := ReadAll(this.data)
		callback(func() (any, error) {
			if err != nil {
				return nil, err
			}
			return types.New(rt, "Uint8Array", rt.ToValue(rt.NewArrayBuffer(data))), nil
		})
	})
}

// stream returns a new ReadableStream.
func (*Blob) stream(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBlob(rt, call.This)
	return stream.NewReadableStream(rt, this.data)
}

type blob struct {
	data  Reader
	size  int64
	type_ string
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
	return ctor, nil
}

func invalidContentType(r rune) bool {
	return r < 0x20 || r > 0x7e
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

// NewBlob returns a new Blob object.
func NewBlob(rt *sobek.Runtime, data Reader, size int64, type_ string) sobek.Value {
	b := rt.Get("Blob")
	if b == nil {
		panic(rt.NewTypeError("Blob is undefined"))
	}
	ret := &blob{data, size, type_}
	obj := rt.ToValue(ret).(*sobek.Object)
	_ = obj.SetPrototype(b.ToObject(rt).Get("prototype").ToObject(rt))
	return obj
}
