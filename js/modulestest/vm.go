// Package modulestest the module test vm
package modulestest

import (
	"context"
	"errors"
	"testing"

	"github.com/dop251/goja"
	"github.com/shiroyk/ski/js"
	"github.com/stretchr/testify/assert"
)

type VM struct{ js.VM }

func (vm *VM) RunString(ctx context.Context, source string) (ret goja.Value, err error) {
	vm.Run(ctx, func() {
		ret, err = vm.Runtime().RunString(source)
	})
	return
}

func (vm *VM) RunModule(ctx context.Context, source string) (ret goja.Value, err error) {
	module, err := vm.Loader().CompileModule("", source)
	if err != nil {
		return
	}
	return vm.VM.RunModule(ctx, module)
}

// New returns a test VM instance
func New(t *testing.T, opts ...js.Option) VM {
	vm := js.NewVM(append([]js.Option{js.WithModuleLoader(js.NewModuleLoader())}, opts...)...)
	assertObject := vm.Runtime().NewObject()
	_ = assertObject.Set("equal", func(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
		a, err := js.Unwrap(call.Argument(0))
		if err != nil {
			js.Throw(vm, err)
		}
		b, err := js.Unwrap(call.Argument(1))
		if err != nil {
			js.Throw(vm, err)
		}
		var msg string
		if !goja.IsUndefined(call.Argument(2)) {
			msg = call.Argument(2).String()
		}
		if !assert.Equal(t, b, a, msg) {
			js.Throw(vm, errors.New("not equal"))
		}
		return
	})
	_ = assertObject.Set("true", func(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
		var msg string
		if !goja.IsUndefined(call.Argument(1)) {
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
