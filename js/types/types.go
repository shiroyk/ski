package types

import (
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
