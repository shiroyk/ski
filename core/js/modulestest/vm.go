// Package modulestest the module test vm
package modulestest

import (
	"context"
	"testing"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/core/js"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"
)

// VM test vm
type VM struct {
	runtime *goja.Runtime
}

// RunString the js string
func (vm *VM) RunString(ctx context.Context, script string) (goja.Value, error) {
	_ = vm.runtime.Set(js.VMContextKey, ctx)
	return vm.runtime.RunString(script)
}

// Run the js program
func (vm *VM) Run(ctx context.Context, program js.Program) (goja.Value, error) {
	_ = vm.runtime.Set(js.VMContextKey, ctx)
	return vm.runtime.RunString(program.Code)
}

// Runtime returns the runtime
func (vm *VM) Runtime() *goja.Runtime {
	return vm.runtime
}

// New returns a test VM instance
func New(t *testing.T) *VM {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.UncapFieldNameMapper())
	js.EnableRequire(vm)
	js.InitGlobalModule(vm)

	assertObject := vm.NewObject()
	_ = assertObject.Set("equal", func(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
		a, err := js.Unwrap(call.Argument(0))
		if err != nil {
			js.Throw(vm, err)
		}
		b, err := js.Unwrap(call.Argument(1))
		if err != nil {
			js.Throw(vm, err)
		}
		return vm.ToValue(assert.Equal(t, a, b, call.Argument(2).String()))
	})
	_ = assertObject.Set("true", func(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
		return vm.ToValue(assert.True(t, call.Argument(0).ToBoolean(), call.Argument(2).String()))
	})

	consoleObject := vm.NewObject()
	_ = consoleObject.Set("log", func(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
		slog.Info(js.Format(call, vm).String())
		return
	})

	_ = vm.Set("console", consoleObject)
	_ = vm.Set("assert", assertObject)

	return &VM{vm}
}
