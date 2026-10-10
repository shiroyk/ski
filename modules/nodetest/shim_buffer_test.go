package nodetest

import "github.com/grafana/sobek"

// newBufferShim returns the `buffer` module, node returns the Buffer
// constructor itself with the module properties attached.
func newBufferShim(rt *sobek.Runtime) *sobek.Object {
	bufferCtor := rt.Get("Buffer").ToObject(rt)
	const (
		maxLength       = int64(9007199254740991)
		maxStringLength = int64(536870888)
	)
	constants := rt.NewObject()
	_ = constants.Set("MAX_LENGTH", maxLength)
	_ = constants.Set("MAX_STRING_LENGTH", maxStringLength)

	_ = bufferCtor.Set("Buffer", bufferCtor)
	_ = bufferCtor.Set("kMaxLength", maxLength)
	_ = bufferCtor.Set("kStringMaxLength", maxStringLength)
	_ = bufferCtor.Set("constants", constants)
	_ = bufferCtor.Set("INSPECT_MAX_BYTES", 50)
	_ = bufferCtor.Set("Blob", rt.Get("Blob"))
	_ = bufferCtor.Set("File", rt.Get("File"))
	_ = bufferCtor.Set("atob", rt.Get("atob"))
	_ = bufferCtor.Set("btoa", rt.Get("btoa"))
	return bufferCtor
}
