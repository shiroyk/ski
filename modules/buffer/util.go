package buffer

import (
	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js/types"
)

// GetReader extracts the underlying Reader from a Blob or File.
// false if value is not a Blob or File.
func GetReader(value sobek.Value) (Reader, bool) {
	switch value.ExportType() {
	case TypeBlob:
		return value.Export().(*blob).data, true
	case TypeFile:
		return value.Export().(*file).data, true
	default:
		return nil, false
	}
}

// IsBuffer returns true if the value is a IsBuffer.
func IsBuffer(rt *sobek.Runtime, value sobek.Value) bool {
	if value.ToObject(rt).GetSymbol(symBuffer) == symBuffer {
		return true
	}
	return false
}

// GetBuffer returns the underlying byte buffer from a ArrayBuffer, TypedArray, DataView, Buffer.
func GetBuffer(rt *sobek.Runtime, value sobek.Value) ([]byte, bool) {
	switch value.ExportType() {
	case types.TypeArrayBuffer:
		return value.Export().(sobek.ArrayBuffer).Bytes(), true
	default:
		switch {
		case IsBuffer(rt, value):
			return value.Export().([]byte), true
		case rt.InstanceOf(value, rt.Get("DataView").(*sobek.Object)):
			fallthrough
		case types.IsTypedArray(rt, value):
			buffer, ok := value.ToObject(rt).Get("buffer").Export().(sobek.ArrayBuffer)
			if !ok {
				panic(rt.NewTypeError("TypedArray buffer is not an ArrayBuffer"))
			}
			return buffer.Bytes(), true
		}
	}
	return nil, false
}
