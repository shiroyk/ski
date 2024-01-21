package js

import (
	"context"
	"errors"
	"maps"
	"sync"

	"github.com/dop251/goja"
	"github.com/shiroyk/ski"
)

// Module is what a module needs to return
type Module interface {
	Instantiate(*goja.Runtime) (goja.Value, error)
}

// Global implements the interface will load into global when the VM initialize (InitGlobalModule).
type Global interface {
	Module
	Global() // is it a global module
}

// Register the given mod as an external JavaScript module that can be imported
// by name.
func Register(name string, mod Module) {
	if _, ok := mod.(Global); !ok {
		name = modulePrefix + name
	}
	registry.Lock()
	registry.native[name] = mod
	registry.Unlock()
}

// GetModule get the module
func GetModule(name string) (Module, bool) {
	registry.RLock()
	defer registry.RUnlock()
	module, ok := registry.native[name]
	return module, ok
}

func RemoveModule(name string) {
	registry.Lock()
	delete(registry.native, name)
	registry.Unlock()
}

// AllModule get all module
func AllModule() map[string]Module {
	registry.RLock()
	defer registry.RUnlock()
	return maps.Clone(registry.native)
}

const modulePrefix = "ski/"

var registry = struct {
	sync.RWMutex
	native map[string]Module
}{
	native: make(map[string]Module),
}

type cjsModule struct {
	prg           *goja.Program
	exportedNames []string
	o             sync.Once
}

func (cm *cjsModule) Link() error { return nil }

func (cm *cjsModule) InitializeEnvironment() error { return nil }

func (cm *cjsModule) Instantiate(_ *goja.Runtime) (goja.CyclicModuleInstance, error) {
	return &cjsModuleInstance{m: cm}, nil
}

func (cm *cjsModule) RequestedModules() []string { return nil }

func (cm *cjsModule) Evaluate(_ *goja.Runtime) *goja.Promise { return nil }

func (cm *cjsModule) GetExportedNames(_ ...goja.ModuleRecord) []string { return cm.exportedNames }

func (cm *cjsModule) ResolveExport(exportName string, _ ...goja.ResolveSetElement) (*goja.ResolvedBinding, bool) {
	return &goja.ResolvedBinding{
		Module:      cm,
		BindingName: exportName,
	}, false
}

type cjsModuleInstance struct {
	m       *cjsModule
	exports *goja.Object
}

func (cmi *cjsModuleInstance) HasTLA() bool { return false }

func (cmi *cjsModuleInstance) GetBindingValue(name string) goja.Value {
	if name == "default" {
		if d := cmi.exports.Get("default"); d != nil {
			return d
		}
		return cmi.exports
	}
	return cmi.exports.Get(name)
}

func (cmi *cjsModuleInstance) ExecuteModule(rt *goja.Runtime, _, _ func(any)) (goja.CyclicModuleInstance, error) {
	f, err := rt.RunProgram(cmi.m.prg)
	if err != nil {
		return nil, err
	}

	jsModule := rt.NewObject()
	cmi.exports = rt.NewObject()
	_ = jsModule.Set("exports", cmi.exports)
	if call, ok := goja.AssertFunction(f); ok {
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
	if goja.IsNull(exports) {
		return nil, ErrInvalidModule
	}
	cmi.exports = exports.ToObject(rt)
	cmi.m.o.Do(func() {
		cmi.m.exportedNames = cmi.exports.Keys()
	})
	return cmi, nil
}

type goModule struct {
	mod           Module
	once          sync.Once
	exportedNames []string
}

func (gm *goModule) Link() error { return nil }

func (gm *goModule) RequestedModules() []string { return nil }

func (gm *goModule) InitializeEnvironment() error { return nil }

func (gm *goModule) Instantiate(rt *goja.Runtime) (goja.CyclicModuleInstance, error) {
	instance, err := gm.mod.Instantiate(rt)
	if err != nil {
		return nil, err
	}
	exports := instance.ToObject(rt)
	gm.once.Do(func() { gm.exportedNames = exports.Keys() })
	return &goModuleInstance{exports}, nil
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

func (gm *goModule) Evaluate(_ *goja.Runtime) *goja.Promise { return nil }

type goModuleInstance struct{ *goja.Object }

func (gmi *goModuleInstance) GetBindingValue(name string) goja.Value {
	if gmi.Object == nil {
		return nil
	}
	if name == "default" {
		if v := gmi.Get("default"); v != nil {
			return v
		}
		return gmi.Object
	}
	return gmi.Get(name)
}

func (gmi *goModuleInstance) HasTLA() bool { return false }

func (gmi *goModuleInstance) ExecuteModule(_ *goja.Runtime, _, _ func(any)) (goja.CyclicModuleInstance, error) {
	return gmi, nil
}

const parserPrefix = "parser/"

type jsParser struct{ ski.Parser }

type exec struct {
	e ski.Executor
}

func (e exec) Exec(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	v, err := e.e.Exec(Context(rt), call.Argument(0).Export())
	if err != nil {
		return goja.Null()
	}
	return rt.ToValue(v)
}

func (m *jsParser) Instantiate(rt *goja.Runtime) (goja.Value, error) {
	object := rt.ToValue(m.Value).ToObject(rt)
	_ = object.SetPrototype(rt.ToValue(map[string]func(call goja.FunctionCall, rt *goja.Runtime) goja.Value{
		"value":    m.Value,
		"element":  m.Element,
		"elements": m.Elements,
	}).ToObject(rt))
	return object, nil
}

func (m *jsParser) Value(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	executor, err := m.Parser.Value(call.Argument(0).String())
	if err != nil {
		Throw(rt, err)
	}
	return rt.ToValue(exec{executor})
}

func (m *jsParser) Element(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	p, ok := m.Parser.(ski.ElementParser)
	if !ok {
		return goja.Null()
	}
	executor, err := p.Element(call.Argument(0).String())
	if err != nil {
		Throw(rt, err)
	}
	return rt.ToValue(exec{executor})
}

func (m *jsParser) Elements(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	p, ok := m.Parser.(ski.ElementParser)
	if !ok {
		return goja.Null()
	}
	executor, err := p.Elements(call.Argument(0).String())
	if err != nil {
		Throw(rt, err)
	}
	return rt.ToValue(exec{executor})
}

// Parser the esm parser of ski.Parser
type Parser struct{ ModuleLoader }

func (p Parser) Value(arg string) (ski.Executor, error) {
	if p.ModuleLoader == nil {
		return nil, errors.New("ModuleLoader can not be nil")
	}
	module, err := p.CompileModule("", arg)
	if err != nil {
		return nil, err
	}
	return _mod{module}, nil
}

// ModExec return a ski.Executor
func ModExec(cm goja.CyclicModuleRecord) ski.Executor { return _mod{cm} }

type _mod struct{ goja.CyclicModuleRecord }

func (m _mod) Exec(ctx context.Context, arg any) (any, error) {
	value, err := RunModule(ski.WithValue(ctx, "content", arg), m)
	if err != nil {
		return nil, err
	}

	return Unwrap(value)
}
