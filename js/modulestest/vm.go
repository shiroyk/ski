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

func (vm *VM) RunString(ctx context.Context, source string) (ret sobek.Value, err error) {
	vm.Run(ctx, func() {
		ret, err = vm.Runtime().RunString(source)
	})
	return
}

func (vm *VM) RunModule(ctx context.Context, source string, args ...any) (ret sobek.Value, err error) {
	module, err := vm.Loader().CompileModule("", source)
	if err != nil {
		return
	}
	return vm.VM.RunModule(ctx, module, args...)
}

// New returns a test VM instance
func New(t *testing.T, opts ...js.Option) VM {
	vm := js.NewVM(append([]js.Option{js.WithModuleLoader(js.NewModuleLoader())}, opts...)...)
	assertObject := vm.Runtime().NewObject()
	_ = assertObject.Set("equal", func(call sobek.FunctionCall, vm *sobek.Runtime) (ret sobek.Value) {
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
	_ = assertObject.Set("true", func(call sobek.FunctionCall, vm *sobek.Runtime) (ret sobek.Value) {
		var msg string
		if !sobek.IsUndefined(call.Argument(1)) {
			msg = call.Argument(1).String()
		}
		if !assert.True(t, call.Argument(0).ToBoolean(), msg) {
			js.Throw(vm, errors.New("should be true"))
		}
		return
	})

	_ = vm.Runtime().Set("assert", assertObject)
	return VM{vm}
}
