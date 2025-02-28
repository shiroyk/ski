// Package modulestest the module test vm
package modulestest

import (
	"context"
	"errors"
	"testing"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/stretchr/testify/assert"
)

type VM struct{ js.VM }

func (vm *VM) RunModule(ctx context.Context, source string, args ...any) (ret sobek.Value, err error) {
	module, err := js.CompileModule("", source)
	if err != nil {
		return
	}
	return vm.VM.RunModule(ctx, module, args...)
}

// New returns a test VM instance
func New(t testing.TB, opts ...js.Option) VM {
	vm := js.NewVM(opts...)
	obj := vm.Runtime().NewObject()
	_ = obj.Set("equal", func(call sobek.FunctionCall, vm *sobek.Runtime) (ret sobek.Value) {
		a, err := js.Unwrap(call.Argument(0))
		if err != nil {
			js.Throw(vm, err)
		}
		b, err := js.Unwrap(call.Argument(1))
		if err != nil {
			js.Throw(vm, err)
		}
		var msg string
		if !sobek.IsUndefined(call.Argument(2)) {
			msg = call.Argument(2).String()
		}
		if !assert.Equal(t, b, a, msg) {
			js.Throw(vm, errors.New("not equal"))
		}
		return
	})
	_ = obj.Set("regexp", func(call sobek.FunctionCall, vm *sobek.Runtime) (ret sobek.Value) {
		a, err := js.Unwrap(call.Argument(0))
		if err != nil {
			js.Throw(vm, err)
		}
		b, err := js.Unwrap(call.Argument(1))
		if err != nil {
			js.Throw(vm, err)
		}
		var msg string
		if !sobek.IsUndefined(call.Argument(2)) {
			msg = call.Argument(2).String()
		}
		if !assert.Regexp(t, b, a, msg) {
			js.Throw(vm, errors.New("not match"))
		}
		return
	})
	_ = obj.Set("true", func(call sobek.FunctionCall, vm *sobek.Runtime) (ret sobek.Value) {
		var msg string
		if !sobek.IsUndefined(call.Argument(1)) {
			msg = call.Argument(1).String()
		}
		if !assert.True(t, call.Argument(0).ToBoolean(), msg) {
			js.Throw(vm, errors.New("should be true"))
		}
		return
	})

	_ = vm.Runtime().Set("assert", obj)
	return VM{vm}
}

// PromiseResult get the promise result
func PromiseResult(value sobek.Value) sobek.Value {
	promise, ok := value.Export().(*sobek.Promise)
	if !ok {
		return value
	}
	switch promise.State() {
	case sobek.PromiseStateRejected:
		return promise.Result()
	case sobek.PromiseStateFulfilled:
		return promise.Result()
	default:
		panic("unexpected promise state")
	}
}
