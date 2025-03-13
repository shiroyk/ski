package buffer

import (
	"io"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js/types"
)

// GetReader extracts the underlying Reader and type from a Blob or File.
// false if value is not a Blob or File.
func GetReader(value sobek.Value) (Reader, string, bool) {
	switch value.ExportType() {
	case TypeBlob:
		b := value.Export().(*blob)
		return b.data, b.type_, true
	case TypeFile:
		f := value.Export().(*file)
		return f.data, f.type_, true
	default:
		return nil, "", false
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
	case types.TypeNil:
		return nil, false
	case types.TypeArrayBuffer:
		return value.Export().(sobek.ArrayBuffer).Bytes(), true
	default:
		switch {
		case IsBuffer(rt, value):
			return value.Export().([]byte), true
		case rt.InstanceOf(value, rt.Get("DataView").(*sobek.Object)):
			fallthrough
		case types.IsTypedArray(rt, value):
			array := value.ToObject(rt)
			b, ok := array.Get("buffer").Export().(sobek.ArrayBuffer)
			if !ok {
				panic(rt.NewTypeError("TypedArray buffer is not an ArrayBuffer"))
			}
			byteLength := array.Get("byteLength").ToInteger()
			byteOffset := array.Get("byteOffset").ToInteger()
			bytes := b.Bytes()
			return bytes[byteOffset : byteOffset+byteLength], true
		}
	}
	return nil, false
}

// ReadAll reads all data from io.ReaderAt.
func ReadAll(r io.ReaderAt) ([]byte, error) {
	s := make([]byte, 0, 512)
	off := int64(0)
	for {
		n, err := r.ReadAt(s[len(s):cap(s)], off)
		s = s[:len(s)+n]
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return s, err
		}
		off = int64(n)

		if len(s) == cap(s) {
			// Add more capacity (let append pick how much).
			s = append(s, 0)[:len(s)]
		}
	}
}
