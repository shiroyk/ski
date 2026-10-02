package common

import (
	"fmt"

	"github.com/grafana/sobek"
)

type ErrCode string

// Node.js error codes, see https://nodejs.org/api/errors.html
const (
	ErrInvalidArgType ErrCode = "ERR_INVALID_ARG_TYPE"
	ErrMissingArgs    ErrCode = "ERR_MISSING_ARGS"
	ErrInvalidThis    ErrCode = "ERR_INVALID_THIS"
)

// newErrorf creates an Error with a Node.js style code property.
func newErrorf(rt *sobek.Runtime, ctor string, code ErrCode, format string, args ...any) *sobek.Object {
	message := fmt.Sprintf(format, args...)
	callable, ok := sobek.AssertFunction(rt.Get(ctor))
	if !ok {
		panic(rt.NewTypeError(message))
	}
	value, err := callable(sobek.Undefined(), rt.ToValue(message))
	if err != nil {
		panic(rt.NewTypeError(message))
	}
	errObj := value.ToObject(rt)
	_ = errObj.Set("code", string(code))
	return errObj
}

// NewErrorf creates a Error with a Node.js style code property.
func NewErrorf(rt *sobek.Runtime, code ErrCode, format string, args ...any) *sobek.Object {
	return newErrorf(rt, "Error", code, format, args...)
}

// NewTypeErrorf creates a TypeError with a Node.js style code property.
func NewTypeErrorf(rt *sobek.Runtime, code ErrCode, format string, args ...any) *sobek.Object {
	return newErrorf(rt, "TypeError", code, format, args...)
}
