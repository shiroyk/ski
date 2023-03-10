// Package modulestest the module test vm
package modulestest

import (
	"context"
	"testing"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/js/common"
	"github.com/shiroyk/cloudcat/js/modules"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"
)

// VM test vm
type VM struct {
	vm *goja.Runtime
}

// RunString the js string
func (vm *VM) RunString(_ context.Context, script string) (goja.Value, error) {
	return vm.vm.RunString(script)
}

// Run the js program
func (vm *VM) Run(_ context.Context, program common.Program) (goja.Value, error) {
	return vm.vm.RunString(program.Code)
}

// Runtime returns the runtime
func (vm *VM) Runtime() *goja.Runtime {
	return vm.vm
}

// New returns a test VM instance
func New(t *testing.T) *VM {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.UncapFieldNameMapper())
	modules.EnableRequire(vm)
	modules.InitGlobalModule(vm)

	assertObject := vm.NewObject()
	_ = assertObject.Set("equal", func(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
		a, err := common.Unwrap(call.Argument(0))
		if err != nil {
			common.Throw(vm, err)
		}
		b, err := common.Unwrap(call.Argument(1))
		if err != nil {
			common.Throw(vm, err)
		}
		return vm.ToValue(assert.Equal(t, a, b, call.Argument(2).String()))
	})
	_ = assertObject.Set("true", func(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
		return vm.ToValue(assert.True(t, call.Argument(0).ToBoolean(), call.Argument(2).String()))
	})

	consoleObject := vm.NewObject()
	_ = consoleObject.Set("log", func(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
		slog.Info(common.Format(call, vm).String())
		return
	})

	_ = vm.Set("console", consoleObject)
	_ = vm.Set("assert", assertObject)

	return &VM{vm}
}
