// Package modulestest the module test vm
package modulestest

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/js"
	"github.com/stretchr/testify/assert"
)

// New returns a test VM instance
func New(t *testing.T) js.VM {
	vm := js.NewVM()
	runtime := vm.Runtime()

	assertObject := runtime.NewObject()
	_ = assertObject.Set("equal", func(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
		a, err := js.Unwrap(call.Argument(0))
		if err != nil {
			js.Throw(vm, err)
		}
		b, err := js.Unwrap(call.Argument(1))
		if err != nil {
			js.Throw(vm, err)
		}
		return vm.ToValue(assert.Equal(t, b, a, call.Argument(2).String()))
	})
	_ = assertObject.Set("true", func(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
		return vm.ToValue(assert.True(t, call.Argument(0).ToBoolean(), call.Argument(1).String()))
	})

	_ = runtime.Set("assert", assertObject)

	return vm
}
