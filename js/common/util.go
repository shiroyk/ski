package common

import (
	"errors"
	"fmt"

	"github.com/dop251/goja"
)

// Throw js exception
func Throw(rt *goja.Runtime, err error) {
	var e *goja.Exception
	if ok := errors.Is(err, e); ok {
		panic(e)
	}
	panic(rt.NewGoError(err))
}

// Unwrap the goja.Value to the raw value
func Unwrap(value goja.Value) (any, error) {
	switch v := value.Export().(type) {
	default:
		return v, nil
	case goja.ArrayBuffer:
		return v.Bytes(), nil
	case *goja.Promise:
		switch v.State() {
		case goja.PromiseStateRejected:
			return nil, errors.New(v.Result().String())
		case goja.PromiseStateFulfilled:
			return v.Result().Export(), nil
		default:
			return nil, fmt.Errorf("unexpected promise state: %v", v.State())
		}
	}
}
