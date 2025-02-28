package modules

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"text/template"

	"github.com/grafana/sobek"
	"github.com/grafana/sobek/parser"
)

var (
	// ErrInvalidModule module is invalid
	ErrInvalidModule = errors.New("invalid module")
	// ErrIllegalModuleName module name is illegal
	ErrIllegalModuleName = errors.New("illegal module name")
	// ErrNotFoundModule not found module
	ErrNotFoundModule = errors.New("cannot found module")
)

type (
	// Loader the js module loader.
	Loader interface {
		// CompileModule compile module from source string (cjs/esm).
		CompileModule(name, source string) (sobek.CyclicModuleRecord, error)
		// ResolveModule resolve the module returns the sobek.ModuleRecord.
		ResolveModule(any, string) (sobek.ModuleRecord, error)
		// EnableRequire enable the global function require to the sobek.Runtime.
		EnableRequire(*sobek.Runtime) Loader
		// EnableImportModuleDynamically sobek runtime SetImportModuleDynamically
		EnableImportModuleDynamically(*sobek.Runtime) Loader
		// InitGlobal instantiate Global module and add their exports to the global scope of the JavaScript runtime.
		InitGlobal(*sobek.Runtime) Loader
		// SetFileLoader set the FileLoader.
		SetFileLoader(fl FileLoader)
	}

	// Option the new Loader options.
	Option func(*loader)

	// FileLoader is a type alias for a function that returns the contents of the referenced file.
	FileLoader func(specifier *url.URL, name string) ([]byte, error)
)

// WithBase the base directory of module loader.
func WithBase(base *url.URL) Option {
	return func(o *loader) { o.base = base }
}

// WithFileLoader the file loader of module loader.
func WithFileLoader(fl FileLoader) Option {
	return func(o *loader) { o.fileLoader = fl }
}

// WithSourceMapLoader the source map loader of module loader.
func WithSourceMapLoader(fn func(path string) ([]byte, error)) Option {
	return func(o *loader) { o.sourceLoader = parser.WithSourceMapLoader(fn) }
}

// NewLoader returns a new module resolver
// if the fileLoader option not provided, uses the default DefaultFileLoader.
func NewLoader(opts ...Option) Loader {
	ml := new(loader)

	for _, option := range opts {
		option(ml)
	}

	if ml.base == nil {
		ml.base = &url.URL{Scheme: "file", Path: "."}
	}
	if ml.fileLoader == nil {
		ml.fileLoader = DefaultFileLoader(http.DefaultClient.Do)
	}
	if ml.sourceLoader == nil {
		ml.sourceLoader = parser.WithDisableSourceMaps
	}
	return ml
}

// DefaultFileLoader the default file loader.
// Supports file and HTTP scheme loading.
func DefaultFileLoader(fetch func(*http.Request) (*http.Response, error)) FileLoader {
	return func(specifier *url.URL, name string) ([]byte, error) {
		switch specifier.Scheme {
		case "http", "https":
			req, err := http.NewRequest(http.MethodGet, specifier.String(), nil)
			if err != nil {
				return nil, err
			}
			res, err := fetch(req)
			if err != nil {
				return nil, err
			}
			defer res.Body.Close()
			return io.ReadAll(res.Body)
		case "file":
			return os.ReadFile(specifier.Path)
		default:
			return nil, fmt.Errorf("scheme not supported %s", specifier.Scheme)
		}
	}
}

type (
	// loader the Loader implement.
	// Allows loading and interop between ES module and CommonJS module.
	loader struct {
		reverse      sync.Map
		goModules    sync.Map
		cacheModules sync.Map

		globalOnce sync.Once
		globals    map[string]string

		fileLoader FileLoader

		base         *url.URL
		sourceLoader parser.Option
	}

	moduleCache struct {
		mod sobek.CyclicModuleRecord
		err error
	}
)

// SetFileLoader set the FileLoader.
func (ml *loader) SetFileLoader(fl FileLoader) {
	if fl == nil {
		return
	}
	ml.fileLoader = fl
}

// EnableRequire enable the global function require to the sobek.Runtime.
func (ml *loader) EnableRequire(rt *sobek.Runtime) Loader {
	_ = rt.Set("require", ml.require)
	return ml
}

