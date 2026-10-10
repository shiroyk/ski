package nodetest

import (
	"math"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js/types"
	"github.com/shiroyk/ski/modules/common"
)

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

	// node validates the INSPECT_MAX_BYTES assignment
	maxBytes := float64(50)
	_ = bufferCtor.DefineAccessorProperty("INSPECT_MAX_BYTES",
		rt.ToValue(func(sobek.FunctionCall, *sobek.Runtime) sobek.Value {
			return rt.ToValue(maxBytes)
		}),
		rt.ToValue(func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
			value := call.Argument(0)
			if !types.IsNumber(value) {
				panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
					`The "INSPECT_MAX_BYTES" property must be of type number. Received type %s (%s)`,
					types.ValueType(value), value.String()))
			}
			number := value.ToFloat()
			if math.IsNaN(number) || number < 0 {
				panic(common.NewRangeErrorf(rt, common.ErrOutOfRange,
					`The value of "INSPECT_MAX_BYTES" is out of range. It must be >= 0. Received %v`, number))
			}
			maxBytes = number
			return sobek.Undefined()
		}), sobek.FLAG_TRUE, sobek.FLAG_TRUE)

	_ = bufferCtor.Set("Blob", rt.Get("Blob"))
	_ = bufferCtor.Set("File", rt.Get("File"))
	_ = bufferCtor.Set("atob", rt.Get("atob"))
	_ = bufferCtor.Set("btoa", rt.Get("btoa"))
	return bufferCtor
}
