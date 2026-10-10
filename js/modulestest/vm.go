// Package modulestest the module test vm
package modulestest

import (
	"context"
	"errors"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/modules"
)

type VM struct {
	modules.Loader
	js.VM
}

func (vm *VM) RunModule(ctx context.Context, source string, args ...any) (ret sobek.Value, err error) {
	module, err := vm.CompileModule("", source)
	if err != nil {
		return nil, err
	}
	err = vm.Run(ctx, func() error {
		instance := vm.Runtime().GetModuleInstance(module)
		if instance == nil {
			if err = module.Link(); err != nil {
				return err
			}
			promise := vm.Runtime().CyclicModuleRecordEvaluate(module, vm.ResolveModule)
			switch promise.State() {
			case sobek.PromiseStateRejected:
				return errors.New(promise.Result().String())
			case sobek.PromiseStateFulfilled:
			default:
			}
			instance = vm.Runtime().GetModuleInstance(module)
		}
		call, ok := sobek.AssertFunction(instance.GetBindingValue("default"))
		if !ok {
			ret = sobek.Undefined()
			return nil
		}

		values := make([]sobek.Value, len(args))
		for i, arg := range args {
			values[i] = vm.Runtime().ToValue(arg)
		}

		ret, err = call(sobek.Undefined(), values...)
		return err
	})
	return
}

// New returns a test VM instance
func New(opts ...js.Option) VM {
	vm := js.NewVM(opts...)
	loader := modules.NewLoader()
	rt := vm.Runtime()
	loader.EnableRequire(rt).EnableImportModuleDynamically(rt).InitGlobal(rt)
	_ = rt.Set("assert", Assert(rt))
	return VM{loader, vm}
}

// PromiseResult get the promise resolve result
// panic when promise reject.
func PromiseResult(value sobek.Value) sobek.Value {
	promise, ok := value.Export().(*sobek.Promise)
	if !ok {
		return value
	}
	switch promise.State() {
	case sobek.PromiseStateRejected:
		panic(promise.Result().String())
	case sobek.PromiseStateFulfilled:
		return promise.Result()
	default:
		panic("unexpected promise state")
	}
}
