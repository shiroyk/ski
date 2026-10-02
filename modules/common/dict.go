package common

import (
	"errors"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js/types"
)

// ExportDict exports a JavaScript dictionary argument into T, following the
// WebIDL dictionary conversion used by Node.js: undefined and null are an empty
// dictionary, arrays, functions and every non object value are rejected with
// ERR_INVALID_ARG_TYPE, everything else is read property by property. Errors
// thrown by a getter of the dictionary are propagated.
//
// name is the argument name used in the error message, e.g. "options".
func ExportDict[T any](rt *sobek.Runtime, dict sobek.Value, name string) T {
	var value T
	if types.IsNil(dict) {
		return value
	}
	if !isDictionary(dict) {
		panic(dictTypeError(rt, name, dict))
	}
	if err := rt.ExportTo(dict, &value); err != nil {
		var ex *sobek.Exception
		if errors.As(err, &ex) {
			// Re-throw errors raised by getters of the dictionary.
			panic(ex)
		}
		panic(dictTypeError(rt, name, dict))
	}
	return value
}

// isDictionary reports whether the value can be used as a WebIDL dictionary,
// Node.js rejects arrays and functions.
func isDictionary(value sobek.Value) bool {
	obj, ok := value.(*sobek.Object)
	if !ok {
		return false
	}
	switch obj.ClassName() {
	case "Array", "Function":
		return false
	}
	return true
}

func dictTypeError(rt *sobek.Runtime, name string, value sobek.Value) *sobek.Object {
	return NewTypeErrorf(rt, ErrInvalidArgType,
		`The %q argument must be of type object. Received type %s`, name, types.ValueType(value))
}
