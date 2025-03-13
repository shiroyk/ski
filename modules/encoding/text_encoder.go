package encoding

import (
	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
)

// TextEncoder takes a stream of code points as input and emits a stream of UTF-8 bytes.
// https://developer.mozilla.org/en-US/docs/Web/API/TextEncoder
type TextEncoder struct{}

func (t *TextEncoder) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()
	_ = p.DefineAccessorProperty("encoding", rt.ToValue(t.encoding), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.Set("encode", t.encode)
	_ = p.Set("encodeInto", t.encodeInto)
	_ = p.SetSymbol(sobek.SymToStringTag, func(sobek.FunctionCall) sobek.Value { return rt.ToValue("TextEncoder") })
	return p
}

func (t *TextEncoder) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	obj := rt.NewObject()
	_ = obj.SetPrototype(call.This.Prototype())
	return obj
}

func (*TextEncoder) encoding(_ sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue("utf-8")
}

func (*TextEncoder) encode(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	var text string
	if v := call.Argument(0); !sobek.IsUndefined(v) {
		text = v.String()
	}
	return js.New(rt, "Uint8Array", rt.ToValue(rt.NewArrayBuffer([]byte(text))))
}

func (*TextEncoder) encodeInto(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if len(call.Arguments) < 2 {
		panic(rt.NewTypeError("TextEncoder.encodeInto requires 2 arguments"))
	}

	text := call.Argument(0).String()
	dest := call.Argument(1).ToObject(rt)
	if ctor := dest.Get("constructor"); ctor != rt.Get("Uint8Array") {
		panic(rt.NewTypeError("argument 2 must be a Uint8Array"))
	}

	buffer := dest.Export().([]byte)
	bytes := []byte(text)
	written := copy(buffer, bytes)

	result := rt.NewObject()
	_ = result.Set("read", written)
	_ = result.Set("written", written)
	return result
}

// Instantiate module
func (t *TextEncoder) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	proto := t.prototype(rt)
	ctor := rt.ToValue(t.constructor).(*sobek.Object)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.Set("prototype", proto)
	return ctor, nil
}