// InitGlobal instantiates global objects for the runtime. It creates a proxy around the global object
// to lazily load global modules when they are first accessed.
//
// This allows for automatic loading of global modules like fetch, TextEncoder, etc.
// when they are referenced in code.
func (ml *loader) InitGlobal(rt *sobek.Runtime) Loader {
	ml.globalOnce.Do(func() {
		// Collect all global module names and their namespaces
		ml.globals = make(map[string]string)
		for namespace, m := range All() {
			if globals, ok := m.(Global); ok {
				for name := range globals {
					ml.globals[name] = namespace
				}
			}
		}
	})

	// Create proxy to handle global property access
	proxy := rt.NewProxy(rt.GlobalObject(), &sobek.ProxyTrapConfig{
		Get: func(target *sobek.Object, property string, receiver sobek.Value) sobek.Value {
			if value := target.Get(property); value != nil {
				return value
			}

			namespace, ok := ml.globals[property]
			if !ok {
				return sobek.Undefined()
			}

			module, ok := ml.goModules.Load(property)
			if !ok {
				mod, ok := Get(namespace)
				if !ok {
					return sobek.Undefined()
				}

				globals, ok := mod.(Global)
				if !ok {
					return sobek.Undefined()
				}

				mod = globals[property]
				if mod == nil {
					return sobek.Undefined()
				}

				module, _ = ml.goModules.LoadOrStore(property, &goModule{mod: mod})
			}

			// Instantiate the module
			record := module.(sobek.CyclicModuleRecord)
			promise := rt.CyclicModuleRecordEvaluate(record, ml.ResolveModule)

			switch promise.State() {
			case sobek.PromiseStateRejected:
				slog.Warn("failed to instantiate global module",
					"namespace", namespace,
					"module", property,
					"error", promise.Result().String())
				return sobek.Undefined()
			case sobek.PromiseStatePending:
				return sobek.Undefined()
			default:
				// Set the instantiated module on the global object
				value := rt.GetModuleInstance(record).(*goModuleInstance).Object
				if err := target.Set(property, value); err != nil {
					throwError(rt, err)
				}
				return value
			}
		},
	})

	rt.SetGlobalObject(rt.ToValue(proxy).(*sobek.Object))
	return ml
}

// EnableImportModuleDynamically sobek runtime SetImportModuleDynamically
func (ml *loader) EnableImportModuleDynamically(rt *sobek.Runtime) Loader {
	rt.SetImportModuleDynamically(func(scriptOrModule any, specifier sobek.Value, promiseCapability any) {
		module, err := ml.ResolveModule(scriptOrModule, specifier.String())
		rt.FinishLoadingImportModule(scriptOrModule, specifier, promiseCapability, module, err)
	})
	return ml
}

// require resolve the module instance.
func (ml *loader) require(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	name := call.Argument(0).String()
	mod, err := ml.ResolveModule(ml.getCurrentModuleRecord(rt), name)
	if err != nil {
		throwError(rt, err)
	}

	instance := rt.GetModuleInstance(mod)
	if instance == nil {
		if err = mod.Link(); err != nil {
			throwError(rt, err)
		}
		cm, ok := mod.(sobek.CyclicModuleRecord)
		if !ok {
			panic(rt.NewGoError(ErrInvalidModule))
		}
		promise := rt.CyclicModuleRecordEvaluate(cm, ml.ResolveModule)
		if promise.State() == sobek.PromiseStateRejected {
			throwError(rt, errors.New(promise.Result().String()))
		}
		instance = rt.GetModuleInstance(mod)
	}

	switch mod.(type) {
	case *cjsModule:
		return instance.(*cjsModuleInstance).exports
	default:
		return rt.NamespaceObjectFor(mod)
	}
}

func (ml *loader) getCurrentModuleRecord(rt *sobek.Runtime) sobek.ModuleRecord {
	var buf [2]sobek.StackFrame
	frames := rt.CaptureCallStack(2, buf[:0])
	if len(frames) == 0 {
		return nil
	}
	mod, _ := ml.ResolveModule(nil, frames[1].SrcName())
	return mod
}

// ResolveModule resolve the module returns the sobek.ModuleRecord.
func (ml *loader) ResolveModule(referencingScriptOrModule any, name string) (sobek.ModuleRecord, error) {
	switch {
	case strings.HasPrefix(name, prefix):
		if mod, ok := ml.resolveGo(name); ok {
			return mod, nil
		}
		return nil, fmt.Errorf("%w '%s'", ErrNotFoundModule, name)
	case strings.HasPrefix(name, nodePrefix):
		if mod, ok := ml.resolveGo(name); ok {
			return mod, nil
		}
		fallthrough
	default:
		return ml.resolve(ml.reversePath(referencingScriptOrModule), name)
	}
}

func (ml *loader) resolve(base *url.URL, specifier string) (sobek.ModuleRecord, error) {
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

	return ml.loadNodeModules(base, specifier)
}

func (ml *loader) resolveGo(specifier string) (sobek.ModuleRecord, bool) {
	if mod, ok := ml.goModules.Load(specifier); ok {
		return mod.(sobek.ModuleRecord), ok
	}
	if module, ok := Get(specifier); ok {
		mod := &goModule{mod: module}
		ml.goModules.Store(specifier, mod)
		return mod, ok
	}
	return nil, false
}

func (ml *loader) reversePath(referencingScriptOrModule any) *url.URL {
	mod, ok := referencingScriptOrModule.(sobek.ModuleRecord)
	if !ok {
		return ml.base
	}

	p, ok := ml.reverse.Load(mod)
	if !ok {
		return ml.base
	}

	u := p.(*url.URL)
	if u.Scheme == "file" && u.Path == "-" {
		return ml.base
	}
	return u
}

