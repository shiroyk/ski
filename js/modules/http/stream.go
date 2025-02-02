package http

import (
	"io"
	"reflect"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
)

// ReadableStream interface represents a readable stream of byte data.
// https://developer.mozilla.org/en-US/docs/Web/API/ReadableStream
type ReadableStream struct{}

func (r *ReadableStream) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()
	_ = p.DefineAccessorProperty("locked", rt.ToValue(r.locked), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.Set("cancel", r.cancel)
	_ = p.Set("getReader", r.getReader)
	_ = p.Set("tee", r.tee)
	_ = p.Set("pipeTo", r.pipeTo)
	_ = p.Set("pipeThrough", r.pipeThrough)
	_ = p.SetSymbol(sobek.SymToStringTag, func(sobek.ConstructorCall) sobek.Value { return rt.ToValue("ReadableStream") })
	return p
}

func (*ReadableStream) constructor(_ sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	panic(rt.NewTypeError("ReadableStream constructor not implement"))
}

func (*ReadableStream) locked(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toReadableStream(rt, call.This)
	return rt.ToValue(this.reader != nil)
}

func (*ReadableStream) cancel(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toReadableStream(rt, call.This)
	return rt.ToValue(js.NewPromise(rt, func() (any, error) {
		if closer, ok := this.source.(io.Closer); ok {
			if err := closer.Close(); err != nil {
				return nil, err
			}
		}
		this.cancel = true
		this.source = nil
		this.reader = nil
		return sobek.Undefined(), nil
	}))
}

func (*ReadableStream) getReader(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toReadableStream(rt, call.This)
	if this.cancel {
		panic(rt.NewTypeError("stream is already canceled"))
	}

	if this.reader != nil {
		panic(rt.NewTypeError("stream is already locked"))
	}

	reader := &streamReader{
		stream: this,
	}
	this.reader = reader

	obj := rt.ToValue(reader).(*sobek.Object)

	if opts := call.Argument(0); !sobek.IsUndefined(opts) {
		if mode := opts.ToObject(rt).Get("mode"); mode != nil && mode.String() == "byob" {
			readerCtor := rt.Get("ReadableStreamBYOBReader")
			if readerCtor == nil {
				panic(rt.NewTypeError("ReadableStreamBYOBReader is not defined"))
			}
			_ = obj.SetPrototype(readerCtor.ToObject(rt).Prototype())
			return obj
		}
	}

	readerCtor := rt.Get("ReadableStreamDefaultReader")
	if readerCtor == nil {
		panic(rt.NewTypeError("ReadableStreamDefaultReader is not defined"))
	}
	_ = obj.SetPrototype(readerCtor.ToObject(rt).Prototype())
	return obj
}

func (*ReadableStream) tee(_ sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	panic(rt.NewTypeError("tee not implement"))
}

func (*ReadableStream) pipeTo(_ sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	panic(rt.NewTypeError("pipeTo not implement"))
}

func (*ReadableStream) pipeThrough(_ sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	panic(rt.NewTypeError("pipeThrough not implement"))
}

type readableStream struct {
	source io.Reader
	reader *streamReader
	cancel bool
}

type streamReader struct {
	stream *readableStream
}

func (r *streamReader) read(buf []byte) (int, error) {
	if r.stream.source == nil {
		return 0, io.EOF
	}

	n, err := r.stream.source.Read(buf)
	if err != nil && err != io.EOF {
		return 0, err
	}
	return n, err
}

func (r *streamReader) cancel() error {
	if closer, ok := r.stream.source.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			return err
		}
	}
	r.stream.cancel = true
	r.stream.source = nil
	r.stream.reader = nil
	return nil
}

var (
	typeReadableStream = reflect.TypeOf((*readableStream)(nil))
	typeStreamReader   = reflect.TypeOf((*streamReader)(nil))
)

func toReadableStream(rt *sobek.Runtime, value sobek.Value) *readableStream {
	if value.ExportType() == typeReadableStream {
		return value.Export().(*readableStream)
	}
	panic(rt.NewTypeError(`Value of "this" must be of type ReadableStream`))
}

func toStreamReader(rt *sobek.Runtime, value sobek.Value) *streamReader {
	if value.ExportType() == typeStreamReader {
		return value.Export().(*streamReader)
	}
	panic(rt.NewTypeError(`Value of "this" must be of type StreamReader`))
}

// ReadableStreamDefaultReader represents a reader that allows reading chunks of data from a ReadableStream.
// https://developer.mozilla.org/en-US/docs/Web/API/ReadableStreamDefaultReader
type ReadableStreamDefaultReader struct{}

func (r *ReadableStreamDefaultReader) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()
	_ = p.Set("read", r.read)
	_ = p.Set("cancel", r.cancel)
	_ = p.Set("releaseLock", r.releaseLock)
	_ = p.SetSymbol(sobek.SymToStringTag, func(sobek.ConstructorCall) sobek.Value { return rt.ToValue("ReadableStreamDefaultReader") })
	return p
}

