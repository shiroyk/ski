package modules

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
	ErrNotFoundModule = errors.New("not found module")
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
		modules:   make(map[string]moduleCache),
		goModules: make(map[string]sobek.CyclicModuleRecord),
		reverse:   make(map[sobek.ModuleRecord]*url.URL),
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
			return fs.ReadFile(os.DirFS("."), specifier.Path)
		default:
			return nil, fmt.Errorf("scheme not supported %s", specifier.Scheme)
		}
	}
}

type (
	// loader the Loader implement.
	// Allows loading and interop between ES module and CommonJS module.
	loader struct {
		sync.Mutex
		modules   map[string]moduleCache
		goModules map[string]sobek.CyclicModuleRecord
		reverse   map[sobek.ModuleRecord]*url.URL

		fileLoader FileLoader

		base         *url.URL
		sourceLoader parser.Option
	}

	moduleCache struct {
		mod sobek.CyclicModuleRecord
		err error
	}
)

// EnableRequire enable the global function require to the sobek.Runtime.
func (ml *loader) EnableRequire(rt *sobek.Runtime) Loader {
	_ = rt.Set("require", ml.require)
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
		panic(rt.NewGoError(err))
	}

	instance := rt.GetModuleInstance(mod)
	if instance == nil {
		if err = mod.Link(); err != nil {
			panic(rt.NewGoError(err))
		}
		cm, ok := mod.(sobek.CyclicModuleRecord)
		if !ok {
			panic(rt.NewGoError(ErrInvalidModule))
		}
		promise := rt.CyclicModuleRecordEvaluate(cm, ml.ResolveModule)
		if promise.State() == sobek.PromiseStateRejected {
			panic(promise.Result())
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
		ml.Lock()
		defer ml.Unlock()
		if mod, ok := ml.goModules[name]; ok {
			return mod, nil
		}
		if e, ok := Get(name); ok {
			mod := &goModule{mod: e}
			ml.goModules[name] = mod
			return mod, nil
		}
		return nil, ErrNotFoundModule
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

func (ml *loader) reversePath(referencingScriptOrModule any) *url.URL {
	if referencingScriptOrModule == nil {
		return ml.base
	}
	mod, ok := referencingScriptOrModule.(sobek.ModuleRecord)
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

func (ml *loader) loadAsFileOrDirectory(modPath *url.URL, modName string) (sobek.ModuleRecord, error) {
	mod, err := ml.loadAsFile(modPath, modName)
	if err != nil {
		if isSyntaxError(err) {
			return nil, err
		}
		return ml.loadAsDirectory(modPath.JoinPath(modName))
	}
	return mod, nil
}

func (ml *loader) loadAsFile(modPath *url.URL, modName string) (module sobek.ModuleRecord, err error) {
	if module, err = ml.loadModule(modPath, modName); err == nil {
		return
	}
	if isSyntaxError(err) {
		return nil, err
	}
	if module, err = ml.loadModule(modPath, modName+".js"); err == nil {
		return
	}
	if isSyntaxError(err) {
		return nil, err
	}
	return ml.loadModule(modPath, modName+".json")
}

func (ml *loader) loadAsDirectory(modPath *url.URL) (module sobek.ModuleRecord, err error) {
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

func (ml *loader) loadNodeModules(base *url.URL, modName string) (mod sobek.ModuleRecord, err error) {
	start := base.Path
	clone := *base
	modPath := &clone
	modPath.Path = ""
	for {
		modPath.Path = filepath.Join(start, "node_modules")

		mod, err = ml.loadAsFileOrDirectory(modPath, modName)
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

	return nil, fmt.Errorf("not found module %s at %s", modName, base)
}

func (ml *loader) loadModule(modPath *url.URL, modName string) (sobek.ModuleRecord, error) {
	file := modPath.JoinPath(modName)
	specifier := file.String()

	ml.Lock()
	cache, exists := ml.modules[specifier]
	ml.Unlock()
	if exists {
		return cache.mod, cache.err
	}

	buf, err := ml.fileLoader(file, modName)
	if err != nil {
		return nil, err
	}
	mod, err := ml.CompileModule(specifier, string(buf))

	ml.Lock()
	if err == nil {
		file.Path = filepath.Dir(file.Path)
		ml.reverse[mod] = file
	}
	ml.modules[specifier] = moduleCache{mod: mod, err: err}
	ml.Unlock()
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

func isSyntaxError(err error) bool {
	_, ok := err.(*sobek.CompilerSyntaxError)
	return ok
}
