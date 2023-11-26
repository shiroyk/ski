package loader

import (
	"errors"
	"sync"

	"github.com/dop251/goja"
)

type cjsModule struct {
	prg           *goja.Program
	exportedNames []string
	o             sync.Once
}

func (cm *cjsModule) Link() error { return nil }

func (cm *cjsModule) InitializeEnvironment() error { return nil }

func (cm *cjsModule) Instantiate(rt *goja.Runtime) (goja.CyclicModuleInstance, error) {
	return &cjsModuleInstance{rt: rt, m: cm}, nil
}

func (cm *cjsModule) RequestedModules() []string { return nil }

func (cm *cjsModule) Evaluate(_ *goja.Runtime) *goja.Promise {
	panic("this shouldn't be called in the current implementation")
}

func (cm *cjsModule) GetExportedNames(_ ...goja.ModuleRecord) []string {
	cm.o.Do(func() {
		panic("somehow we first got to GetExportedNames of a commonjs module before they were set" +
			"- this should never happen and is some kind of a bug")
	})
	return cm.exportedNames
}

func (cm *cjsModule) ResolveExport(exportName string, _ ...goja.ResolveSetElement) (*goja.ResolvedBinding, bool) {
	return &goja.ResolvedBinding{
		Module:      cm,
		BindingName: exportName,
	}, false
}

type cjsModuleInstance struct {
	rt               *goja.Runtime
	m                *cjsModule
	exports          *goja.Object
	isEsModuleMarked bool
}

func (cmi *cjsModuleInstance) HasTLA() bool { return false }

func (cmi *cjsModuleInstance) GetBindingValue(name string) goja.Value {
	if name == "default" {
		d := cmi.exports.Get("default")
		if d != nil {
			return d
		}
		return cmi.exports
	}
	return cmi.exports.Get(name)
}

func (cmi *cjsModuleInstance) ExecuteModule(rt *goja.Runtime, _, _ func(any)) (goja.CyclicModuleInstance, error) {
	v, err := rt.RunProgram(cmi.m.prg)
	if err != nil {
		return nil, err
	}

	module := rt.NewObject()
	cmi.exports = rt.NewObject()
	_ = module.Set("exports", cmi.exports)
	jsRequire := rt.Get("require")
	call, ok := goja.AssertFunction(v)
	if !ok {
		return nil, errors.New("somehow a commonjs module is not wrapped in a function")
	}
	if _, err = call(cmi.exports, cmi.exports, jsRequire, module); err != nil {
		return nil, err
	}
	exportsV := module.Get("exports")
	if goja.IsNull(exportsV) {
		return nil, errors.New("exports must be an object") // TODO make this message more specific for commonjs
	}
	cmi.exports = exportsV.ToObject(rt)

	cmi.m.o.Do(func() {
		cmi.m.exportedNames = cmi.exports.Keys()
	})
	__esModule := cmi.exports.Get("__esModule") //nolint:revive,stylecheck
	cmi.isEsModuleMarked = __esModule != nil && __esModule.ToBoolean()
	return cmi, nil
}