func (*ReadableStreamDefaultReader) constructor(_ sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	panic(rt.NewTypeError("ReadableStreamDefaultReader constructor not implement"))
}

func (r *ReadableStreamDefaultReader) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	proto := r.prototype(rt)
	ctor := rt.ToValue(r.constructor).(*sobek.Object)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.Set("prototype", proto)
	_ = ctor.SetPrototype(proto)
	return ctor, nil
}

func (*ReadableStreamDefaultReader) Global() {}

func (*ReadableStreamDefaultReader) read(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toStreamReader(rt, call.This)
	buffer := make([]byte, 1024) // default chunk size
	return rt.ToValue(js.NewPromise(rt, func() (int, error) { return this.read(buffer) },
		func(n int, err error) (any, error) {
			if err != nil {
				if err == io.EOF {
					ret := rt.NewObject()
					_ = ret.Set("done", true)
					_ = ret.Set("value", sobek.Undefined())
					return ret, nil
				}
				return nil, err
			}
			ret := rt.NewObject()
			_ = ret.Set("done", false)
			value, err := rt.New(rt.Get("Uint8Array"), rt.ToValue(rt.NewArrayBuffer(buffer[:n])))
			if err != nil {
				js.Throw(rt, err)
			}
			_ = ret.Set("value", value)
			return ret, nil
		}))
}

func (*ReadableStreamDefaultReader) cancel(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toStreamReader(rt, call.This)
	return rt.ToValue(js.NewPromise(rt, func() (any, error) { return nil, this.cancel() }))
}

func (*ReadableStreamDefaultReader) releaseLock(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toStreamReader(rt, call.This)
	if this.stream != nil {
		this.stream.reader = nil
		this.stream = nil
	}
	return sobek.Undefined()
}

// ReadableStreamBYOBReader represents a reader that allows reading chunks of data from a ReadableStream
// into a developer-supplied buffer.
// https://developer.mozilla.org/en-US/docs/Web/API/ReadableStreamBYOBReader
type ReadableStreamBYOBReader struct{}

func (r *ReadableStreamBYOBReader) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()
	_ = p.Set("read", r.read)
	_ = p.Set("cancel", r.cancel)
	_ = p.Set("releaseLock", r.releaseLock)
	_ = p.SetSymbol(sobek.SymToStringTag, func(sobek.ConstructorCall) sobek.Value { return rt.ToValue("ReadableStreamBYOBReader") })
	return p
}

func (*ReadableStreamBYOBReader) constructor(_ sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	panic(rt.NewTypeError("ReadableStreamBYOBReader constructor not implement"))
}

func (r *ReadableStreamBYOBReader) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	proto := r.prototype(rt)
	ctor := rt.ToValue(r.constructor).(*sobek.Object)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.Set("prototype", proto)
	_ = ctor.SetPrototype(proto)
	return ctor, nil
}

func (*ReadableStreamBYOBReader) Global() {}

func (*ReadableStreamBYOBReader) read(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toStreamReader(rt, call.This)
	if len(call.Arguments) < 1 {
		panic(rt.NewTypeError("ReadableStreamBYOBReader.read requires a buffer argument"))
	}

	data := call.Argument(0)
	var buffer []byte
	switch data.ExportType() {
	case typeArrayBuffer:
		buffer = data.Export().(sobek.ArrayBuffer).Bytes()
	case typeBytes:
		buffer = data.Export().([]byte)
	default:
		panic(rt.NewTypeError("argument must be an ArrayBuffer or Uint8Array"))
	}

	return rt.ToValue(js.NewPromise(rt, func() (int, error) { return this.read(buffer) },
		func(n int, err error) (any, error) {
			if err != nil {
				if err == io.EOF {
					ret := rt.NewObject()
					_ = ret.Set("done", true)
					_ = ret.Set("value", sobek.Undefined())
					return ret, nil
				}
				return nil, err
			}
			ret := rt.NewObject()
			_ = ret.Set("done", false)
			value, err := rt.New(rt.Get("Uint8Array"), rt.ToValue(rt.NewArrayBuffer(buffer[:n])))
			if err != nil {
				js.Throw(rt, err)
			}
			_ = ret.Set("value", value)
			return ret, nil
		}))
}

func (*ReadableStreamBYOBReader) cancel(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toStreamReader(rt, call.This)
	return rt.ToValue(js.NewPromise(rt, func() (any, error) { return nil, this.cancel() }))
}

func (*ReadableStreamBYOBReader) releaseLock(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toStreamReader(rt, call.This)
	if this.stream != nil {
		this.stream.reader = nil
		this.stream = nil
	}
	return sobek.Undefined()
}

func (r *ReadableStream) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	proto := r.prototype(rt)
	ctor := rt.ToValue(r.constructor).(*sobek.Object)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.Set("prototype", proto)
	_ = ctor.SetPrototype(proto)
	return ctor, nil
}

func (*ReadableStream) Global() {}
