package loader

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
	"text/template"

	"github.com/dop251/goja"
	"github.com/dop251/goja/parser"
	"github.com/shiroyk/cloudcat"
	"github.com/shiroyk/cloudcat/plugin/jsmodule"
)

var (
	// ErrInvalidModule module is invalid
	ErrInvalidModule = errors.New("invalid module")
	// ErrIllegalModuleName module name is illegal
	ErrIllegalModuleName = errors.New("illegal module name")
)

type (
	// ModuleLoader the js module loader.
	ModuleLoader interface {
		// EnableRequire enable the global function require to the goja.Runtime.
		EnableRequire(rt *goja.Runtime)
		// ResolveModule resolve the module returns the goja.ModuleRecord.
		ResolveModule(any, string) (goja.ModuleRecord, error)
	}

	// FileLoader is a type alias for a function that returns the contents of the referenced file.
	FileLoader func(specifier *url.URL, name string) ([]byte, error)
)

// Option the default moduleLoader options.
type Option func(*moduleLoader)

// WithBase the base directory of module loader.
func WithBase(base *url.URL) Option {
	return func(o *moduleLoader) {
		o.base = base
	}
}

// WithFileLoader the file loader of module loader.
func WithFileLoader(fileLoader FileLoader) Option {
	return func(o *moduleLoader) {
		o.fileLoader = fileLoader
	}
}

// NewModuleLoader returns a new module resolver
// if the fileLoader option not provided, uses the default DefaultFileLoader.
func NewModuleLoader(opts ...Option) ModuleLoader {
	mr := &moduleLoader{
		modules:   make(map[string]moduleCache),
		goModules: make(map[string]goja.CyclicModuleRecord),
		reverse:   make(map[goja.ModuleRecord]*url.URL),
	}

	for _, option := range opts {
		option(mr)
	}

	if mr.base == nil {
		mr.base = &url.URL{Scheme: "file", Path: "."}
	}
	if mr.fileLoader == nil {
		mr.fileLoader = DefaultFileLoader()
	}
	mr.parserOption = parser.WithSourceMapLoader(func(path string) ([]byte, error) {
		uri, err := url.Parse(path)
		if err != nil {
			return nil, err
		}
		return mr.fileLoader(uri, path)
	})
	return mr
}

