package js

import (
	"errors"
	"sync/atomic"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/modules"
)

type wrap struct{ modules.Loader }

var loader atomic.Value

func init() {
	SetLoader(modules.NewLoader())
}

// Loader get Loader
func Loader() modules.Loader { return loader.Load().(wrap).Loader }

// SetLoader set the Loader
func SetLoader(ml modules.Loader) { loader.Store(wrap{ml}) }

// ModuleInstance return the sobek.ModuleInstance.
func ModuleInstance(rt *sobek.Runtime, module sobek.CyclicModuleRecord) (sobek.ModuleInstance, error) {
	instance := rt.GetModuleInstance(module)
	if instance == nil {
		if err := module.Link(); err != nil {
			return nil, err
		}
		promise := rt.CyclicModuleRecordEvaluate(module, Loader().ResolveModule)
		switch promise.State() {
		case sobek.PromiseStateRejected:
			return nil, promise.Result().Export().(error)
		case sobek.PromiseStateFulfilled:
		default:
		}
		return rt.GetModuleInstance(module), nil
	}
	return instance, nil
}

// ModuleCallable return the sobek.CyclicModuleRecord default export as sobek.Callable.
func ModuleCallable(rt *sobek.Runtime, module sobek.CyclicModuleRecord) (sobek.Callable, error) {
	instance, err := ModuleInstance(rt, module)
	if err != nil {
		return nil, err
	}
	value := instance.GetBindingValue("default")
	call, ok := sobek.AssertFunction(value)
	if !ok {
		return nil, errors.New("module default export is not a function")
	}
	return call, nil
}
