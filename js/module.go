package js

import (
	"context"
	"maps"
	"sync"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski"
)

// Module is what a module needs to return
type Module interface {
	Instantiate(*sobek.Runtime) (sobek.Value, error)
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
	prg           *sobek.Program
	exportedNames []string
	o             sync.Once
}

func (cm *cjsModule) Link() error { return nil }

func (cm *cjsModule) InitializeEnvironment() error { return nil }

func (cm *cjsModule) Instantiate(_ *sobek.Runtime) (sobek.CyclicModuleInstance, error) {
	return &cjsModuleInstance{m: cm}, nil
}

func (cm *cjsModule) RequestedModules() []string { return nil }

func (cm *cjsModule) Evaluate(_ *sobek.Runtime) *sobek.Promise { return nil }

func (cm *cjsModule) GetExportedNames(callback func([]string), _ ...sobek.ModuleRecord) bool {
	callback(cm.exportedNames)
	return true
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

func (cmi *cjsModuleInstance) ExecuteModule(rt *sobek.Runtime, _, _ func(any)) (sobek.CyclicModuleInstance, error) {
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

func (gm *goModule) Instantiate(rt *sobek.Runtime) (sobek.CyclicModuleInstance, error) {
	instance, err := gm.mod.Instantiate(rt)
	if err != nil {
		return nil, err
	}
	exports := instance.ToObject(rt)
	gm.once.Do(func() { gm.exportedNames = exports.Keys() })
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

func (gmi *goModuleInstance) ExecuteModule(_ *sobek.Runtime, _, _ func(any)) (sobek.CyclicModuleInstance, error) {
	return gmi, nil
}

const _js_executor_prefix = "executor/"

type _js_executor map[string]ski.NewExecutor

func (m _js_executor) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	var object *sobek.Object
	main, ok := m[""]
	if ok {
		object = rt.ToValue(toJSExec(main)).ToObject(rt)
	} else {
		object = rt.NewObject()
	}
	proto := object.Prototype()
	for k, v := range m {
		if k == "" {
			continue
		}
		_ = proto.Set(k, toJSExec(v))
	}
	return object, nil
}

type _js_exec struct {
	e ski.Executor
}

func (e _js_exec) Exec(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	v, err := e.e.Exec(Context(rt), call.Argument(0).Export())
	if err != nil {
		return sobek.Null()
	}
	return rt.ToValue(v)
}

func toJSExec(init ski.NewExecutor) func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
		args := make([]ski.Executor, 0, len(call.Arguments))
		for _, arg := range call.Arguments {
			args = append(args, ski.Raw(arg.Export()))
		}
		exec, err := init(args)
		if err != nil {
			Throw(rt, err)
		}
		return rt.ToValue(_js_exec{exec})
	}
}

// Executor the ski.Executor
type Executor struct{ sobek.CyclicModuleRecord }

func js(arg ski.Arguments) (ski.Executor, error) {
	if len(arg) == 0 {
		return nil, ErrInvalidModule
	}
	module, err := GetScheduler().Loader().CompileModule("", arg.GetString(0))
	if err != nil {
		return nil, err
	}
	return Executor{module}, nil
}

func (p Executor) Exec(ctx context.Context, arg any) (any, error) {
	value, err := RunModule(ctx, p, arg)
	if err != nil {
		return nil, err
	}

	unwrap, err := Unwrap(value)
	if err != nil {
		return nil, err
	}
	return unwrap, nil
}
