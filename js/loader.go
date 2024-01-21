package js

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/dop251/goja"
	"github.com/dop251/goja/parser"
	"github.com/shiroyk/ski"
)

var (
	// ErrInvalidModule module is invalid
	ErrInvalidModule = errors.New("invalid module")
	// ErrIllegalModuleName module name is illegal
	ErrIllegalModuleName = errors.New("illegal module name")
	// ErrNotFoundModule not found module
	ErrNotFoundModule = errors.New("not found module")
)

type (
	// ModuleLoader the js module loader.
	ModuleLoader interface {
		// CompileModule compile module from source string (cjs/esm).
		CompileModule(name, source string) (goja.CyclicModuleRecord, error)
		// ResolveModule resolve the module returns the goja.ModuleRecord.
		ResolveModule(any, string) (goja.ModuleRecord, error)
		// EnableRequire enable the global function require to the goja.Runtime.
		EnableRequire(*goja.Runtime) ModuleLoader
		// EnableImportModuleDynamically goja runtime SetImportModuleDynamically
		EnableImportModuleDynamically(*goja.Runtime) ModuleLoader
	}

	// LoaderOption the default moduleLoader options.
	LoaderOption func(*moduleLoader)

	// FileLoader is a type alias for a function that returns the contents of the referenced file.
	FileLoader func(specifier *url.URL, name string) ([]byte, error)

	// emptyLoader
	emptyLoader struct{}
)

// WithBaseLoader the base directory of module loader.
func WithBaseLoader(base *url.URL) LoaderOption {
	return func(o *moduleLoader) { o.base = base }
}

// WithFileLoader the file loader of module loader.
func WithFileLoader(loader FileLoader) LoaderOption {
	return func(o *moduleLoader) { o.fileLoader = loader }
}

// WithSourceMapLoader the source map loader of module loader.
func WithSourceMapLoader(loader func(path string) ([]byte, error)) LoaderOption {
	return func(o *moduleLoader) { o.sourceLoader = parser.WithSourceMapLoader(loader) }
}

// NewModuleLoader returns a new module resolver
// if the fileLoader option not provided, uses the default DefaultFileLoader.
func NewModuleLoader(opts ...LoaderOption) ModuleLoader {
	ml := &moduleLoader{
		modules:   make(map[string]moduleCache),
		goModules: make(map[string]goja.CyclicModuleRecord),
		parsers:   make(map[string]goja.CyclicModuleRecord),
		reverse:   make(map[goja.ModuleRecord]*url.URL),
	}

	for _, option := range opts {
		option(ml)
	}

	if ml.base == nil {
		ml.base = &url.URL{Scheme: "file", Path: "."}
	}
	if ml.fileLoader == nil {
		ml.fileLoader = DefaultFileLoader(ski.NewFetch())
	}
	if ml.sourceLoader == nil {
		ml.sourceLoader = parser.WithDisableSourceMaps
	}
	return ml
}

// DefaultFileLoader the default file loader.
// Supports file and HTTP scheme loading.
func DefaultFileLoader(fetch ski.Fetch) FileLoader {
	return func(specifier *url.URL, name string) ([]byte, error) {
		switch specifier.Scheme {
		case "http", "https":
			req, err := http.NewRequest(http.MethodGet, specifier.String(), nil)
			if err != nil {
				return nil, err
			}
			res, err := fetch.Do(req)
			if err != nil {
				return nil, err
			}
			defer res.Body.Close()
			body, err := io.ReadAll(res.Body)
			return body, err
		case "file":
			return fs.ReadFile(os.DirFS("."), specifier.Path)
		default:
			return nil, fmt.Errorf("scheme not supported %s", specifier.Scheme)
		}
	}
}

type (
	// moduleLoader the ModuleLoader implement.
	// Allows loading and interop between ES module and CommonJS module.
	moduleLoader struct {
		sync.Mutex
		modules   map[string]moduleCache
		goModules map[string]goja.CyclicModuleRecord
		parsers   map[string]goja.CyclicModuleRecord
		reverse   map[goja.ModuleRecord]*url.URL

		fileLoader FileLoader

		base         *url.URL
		sourceLoader parser.Option
	}

	moduleCache struct {
		mod goja.CyclicModuleRecord
		err error
	}
)

// EnableRequire enable the global function require to the goja.Runtime.
func (ml *moduleLoader) EnableRequire(rt *goja.Runtime) ModuleLoader {
	_ = rt.Set("require", ml.require)
	return ml
}

