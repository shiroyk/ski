package loader

import (
	"sync"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/plugin/jsmodule"
)

type goModule struct {
	mod           jsmodule.Module
	once          sync.Once
	exportedNames []string
}

func (gm *goModule) Link() error { return nil }

func (gm *goModule) RequestedModules() []string { return nil }

func (gm *goModule) InitializeEnvironment() error { return nil }

func (gm *goModule) Instantiate(rt *goja.Runtime) (goja.CyclicModuleInstance, error) {
	object := rt.ToValue(gm.mod.Exports()).ToObject(rt)
	gm.once.Do(func() {
		gm.exportedNames = object.Keys()
	})
	return &goModuleInstance{object}, nil
}

func (gm *goModule) GetExportedNames(_ ...goja.ModuleRecord) []string {
	return gm.exportedNames
}

func (gm *goModule) ResolveExport(exportName string, _ ...goja.ResolveSetElement) (*goja.ResolvedBinding, bool) {
	return &goja.ResolvedBinding{
		Module:      gm,
		BindingName: exportName,
	}, false
}

func (gm *goModule) Evaluate(_ *goja.Runtime) *goja.Promise { panic("this shouldn't happen") }

type goModuleInstance struct{ export *goja.Object }

func (gmi *goModuleInstance) GetBindingValue(name string) goja.Value {
	if name == "default" {
		return gmi.export
	}
	if gmi.export == nil {
		return nil
	}
	return gmi.export.Get(name)
}

func (gmi *goModuleInstance) HasTLA() bool { return false }

func (gmi *goModuleInstance) ExecuteModule(_ *goja.Runtime, _, _ func(any)) (goja.CyclicModuleInstance, error) {
	return gmi, nil
}
