package modules

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
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
		// EnableImportMeta sobek runtime SetFinalImportMeta
		EnableImportMeta(*sobek.Runtime) Loader
		// InitGlobal instantiates global objects for the runtime. It creates a proxy around the global object
		// to lazily load global modules when they are first accessed.
		//
		// This allows for automatic loading of global modules like fetch, TextEncoder, etc.
		// when they are referenced in code.
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
	ml := &loader{
		reverse:      make(map[sobek.ModuleRecord]*url.URL),
		goModules:    make(map[string]*goModule),
		cacheModules: make(map[string]moduleCache),
	}

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
		mu           sync.Mutex
		reverse      map[sobek.ModuleRecord]*url.URL
		goModules    map[string]*goModule
		cacheModules map[string]moduleCache

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

var typeProxy = reflect.TypeFor[sobek.Proxy]()

// InitGlobal instantiates global objects for the runtime. It creates a proxy around the global object
// to lazily load global modules when they are first accessed.
//
// This allows for automatic loading of global modules like fetch, TextEncoder, etc.
// when they are referenced in code.
func (ml *loader) InitGlobal(rt *sobek.Runtime) Loader {
	globals := make(map[string]string)
	for namespace, m := range All() {
		if global, ok := m.(Global); ok {
			for name := range global {
				globals[name] = namespace
			}
		}
	}

	// Create proxy to handle global property access
	handler := &sobek.ProxyTrapConfig{}
	handler.Get = func(target *sobek.Object, property string, receiver sobek.Value) sobek.Value {
		value := target.Get(property)
		if value != nil {
			return value
		}

		namespace, ok := globals[property]
		if !ok {
			return nil
		}

		mod, ok := ml.resolveGlobal(namespace, property)
		if ok {
			instance, err := mod.Instantiate(rt)
			if err != nil {
				throwError(rt, fmt.Errorf("%w '%s'", err, property))
			}
			value = instance.(*goModuleInstance).Object
		} else {
			value = sobek.Undefined()
		}

		if err := target.Set(property, value); err != nil {
			panic(err)
		}
		return value
	}
	handler.Has = func(target *sobek.Object, property string) bool {
		if value := target.Get(property); value != nil {
			return true
		}

		_, ok := globals[property]
		if ok {
			return true
		}
		return false
	}
	handler.OwnKeys = func(target *sobek.Object) (object *sobek.Object) {
		keys := target.GetOwnPropertyNames()
		ret := make([]string, 0, len(keys)+len(globals))
		for _, name := range keys {
			if _, ok := globals[name]; !ok {
				ret = append(ret, name)
			}
		}
		for name := range globals {
			ret = append(ret, name)
		}
		return rt.ToValue(ret).(*sobek.Object)
	}
	globalThis := rt.GlobalObject()
	if globalThis.ExportType() == typeProxy {
		globalThis = globalThis.Export().(sobek.Proxy).Target()
	}
	rt.SetGlobalObject(rt.ToValue(rt.NewProxy(globalThis, handler)).(*sobek.Object))
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

// EnableImportMeta sobek runtime SetFinalImportMeta
func (ml *loader) EnableImportMeta(rt *sobek.Runtime) Loader {
	rt.SetFinalImportMeta(func(object *sobek.Object, record sobek.ModuleRecord) {
		_ = object.Set("resolve", func(specifier string) (string, error) {
			u := ml.reversePath(record)
			c := *u
			if strings.HasPrefix(specifier, "/") {
				c.Path = specifier[1:]
			} else {
				c.Path = filepath.Join(u.Path, specifier)
			}
			return c.String(), nil
		})
		_ = object.DefineAccessorProperty("url", rt.ToValue(func(call sobek.FunctionCall) sobek.Value {
			return rt.ToValue(ml.reversePath(record).String())
		}), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
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
			throwError(rt, ErrInvalidModule)
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
	case *goModule:
		return instance.(*goModuleInstance).Object
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
	ml.mu.Lock()
	defer ml.mu.Unlock()
	if mod, ok := ml.goModules[specifier]; ok {
		return mod, ok
	}
	if module, ok := Get(specifier); ok {
		mod := &goModule{mod: module}
		ml.goModules[specifier] = mod
		return mod, ok
	}
	return nil, false
}

func (ml *loader) resolveGlobal(namespace, name string) (*goModule, bool) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	module, ok := ml.goModules[name]
	if !ok {
		mod, ok := Get(namespace)
		if !ok {
			return nil, false
		}
		globals, ok := mod.(Global)
		if !ok {
			return nil, false
		}
		mod, ok = globals[name]
		if !ok {
			return nil, false
		}
		module = &goModule{mod: mod}
		ml.goModules[name] = module
	}
	return module, true
}

func (ml *loader) reversePath(referencingScriptOrModule any) *url.URL {
	mod, ok := referencingScriptOrModule.(sobek.ModuleRecord)
	if !ok {
		return ml.base
	}

	ml.mu.Lock()
	u, ok := ml.reverse[mod]
	ml.mu.Unlock()
	if !ok {
		return ml.base
	}

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
	base = new(*base)
	for {
		if filepath.Base(start) != "node_modules" {
			base.Path = filepath.Join(start, "node_modules")
		} else {
			base.Path = start
		}

		mod, err = ml.loadAsFileOrDirectory(base, specifier)
		if mod != nil || isSyntaxError(err) {
			return mod, err
		}

		if start == ".." { // Dir('..') is '.'
			break
		}
		parent := filepath.Dir(start)
		if parent == start {
			break
		}
		start = parent
	}

	return nil, fmt.Errorf("%w '%s'", ErrNotFoundModule, specifier)
}

func (ml *loader) loadModule(base *url.URL, specifier string) (sobek.ModuleRecord, error) {
	var (
		absolute *url.URL
		query    string
	)

	if path, raw, ok := strings.Cut(specifier, "?"); ok {
		specifier = path
		query = raw
	}

	if strings.HasPrefix(specifier, "/") {
		absolute = new(*base)
		absolute.Path = specifier[1:]
		absolute.RawQuery = query
	} else {
		absolute = base.JoinPath(specifier)
		absolute.RawQuery = query
	}
	filename := absolute.String()

	ml.mu.Lock()
	cache, exists := ml.cacheModules[filename]
	ml.mu.Unlock()
	if exists {
		return cache.mod, cache.err
	}

	buf, err := ml.fileLoader(absolute, specifier)
	if err != nil {
		return nil, err
	}

	mod, err := ml.CompileModule(filename, string(buf))
	ml.mu.Lock()
	if err == nil {
		ml.reverse[mod] = absolute.JoinPath("..")
	}
	ml.cacheModules[filename] = moduleCache{mod: mod, err: err}
	ml.mu.Unlock()
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
	ex, err := rt.New(rt.Get("Error"), rt.ToValue(err.Error()))
	if err != nil {
		panic(err)
	}
	panic(ex)
}

func isSyntaxError(err error) bool {
	_, ok := err.(*sobek.CompilerSyntaxError)
	return ok
}