// require resolve the module instance.
func (ml *moduleLoader) require(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	name := call.Argument(0).String()
	mod, err := ml.ResolveModule(ml.getCurrentModuleRecord(rt), name)
	if err != nil {
		Throw(rt, err)
	}
	if mod, ok := mod.(*goModule); ok {
		instance, err := mod.mod.Instantiate(rt)
		if err != nil {
			Throw(rt, err)
		}
		return instance
	}

	instance := rt.GetModuleInstance(mod)
	if instance == nil {
		if err = mod.Link(); err != nil {
			Throw(rt, err)
		}
		cm, ok := mod.(goja.CyclicModuleRecord)
		if !ok {
			Throw(rt, ErrInvalidModule)
		}
		promise := rt.CyclicModuleRecordEvaluate(cm, ml.ResolveModule)
		if promise.State() == goja.PromiseStateRejected {
			panic(promise.Result())
		}
		instance = rt.GetModuleInstance(mod)
	}

	switch mod.(type) {
	case *cjsModule:
		return instance.(*cjsModuleInstance).GetBindingValue("default")
	case *goja.SourceTextModuleRecord:
		if v := instance.GetBindingValue("default"); v != nil {
			return v
		}
	}

	return rt.NamespaceObjectFor(mod)
}

func (ml *moduleLoader) EnableImportModuleDynamically(rt *goja.Runtime) ModuleLoader {
	rt.SetImportModuleDynamically(func(referencingScriptOrModule any, specifier goja.Value, promiseCapability any) {
		NewPromise(rt,
			func() (goja.ModuleRecord, error) {
				return ml.ResolveModule(referencingScriptOrModule, specifier.String())
			},
			func(module goja.ModuleRecord, err error) (any, error) {
				rt.FinishLoadingImportModule(referencingScriptOrModule, specifier, promiseCapability, module, err)
				return nil, err
			})
	})
	return ml
}

func (ml *moduleLoader) getCurrentModuleRecord(rt *goja.Runtime) goja.ModuleRecord {
	var buf [2]goja.StackFrame
	frames := rt.CaptureCallStack(2, buf[:0])
	if len(frames) == 0 {
		return nil
	}
	mod, _ := ml.ResolveModule(nil, frames[1].SrcName())
	return mod
}

// ResolveModule resolve the module returns the goja.ModuleRecord.
func (ml *moduleLoader) ResolveModule(referencingScriptOrModule any, name string) (goja.ModuleRecord, error) {
	switch {
	case strings.HasPrefix(name, modulePrefix):
		ml.Lock()
		defer ml.Unlock()
		if mod, ok := ml.goModules[name]; ok {
			return mod, nil
		}
		if e, ok := GetModule(name); ok {
			mod := &goModule{mod: e}
			ml.goModules[name] = mod
			return mod, nil
		}
		return nil, ErrNotFoundModule
	case strings.HasPrefix(name, parserPrefix):
		ml.Lock()
		defer ml.Unlock()
		name = strings.TrimPrefix(name, parserPrefix)
		if mod, ok := ml.parsers[name]; ok {
			return mod, nil
		}
		if p, ok := ski.GetParser(name); ok {
			mod := &goModule{mod: &jsParser{p}}
			ml.parsers[name] = mod
			return mod, nil
		}
		return nil, ErrNotFoundModule
	default:
		return ml.resolve(ml.reversePath(referencingScriptOrModule), name)
	}
}

func (ml *moduleLoader) resolve(base *url.URL, specifier string) (goja.ModuleRecord, error) {
	if specifier == "" {
		return nil, ErrIllegalModuleName
	}

	if isBasePath(specifier) {
		return ml.loadAsFileOrDirectory(base, specifier)
	}

	if strings.Contains(specifier, "://") {
		uri, err := url.Parse(specifier)
		if err != nil {
			return nil, err
		}
		return ml.loadModule(uri, "")
	}

	mod, err := ml.loadNodeModules(specifier)
	if err != nil {
		return nil, fmt.Errorf("module %s not found with error %s", specifier, err)
	}
	return mod, nil
}

func (ml *moduleLoader) reversePath(referencingScriptOrModule any) *url.URL {
	if referencingScriptOrModule == nil {
		return ml.base
	}
	mod, ok := referencingScriptOrModule.(goja.ModuleRecord)
	if !ok {
		return ml.base
	}

	ml.Lock()
	p, ok := ml.reverse[mod]
	ml.Unlock()

	if !ok {
		return ml.base
	}

	if p.String() == "file://-" {
		return ml.base
	}
	return p
}

func (ml *moduleLoader) loadAsFileOrDirectory(modPath *url.URL, modName string) (goja.ModuleRecord, error) {
	mod, err := ml.loadAsFile(modPath, modName)
	if err != nil {
		return ml.loadAsDirectory(modPath.JoinPath(modName))
	}
	return mod, nil
}

