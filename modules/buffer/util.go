package buffer

import (
	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
)

// GetReader extracts the underlying Reader from a Blob or File.
// panic if value is not a Blob or File.
func GetReader(rt *sobek.Runtime, value sobek.Value) Reader {
	switch value.ExportType() {
	case TypeBlob:
		return value.Export().(*blob).data
	case TypeFile:
		return value.Export().(*file).data
	default:
		panic(rt.NewTypeError(`Value is not a Blob`))
	}
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

// IsBuffer returns true if the value is a IsBuffer.
func IsBuffer(rt *sobek.Runtime, value sobek.Value) bool {
	if value.ToObject(rt).GetSymbol(symBuffer) == symBuffer {
		return true
	}
	return false
}

// GetBuffer returns the underlying byte buffer from a ArrayBuffer, Blob, File, TypedArray, DataView, Buffer.
func GetBuffer(rt *sobek.Runtime, value sobek.Value) ([]byte, bool) {
	switch value.ExportType() {
	case TypeBlob, TypeFile:
		data, err := toBlob(rt, value).read()
		if err != nil {
			js.Throw(rt, err)
		}
		return data, true
	case TypeArrayBuffer:
		return value.Export().(sobek.ArrayBuffer).Bytes(), true
	default:
		switch {
		case IsBuffer(rt, value):
			return value.Export().([]byte), true
		case rt.InstanceOf(value, rt.Get("DataView").(*sobek.Object)):
			fallthrough
		case IsTypedArray(rt, value):
			buffer, ok := value.ToObject(rt).Get("buffer").Export().(sobek.ArrayBuffer)
			if !ok {
				panic(rt.NewTypeError("TypedArray buffer is not an ArrayBuffer"))
			}
			return buffer.Bytes(), true
		}
	}
	return nil, false
}