// DefaultFileLoader the default file loader.
// Supports file and HTTP scheme loading.
func DefaultFileLoader() FileLoader {
	fetch := cloudcat.MustResolveLazy[cloudcat.Fetch]()
	return func(specifier *url.URL, name string) ([]byte, error) {
		switch specifier.Scheme {
		case "http", "https":
			req, err := http.NewRequest(http.MethodGet, specifier.String(), nil)
			if err != nil {
				return nil, err
			}
			res, err := fetch().Do(req)
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
		modules    map[string]moduleCache
		goModules  map[string]goja.CyclicModuleRecord
		reverse    map[goja.ModuleRecord]*url.URL
		fileLoader FileLoader

		base         *url.URL
		parserOption parser.Option
	}

	moduleCache struct {
		mod goja.ModuleRecord
		err error
	}
)

// EnableRequire enable the global function require to the goja.Runtime.
func (ml *moduleLoader) EnableRequire(rt *goja.Runtime) { _ = rt.Set("require", ml.require) }

// require resolve the module instance.
func (ml *moduleLoader) require(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	name := call.Argument(0).String()
	module, err := ml.ResolveModule(nil, name)
	if err != nil {
		panic(rt.ToValue(err))
	}
	if nm, ok := module.(*goModule); ok {
		return rt.ToValue(nm.mod.Exports())
	}
	if err = module.Link(); err != nil {
		panic(rt.ToValue(err))
	}
	cm, ok := module.(goja.CyclicModuleRecord)
	if !ok {
		panic(rt.ToValue(ErrInvalidModule))
	}
	promise := rt.CyclicModuleRecordEvaluate(cm, ml.ResolveModule)
	switch promise.State() {
	case goja.PromiseStateRejected:
		panic(promise.Result())
	case goja.PromiseStatePending:
		// TODO TLA
		println(name, "evaluate pending")
	}
	if cjs, ok := module.(*cjsModule); ok {
		return rt.GetModuleInstance(cjs).(*cjsModuleInstance).exports
	}
	obj := rt.NamespaceObjectFor(cm)
	if d := obj.Get("default"); d != nil {
		return d
	}
	return obj
}

// ResolveModule resolve the module returns the goja.ModuleRecord.
func (ml *moduleLoader) ResolveModule(referencingScriptOrModule any, name string) (goja.ModuleRecord, error) {
	switch {
	case strings.HasPrefix(name, jsmodule.ExtPrefix):
		if mod, ok := ml.goModules[name]; ok {
			return mod, nil
		}
		if e, ok := jsmodule.GetModule(name); ok {
			mod := &goModule{mod: e}
			ml.goModules[name] = mod
			return mod, nil
		}
		return nil, ErrIllegalModuleName
	default:
		return ml.resolve(ml.reversePath(referencingScriptOrModule), name)
	}
}

func (ml *moduleLoader) resolve(base *url.URL, modPath string) (goja.ModuleRecord, error) {
	modName := filepath.Base(modPath)
	if modName == "" {
		return nil, ErrIllegalModuleName
	}

	if isBasePath(modPath) {
		return ml.loadAsFileOrDirectory(base, modName)
	}

	uri, err := url.Parse(modPath)
	if err != nil {
		return nil, err
	}
	if uri.Scheme != "" && uri.Scheme != "file" {
		return ml.loadModule(uri, "")
	}

	// default scheme file
	uri.Scheme = "file"
	uri.Path = strings.TrimSuffix(uri.Path, modName)
	mod, err := ml.loadNodeModules(uri, modName)
	if err != nil {
		return nil, fmt.Errorf("module %s not found with error %s", modPath, err)
	}
	return mod, nil
}

func (ml *moduleLoader) reversePath(referencingScriptOrModule any) *url.URL {
	if referencingScriptOrModule == nil {
		return ml.base
	}
	p, ok := ml.reverse[referencingScriptOrModule.(goja.ModuleRecord)]
	if !ok {
		if referencingScriptOrModule != nil {
			// TODO fix this
		}
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

func (ml *moduleLoader) loadNodeModules(modPath *url.URL, modName string) (mod goja.ModuleRecord, err error) {
	start := modPath.Path
	for {
		var p string
		if path.Base(start) != "node_modules" {
			p = path.Join(start, "node_modules")
		} else {
			p = start
		}
		if mod, err = ml.loadAsFileOrDirectory(modPath.JoinPath(p), modName); mod != nil || err != nil {
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

	return nil, fmt.Errorf("not found module %s", modPath)
}

func (ml *moduleLoader) loadModule(modPath *url.URL, modName string) (goja.ModuleRecord, error) {
	file := modPath.JoinPath(modName)
	specifier := file.String()
	cache, exists := ml.modules[specifier]
	if exists {
		return cache.mod, cache.err
	}

	buf, err := ml.fileLoader(file, modName)
	if err != nil {
		return nil, err
	}
	mod, err := ml.compileModule(specifier, string(buf))
	ml.reverse[mod] = modPath
	ml.modules[specifier] = moduleCache{mod: mod, err: err}
	return mod, err
}

func (ml *moduleLoader) compileModule(path, source string) (goja.ModuleRecord, error) {
	if filepath.Ext(path) == ".json" {
		source = "module.exports = JSON.parse('" + template.JSEscapeString(source) + "')"
		return ml.compileCjsModule(path, source)
	}

	ast, err := goja.Parse(path, source, parser.IsModule, ml.parserOption)
	if err != nil {
		return nil, err
	}

	isModule := len(ast.ExportEntries) > 0 || len(ast.ImportEntries) > 0 || ast.HasTLA
	if !isModule {
		return ml.compileCjsModule(path, source)
	}

	return goja.ModuleFromAST(ast, ml.ResolveModule)
}

func (ml *moduleLoader) compileCjsModule(path, source string) (goja.ModuleRecord, error) {
	source = "(function(exports, require, module) {" + source + "\n})"

	ast, err := goja.Parse(path, source, ml.parserOption)
	if err != nil {
		return nil, err
	}

	prg, err := goja.CompileAST(ast, false)
	if err != nil {
		return nil, err
	}

	return &cjsModule{prg: prg}, nil
}

func isBasePath(modPath string) bool {
	return strings.HasPrefix(modPath, "./") ||
		strings.HasPrefix(modPath, "/") ||
		strings.HasPrefix(modPath, "../") ||
		modPath == "." || modPath == ".."
}