func (ml *moduleLoader) loadAsFile(modPath *url.URL, modName string) (module goja.ModuleRecord, err error) {
	if module, err = ml.loadModule(modPath, modName); err == nil {
		return
	}
	if module, err = ml.loadModule(modPath, modName+".js"); err == nil {
		return
	}
	return ml.loadModule(modPath, modName+".json")
}

func (ml *moduleLoader) loadAsDirectory(modPath *url.URL) (module goja.ModuleRecord, err error) {
	buf, err := ml.fileLoader(modPath.JoinPath("package.json"), "package.json")
	if err != nil {
		return ml.loadModule(modPath, "index.js")
	}
	var pkg struct {
		Main string `json:"main"`
	}
	err = json.Unmarshal(buf, &pkg)
	if err != nil || len(pkg.Main) == 0 {
		return ml.loadModule(modPath, "index.js")
	}

	if module, err = ml.loadAsFile(modPath, pkg.Main); module != nil || err != nil {
		return
	}

	return ml.loadModule(modPath, "index.js")
}

func (ml *moduleLoader) loadNodeModules(modName string) (mod goja.ModuleRecord, err error) {
	start := ml.base.Path
	for {
		var p string
		if path.Base(start) != "node_modules" {
			p = path.Join(start, "node_modules")
		} else {
			p = start
		}
		if mod, err = ml.loadAsFileOrDirectory(ml.base.JoinPath(p), modName); mod != nil || err != nil {
			return
		}
		if start == ".." { // Dir('..') is '.'
			break
		}
		parent := path.Dir(start)
		if parent == start {
			break
		}
		start = parent
	}

	return nil, fmt.Errorf("not found module %s at %s", modName, ml.base)
}

func (ml *moduleLoader) loadModule(modPath *url.URL, modName string) (goja.ModuleRecord, error) {
	file := modPath.JoinPath(modName)
	specifier := file.String()

	ml.Lock()
	defer ml.Unlock()

	cache, exists := ml.modules[specifier]
	if exists {
		return cache.mod, cache.err
	}

	buf, err := ml.fileLoader(file, modName)
	if err != nil {
		return nil, err
	}
	mod, err := ml.CompileModule(specifier, string(buf))
	if err == nil {
		file.Path = filepath.Dir(file.Path)
		ml.reverse[mod] = file
	}
	ml.modules[specifier] = moduleCache{mod: mod, err: err}
	return mod, err
}

func (ml *moduleLoader) CompileModule(name, source string) (goja.CyclicModuleRecord, error) {
	if filepath.Ext(name) == ".json" {
		source = "module.exports = JSON.parse('" + template.JSEscapeString(source) + "')"
		return ml.compileCjsModule(name, source)
	}

	ast, err := goja.Parse(name, source, parser.IsModule, ml.sourceLoader)
	if err != nil {
		return nil, err
	}

	isModule := len(ast.ExportEntries) > 0 || len(ast.ImportEntries) > 0 || ast.HasTLA
	if !isModule {
		return ml.compileCjsModule(name, source)
	}

	return goja.ModuleFromAST(ast, ml.ResolveModule)
}

func (ml *moduleLoader) compileCjsModule(name, source string) (goja.CyclicModuleRecord, error) {
	source = "(function(exports, require, module) {" + source + "\n})"

	ast, err := goja.Parse(name, source, ml.sourceLoader)
	if err != nil {
		return nil, err
	}

	prg, err := goja.CompileAST(ast, false)
	if err != nil {
		return nil, err
	}

	return &cjsModule{prg: prg}, nil
}

func isBasePath(path string) bool {
	return strings.HasPrefix(path, "/") ||
		strings.HasPrefix(path, "./") ||
		strings.HasPrefix(path, "../") ||
		path == "." || path == ".."
}

var errNotSupport = errors.New("js.ModuleLoader not provided, require and module not working")

func (e emptyLoader) CompileModule(name string, source string) (goja.CyclicModuleRecord, error) {
	return goja.ParseModule(name, source, e.ResolveModule)
}
func (emptyLoader) ResolveModule(any, string) (goja.ModuleRecord, error) {
	return nil, errNotSupport
}
func (e emptyLoader) EnableRequire(rt *goja.Runtime) ModuleLoader {
	_ = rt.Set("require", func() {
		panic(rt.NewGoError(errNotSupport))
	})
	return e
}
func (e emptyLoader) EnableImportModuleDynamically(rt *goja.Runtime) ModuleLoader {
	rt.SetImportModuleDynamically(func(referencingScriptOrModule any, specifier goja.Value, promiseCapability any) {
		NewPromise(rt,
			func() (goja.ModuleRecord, error) {
				return nil, errNotSupport
			})
	})
	return e
}
