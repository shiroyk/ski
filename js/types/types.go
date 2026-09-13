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

var (
	IsNumber = sobek.IsNumber
	IsString = sobek.IsString
	IsBigInt = sobek.IsBigInt
)

// IsFunc check value is a function
func IsFunc(value sobek.Value) bool {
	if value == nil {
		return false
	}
	return value.ExportType() == TypeFunc
}

// IsNil check value is nil or null or undefined.
func IsNil(v sobek.Value) bool {
	return v == nil || sobek.IsNull(v) || sobek.IsUndefined(v)
}

// IsPromise check value is sobek.Promise.
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

// ToString returns the string of the value, empty string for nil, null or undefined
func ToString(value sobek.Value) string {
	if IsNil(value) {
		return ""
	}
	return value.String()
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

// Iterate returns a sequence of the elements of the value and the number of
// elements the sequence yields.
// nil, null and undefined return an empty sequence.
// Slice, array and typed array values (e.g. Array, Uint8Array) yield each
// element following their own enumerable keys, so sparse arrays skip the
// missing indexes.
// Any other value yields itself once, which allows the caller to handle both
// arrays and single values with one loop.
func Iterate(value sobek.Value) (iter.Seq[sobek.Value], int) {
	if IsNil(value) {
		return func(yield func(sobek.Value) bool) {}, 0
	}
	switch value.ExportType().Kind() {
	case reflect.Slice, reflect.Array:
		object := value.(*sobek.Object)
		idx := object.Keys()
		return func(yield func(sobek.Value) bool) {
			for _, i := range idx {
				if !yield(object.Get(i)) {
					return
				}
			}
		}, len(idx)
	default:
		return func(yield func(sobek.Value) bool) { yield(value) }, 1
	}
}

// Iterate2 returns a sequence of the key-value pairs of the value and the
// number of pairs the sequence yields.
// Only map values (e.g. a plain JavaScript object or a Go map) are iterated,
// following their own enumerable keys, so array indexes and non-enumerable
// properties are excluded.
// nil, null, undefined and any non-map value return an empty sequence, use
// Iterate if a non-map value should be yielded once instead of being skipped.
func Iterate2(value sobek.Value) (iter.Seq2[string, sobek.Value], int) {
	switch {
	case IsNil(value), value.ExportType().Kind() != reflect.Map:
		return func(yield func(string, sobek.Value) bool) {}, 0
	}
	object := value.(*sobek.Object)
	keys := object.Keys()
	return func(yield func(string, sobek.Value) bool) {
		for _, key := range keys {
			if !yield(key, object.Get(key)) {
				return
			}
		}
	}, len(keys)
}
