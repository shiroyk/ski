// Package modulestest the module test vm
package modulestest

import (
	"errors"
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

	_ = runtime.Set("assert", assertObject)

	return vm
}
