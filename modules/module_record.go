package modules

import (
	"sync"

	"github.com/grafana/sobek"
)

type cjsModule struct {
	prg           *sobek.Program
	exportedNames []string
	callback      []func([]string)
}

func (cm *cjsModule) Link() error { return nil }

func (cm *cjsModule) InitializeEnvironment() error { return nil }

func (cm *cjsModule) Instantiate(_ *sobek.Runtime) (sobek.CyclicModuleInstance, error) {
	return &cjsModuleInstance{m: cm}, nil
}

func (cm *cjsModule) RequestedModules() []string { return nil }

func (cm *cjsModule) Evaluate(_ *sobek.Runtime) *sobek.Promise { return nil }

func (cm *cjsModule) GetExportedNames(callback func([]string), _ ...sobek.ModuleRecord) bool {
	if cm.exportedNames != nil {
		callback(cm.exportedNames)
		return true
	}
	cm.callback = append(cm.callback, callback)
	return false
}

func (cm *cjsModule) ResolveExport(exportName string, _ ...sobek.ResolveSetElement) (*sobek.ResolvedBinding, bool) {
	return &sobek.ResolvedBinding{
		Module:      cm,
		BindingName: exportName,
	}, false
}

type cjsModuleInstance struct {
	m       *cjsModule
	exports *sobek.Object
}

func (cmi *cjsModuleInstance) HasTLA() bool { return false }

func (cmi *cjsModuleInstance) GetBindingValue(name string) sobek.Value {
	if name == "default" {
		if d := cmi.exports.Get("default"); d != nil {
			return d
		}
		return cmi.exports
	}
	return cmi.exports.Get(name)
}

func (cmi *cjsModuleInstance) ExecuteModule(rt *sobek.Runtime, _, _ func(any) error) (sobek.CyclicModuleInstance, error) {
	f, err := rt.RunProgram(cmi.m.prg)
	if err != nil {
		return nil, err
	}

	jsModule := rt.NewObject()
	cmi.exports = rt.NewObject()
	_ = jsModule.Set("exports", cmi.exports)
	if call, ok := sobek.AssertFunction(f); ok {
		jsRequire := rt.Get("require")

		// Run the module source, with "cmi.exports" as "this",
		// "cmi.exports" as the "exports" variable, "jsRequire"
		// as the "require" variable and "jsModule" as the
		// "module" variable (Nodejs capable).
		_, err = call(cmi.exports, cmi.exports, jsRequire, jsModule)
		if err != nil {
			return nil, err
		}
	}

	exports := jsModule.Get("exports")
	if sobek.IsNull(exports) {
		return nil, ErrInvalidModule
	}
	cmi.exports = exports.ToObject(rt)
	if cmi.m.exportedNames == nil {
		cmi.m.exportedNames = cmi.exports.GetOwnPropertyNames()
		if cmi.m.exportedNames == nil {
			cmi.m.exportedNames = []string{}
		}
		for _, callback := range cmi.m.callback {
			callback(cmi.m.exportedNames)
		}
	}
	return cmi, nil
}

type goModule struct {
	mod           Module
	exportedNames []string
	once          sync.Once
}

func (gm *goModule) Link() error { return nil }

func (gm *goModule) RequestedModules() []string { return nil }

func (gm *goModule) InitializeEnvironment() error { return nil }

func (gm *goModule) Instantiate(rt *sobek.Runtime) (sobek.CyclicModuleInstance, error) {
	instance, err := gm.mod.Instantiate(rt)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, ErrInvalidModule
	}
	exports := instance.ToObject(rt)
	gm.once.Do(func() { gm.exportedNames = exports.GetOwnPropertyNames() })
	return &goModuleInstance{exports}, nil
}

func (gm *goModule) GetExportedNames(callback func([]string), _ ...sobek.ModuleRecord) bool {
	callback(gm.exportedNames)
	return true
}

func (gm *goModule) ResolveExport(exportName string, _ ...sobek.ResolveSetElement) (*sobek.ResolvedBinding, bool) {
	return &sobek.ResolvedBinding{
		Module:      gm,
		BindingName: exportName,
	}, false
}

func (gm *goModule) Evaluate(_ *sobek.Runtime) *sobek.Promise { return nil }

type goModuleInstance struct{ *sobek.Object }

func (gmi *goModuleInstance) GetBindingValue(name string) sobek.Value {
	if name == "default" {
		return gmi.Object
	}
	return gmi.Get(name)
}

func (gmi *goModuleInstance) HasTLA() bool { return false }

func (gmi *goModuleInstance) ExecuteModule(_ *sobek.Runtime, _, _ func(any) error) (sobek.CyclicModuleInstance, error) {
	return gmi, nil
}
