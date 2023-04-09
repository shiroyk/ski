package common

import (
	"context"
	"errors"
	"fmt"

	"github.com/dop251/goja"
	"github.com/spf13/cast"
)

// Throw js exception
func Throw(vm *goja.Runtime, err error) {
	if e, ok := err.(*goja.Exception); ok { //nolint:errorlint
		panic(e)
	}
	panic(vm.ToValue(err))
}

// ToBytes tries to return a byte slice from compatible types.
func ToBytes(data any) ([]byte, error) {
	switch dt := data.(type) {
	case []byte:
		return dt, nil
	case string:
		return []byte(dt), nil
	case goja.ArrayBuffer:
		return dt.Bytes(), nil
	default:
		return nil, fmt.Errorf("invalid type %T, expected string, []byte or ArrayBuffer", data)
	}
}

// ToStrings tries to return a string slice or string from compatible types.
func ToStrings(data any) (s any, err error) {
	switch dt := data.(type) {
	case string:
		return dt, nil
	case []string:
		return dt, nil
	case []any:
		return cast.ToStringSliceE(dt)
	case goja.ArrayBuffer:
		return string(dt.Bytes()), nil
	default:
		return nil, fmt.Errorf("invalid type %T, expected string, string array or ArrayBuffer", data)
	}
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

// VMContext returns the current context of the goja.Runtime
func VMContext(vm *goja.Runtime) context.Context {
	ctx := context.Background()
	if v := vm.Get(VMContextKey).Export(); v != nil {
		if c, ok := v.(context.Context); ok {
			ctx = c
		}
	}
	return ctx
}
