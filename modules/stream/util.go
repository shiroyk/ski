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

// IsClosed returns ReadableStream is closed.
func IsClosed(value sobek.Value) bool {
	if value != nil && value.ExportType() == TypeReadableStream {
		return value.Export().(*readableStream).closed.Load()
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