func (ml *loader) loadAsFileOrDirectory(base *url.URL, specifier string) (sobek.ModuleRecord, error) {
	mod, err := ml.loadAsFile(base, specifier)
	if err != nil {
		if isSyntaxError(err) {
			return nil, err
		}
		return ml.loadAsDirectory(base.JoinPath(specifier))
	}
	return mod, nil
}

func (ml *loader) loadAsFile(base *url.URL, specifier string) (module sobek.ModuleRecord, err error) {
	if module, err = ml.loadModule(base, specifier); err == nil {
		return
	}
	if isSyntaxError(err) {
		return nil, err
	}
	if module, err = ml.loadModule(base, specifier+".js"); err == nil {
		return
	}
	if isSyntaxError(err) {
		return nil, err
	}
	return ml.loadModule(base, specifier+".json")
}

func (ml *loader) loadAsDirectory(base *url.URL) (mod sobek.ModuleRecord, err error) {
	buf, err := ml.fileLoader(base.JoinPath("package.json"), "package.json")
	if err != nil {
		return ml.loadModule(base, "index.js")
	}

	var pkg struct {
		Main   string `json:"main"`
		Module string `json:"module"`
	}
	if err = json.Unmarshal(buf, &pkg); err != nil {
		return ml.loadModule(base, "index.js")
	}

	for _, entry := range []string{pkg.Module, pkg.Main} {
		if len(entry) > 0 {
			if mod, err = ml.loadAsFile(base, entry); err != nil {
				if isSyntaxError(err) {
					return nil, err
				}
				err = nil
			} else {
				return
			}
		}
	}

	return ml.loadModule(base, "index.js")
}

func (ml *loader) loadNodeModules(base *url.URL, specifier string) (mod sobek.ModuleRecord, err error) {
	start := base.Path
	u := *base
	nodeModules := &u
	nodeModules.Path = ""
	for {
		if path.Base(start) != "node_modules" {
			nodeModules.Path = filepath.Join(start, "node_modules")
		} else {
			nodeModules.Path = start
		}

		mod, err = ml.loadAsFileOrDirectory(nodeModules, specifier)
		if mod != nil || isSyntaxError(err) {
			return mod, err
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

	return nil, fmt.Errorf("%w '%s'", ErrNotFoundModule, specifier)
}

func (ml *loader) loadModule(base *url.URL, specifier string) (sobek.ModuleRecord, error) {
	var absolute *url.URL
	if strings.HasPrefix(specifier, "/") {
		u := *base
		u.Path = specifier
		absolute = &u
	} else {
		absolute = base.JoinPath(specifier)
	}
	filename := absolute.String()

	cache, exists := ml.cacheModules.Load(filename)
	if exists {
		m := cache.(moduleCache)
		return m.mod, m.err
	}

	buf, err := ml.fileLoader(absolute, specifier)
	if err != nil {
		return nil, err
	}

	mod, err := ml.CompileModule(filename, string(buf))
	if err == nil {
		ml.reverse.Store(mod, absolute.JoinPath(".."))
	}
	ml.cacheModules.Store(filename, moduleCache{mod: mod, err: err})
	return mod, err
}

func (ml *loader) CompileModule(name, source string) (sobek.CyclicModuleRecord, error) {
	if filepath.Ext(name) == ".json" {
		source = "module.exports = JSON.parse('" + template.JSEscapeString(source) + "')"
		return ml.compileCjsModule(name, source)
	}

	ast, err := sobek.Parse(name, source, parser.IsModule, ml.sourceLoader)
	if err != nil {
		return nil, err
	}

	isModule := len(ast.ExportEntries) > 0 || len(ast.ImportEntries) > 0 || ast.HasTLA
	if !isModule {
		return ml.compileCjsModule(name, source)
	}

	return sobek.ModuleFromAST(ast, ml.ResolveModule)
}

func (ml *loader) compileCjsModule(name, source string) (sobek.CyclicModuleRecord, error) {
	source = "(function(exports, require, module) {" + source + "\n})"

	ast, err := sobek.Parse(name, source, ml.sourceLoader)
	if err != nil {
		return nil, err
	}

	prg, err := sobek.CompileAST(ast, false)
	if err != nil {
		return nil, err
	}

	return &cjsModule{prg: prg}, nil
}

func isBasePath(path string) bool {
	result := path == "." || path == ".." ||
		strings.HasPrefix(path, "/") ||
		strings.HasPrefix(path, "./") ||
		strings.HasPrefix(path, "../")

	if runtime.GOOS == "windows" {
		result = result ||
			strings.HasPrefix(path, `.\`) ||
			strings.HasPrefix(path, `..\`) ||
			filepath.IsAbs(path)
	}

	return result
}

// throwError throw js error
func throwError(rt *sobek.Runtime, err error) sobek.Value {
	ctor, ok := sobek.AssertConstructor(rt.Get("Error"))
	if !ok {
		panic(rt.ToValue(err.Error()))
	}
	obj, err := ctor(nil, rt.ToValue(err.Error()))
	if err != nil {
		panic(rt.ToValue(err.Error()))
	}
	panic(obj)
}

func isSyntaxError(err error) bool {
	_, ok := err.(*sobek.CompilerSyntaxError)
	return ok
}
