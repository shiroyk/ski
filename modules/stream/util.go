package stream

import (
	"io"

	"github.com/grafana/sobek"
)

// IsLocked returns ReadableStream is locked.
func IsLocked(value sobek.Value) bool {
	if value != nil && value.ExportType() == TypeReadableStream {
		return value.Export().(*readableStream).locked()
	}
	return false
}

// IsDisturbed returns ReadableStream has been read from or canceled.
func IsDisturbed(value sobek.Value) bool {
	if value != nil && value.ExportType() == TypeReadableStream {
		return value.Export().(*readableStream).disturbed.Load()
	}
	return false
}

// GetStreamSource extracts the underlying io.Reader from a ReadableStream.
func GetStreamSource(rt *sobek.Runtime, value sobek.Value) io.Reader {
	if value.ExportType() == TypeReadableStream {
		return value.Export().(*readableStream).source
	}
	panic(rt.NewTypeError(`Value is not a ReadableStream`))
}
