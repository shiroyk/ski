package js

import (
	"context"
	"errors"
	"fmt"
	"iter"

	"github.com/grafana/sobek"
)

// Throw js exception
func Throw(rt *sobek.Runtime, err error) {
	var ex *sobek.Exception
	if errors.As(err, &ex) { //nolint:errorlint
		panic(ex)
	}
	panic(rt.NewGoError(err))
}

// ToBytes tries to return a byte slice from compatible types.
func ToBytes(data any) ([]byte, error) {
	switch dt := data.(type) {
	case []byte:
		return dt, nil
	case string:
		return []byte(dt), nil
	case sobek.ArrayBuffer:
		return dt.Bytes(), nil
	default:
		return nil, fmt.Errorf("expected string, []byte or ArrayBuffer, but got %T, ", data)
	}
}

// Unwrap the sobek.Value to the raw value
func Unwrap(value sobek.Value) (any, error) {
	if value == nil {
		return nil, nil
	}
	switch v := value.Export().(type) {
	default:
		return v, nil
	case sobek.ArrayBuffer:
		return v.Bytes(), nil
	case *sobek.Promise:
		switch v.State() {
		case sobek.PromiseStateRejected:
			return nil, errors.New(v.Result().String())
		case sobek.PromiseStateFulfilled:
			return v.Result().Export(), nil
		default:
			return nil, errors.New("unexpected promise pending state")
		}
	}
}

// Context returns the current context of the sobek.Runtime
func Context(rt *sobek.Runtime) context.Context { return self(rt).ctx }

func FreezeObject(rt *sobek.Runtime, obj sobek.Value) error {
	global := rt.GlobalObject().Get("Object").ToObject(rt)
	freeze, ok := sobek.AssertFunction(global.Get("freeze"))
	if !ok {
		panic("failed to get the Object.freeze function from the runtime")
	}
	_, err := freeze(sobek.Undefined(), obj)
	return err
}

// Iterator returns a JavaScript iterator
func Iterator(rt *sobek.Runtime, seq iter.Seq[any]) *sobek.Object {
	p := rt.NewObject()
	next, _ := iter.Pull(seq)
	_ = p.SetSymbol(sobek.SymIterator, func(call sobek.FunctionCall) sobek.Value { return call.This })
	_ = p.Set("next", func(call sobek.FunctionCall) sobek.Value {
		ret := rt.NewObject()
		value, ok := next()
		_ = ret.Set("value", value)
		_ = ret.Set("done", !ok)
		return ret
	})
	return p
}

// New create a new object from the constructor name
func New(rt *sobek.Runtime, name string, args ...sobek.Value) *sobek.Object {
	ctor := rt.Get(name)
	if ctor == nil {
		panic(rt.NewTypeError("%s is not defined", name))
	}
	o, err := rt.New(ctor, args...)
	if err != nil {
		Throw(rt, err)
	}
	return o
}
