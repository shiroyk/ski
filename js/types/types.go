package types

import (
	"iter"
	"reflect"

	"github.com/grafana/sobek"
)

var (
	TypeFunc        = reflect.TypeOf((func(sobek.FunctionCall) sobek.Value)(nil))
	TypeInt         = reflect.TypeOf(int64(0))
	TypeFloat       = reflect.TypeOf(0.0)
	TypeString      = reflect.TypeOf("")
	TypeBytes       = reflect.TypeOf(([]byte)(nil))
	TypeArrayBuffer = reflect.TypeOf(sobek.ArrayBuffer{})
	TypeError       = reflect.TypeOf((*error)(nil)).Elem()
	TypePromise     = reflect.TypeOf((*sobek.Promise)(nil))
	TypeNil         = reflect.TypeOf(nil)
)

func IsFunc(value sobek.Value) bool {
	if value == nil {
		return false
	}
	return value.ExportType() == TypeFunc
}

func IsNumber(value sobek.Value) bool {
	if value == nil {
		return false
	}
	return value.ExportType() == TypeInt || value.ExportType() == TypeFloat
}

func IsInt(value sobek.Value) bool {
	if value == nil {
		return false
	}
	return value.ExportType() == TypeInt
}

func IsString(value sobek.Value) bool {
	if value == nil {
		return false
	}
	return value.ExportType() == TypeString
}

func IsPromise(value sobek.Value) bool {
	if value == nil {
		return false
	}
	return value.ExportType() == TypePromise
}

var typedArrayTypes = []string{
	"Int8Array", "Uint8Array", "Uint8ClampedArray",
	"Int16Array", "Uint16Array",
	"Int32Array", "Uint32Array",
	"Float32Array", "Float64Array",
	"BigInt64Array", "BigUint64Array",
}

// IsTypedArray returns true if the value is a TypedArray.
func IsTypedArray(rt *sobek.Runtime, value sobek.Value) bool {
	for _, typ := range typedArrayTypes {
		if rt.InstanceOf(value, rt.Get(typ).(*sobek.Object)) {
			return true
		}
	}
	return false
}

// IsUint8Array returns true if the value is a Uint8Array.
func IsUint8Array(rt *sobek.Runtime, value sobek.Value) bool {
	if rt.InstanceOf(value, rt.Get("Uint8Array").(*sobek.Object)) {
		return true
	}
	return false
}

// Iterator returns a JavaScript iterator
func Iterator(rt *sobek.Runtime, seq iter.Seq[any]) *sobek.Object {
	p := rt.NewObject()
	next, _ := iter.Pull(seq)
	_ = p.SetSymbol(sobek.SymIterator, func(call sobek.FunctionCall) sobek.Value { return call.This })
	_ = p.Set("next", func(call sobek.FunctionCall) sobek.Value {
		ret := rt.NewObject()
		value, ok := next()
		_ = ret.Set("value", value)
		_ = ret.Set("done", !ok)
		return ret
	})
	return p
}

// New create a new object from the constructor name
func New(rt *sobek.Runtime, name string, args ...sobek.Value) *sobek.Object {
	ctor := rt.Get(name)
	if ctor == nil {
		panic(rt.NewTypeError("%s is not defined", name))
	}
	o, err := rt.New(ctor, args...)
	if err != nil {
		panic(err)
	}
	return o
}
