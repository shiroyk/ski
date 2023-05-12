package js

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"text/template"

	"github.com/dop251/goja"
	"github.com/dop251/goja/parser"
	"github.com/shiroyk/cloudcat/core"
	"github.com/shiroyk/cloudcat/plugin/jsmodule"
)

var (
	// ErrInvalidModule module is invalid
	ErrInvalidModule = errors.New("invalid module")
	// ErrIllegalModuleName module name is illegal
	ErrIllegalModuleName = errors.New("illegal module name")

	// ErrModuleFileDoesNotExist module not exist
	ErrModuleFileDoesNotExist = errors.New("module file does not exist")
)

// Copyright dop251/goja_nodejs, licensed under the MIT License.
// NodeJS module search algorithm described by
// https://nodejs.org/api/modules.html#modules_all_together

// EnableRequire set runtime require module
func EnableRequire(vm *goja.Runtime, path ...string) {
	fetcher, _ := cloudcat.Resolve[cloudcat.Fetch]()
	req := &require{
		vm:            vm,
		modules:       make(map[string]*goja.Object),
		nodeModules:   make(map[string]*goja.Object),
		globalFolders: path,
		fetcher:       fetcher,
	}

	_ = vm.Set("require", req.Require)
}

type require struct {
	vm          *goja.Runtime
	modules     map[string]*goja.Object
	nodeModules map[string]*goja.Object
	fetcher     cloudcat.Fetch

	globalFolders []string
}

// Require load a js module from path or URL
func (r *require) Require(name string) (export goja.Value, err error) {
	var module *goja.Object
	switch {
	case name == "":
		err = ErrIllegalModuleName
	case isHTTP(name):
		module, err = r.resolveRemote(name)
	case strings.HasPrefix(name, jsmodule.ExtPrefix):
		return r.resolveNative(name)
	default:
		module, err = r.resolveFile(name)
	}
	if err != nil {
		return nil, err
	}
	return module.Get("exports"), nil
}

func (r *require) resolveNative(name string) (*goja.Object, error) {
	if native, ok := r.modules[name]; ok {
		return native, nil
	}
	if e, ok := jsmodule.GetModule(name); ok {
		mod := r.vm.ToValue(e.Exports()).ToObject(r.vm)
		r.modules[name] = mod
		return mod, nil
	}
	return nil, ErrIllegalModuleName
}

//nolint:nakedret
func (r *require) resolveFile(modPath string) (module *goja.Object, err error) {
	origPath, modPath := modPath, path.Clean(modPath)
	if modPath == "" {
		return nil, ErrIllegalModuleName
	}

	var start string
	err = nil
	if path.IsAbs(origPath) {
		start = "/"
	} else {
		start = r.getCurrentModulePath()
	}

	p := path.Join(start, modPath)

	if strings.HasPrefix(origPath, "./") || //nolint:nestif
		strings.HasPrefix(origPath, "/") ||
		strings.HasPrefix(origPath, "../") ||
		origPath == "." || origPath == ".." {
		if module = r.modules[p]; module != nil {
			return
		}
		module, err = r.loadAsFileOrDirectory(p)
		if err == nil && module != nil {
			r.modules[p] = module
		}
	} else {
		if module = r.nodeModules[p]; module != nil {
			return
		}
		module, err = r.loadNodeModules(modPath, start)
		if err == nil && module != nil {
			r.nodeModules[p] = module
		}
	}

	if module == nil && err == nil {
		err = ErrInvalidModule
	}
	return
}

func (r *require) resolveRemote(name string) (module *goja.Object, err error) {
	data, cached, err := r.fetchFile(name)
	if err != nil {
		return nil, err
	}
	if mod, exists := r.modules[name]; exists {
		if cached {
			return mod, nil
		}
	}

	module = r.vm.NewObject()
	_ = module.Set("exports", r.vm.NewObject())
	r.modules[name] = module

	source := "(function(exports, require, module) {" + string(data) + "\n})"
	if err = r.compileModule(name, source, module); err != nil {
		delete(r.modules, name)
		return nil, err
	}

	return
}

func (r *require) fetchFile(name string) ([]byte, bool, error) {
	if r.fetcher == nil {
		r.fetcher = cloudcat.MustResolve[cloudcat.Fetch]()
	}
	req, err := http.NewRequest(http.MethodGet, name, nil)
	if err != nil {
		return nil, false, err
	}
	res, err := r.fetcher.Do(req)
	if err != nil {
		return nil, false, err
	}
	body, err := io.ReadAll(res.Body)
	return body, cloudcat.IsFromCache(res), err
}

func (r *require) loadAsFileOrDirectory(path string) (module *goja.Object, err error) {
	if module, err = r.loadAsFile(path); module != nil || err != nil {
		return
	}

	return r.loadAsDirectory(path)
}

func (r *require) loadAsFile(path string) (module *goja.Object, err error) {
	if module, err = r.loadModule(path); module != nil || err != nil {
		return
	}

	p := path + ".js"
	if module, err = r.loadModule(p); module != nil || err != nil {
		return
	}

	p = path + ".json"
	return r.loadModule(p)
}

func (r *require) loadIndex(modPath string) (module *goja.Object, err error) {
	p := path.Join(modPath, "index.js")
	if module, err = r.loadModule(p); module != nil || err != nil {
		return
	}

	p = path.Join(modPath, "index.json")
	return r.loadModule(p)
}

func (r *require) loadAsDirectory(modPath string) (module *goja.Object, err error) {
	p := path.Join(modPath, "package.json")
	buf, err := r.loadSource(p)
	if err != nil {
		return r.loadIndex(modPath)
	}
	var pkg struct {
		Main string `json:"main"`
	}
	err = json.Unmarshal(buf, &pkg)
	if err != nil || len(pkg.Main) == 0 {
		return r.loadIndex(modPath)
	}

	m := path.Join(modPath, pkg.Main)
	if module, err = r.loadAsFile(m); module != nil || err != nil {
		return
	}

	return r.loadIndex(m)
}

// loadSource is used loads files from the host's filesystem.
func (r *require) loadSource(filename string) ([]byte, error) {
	if isHTTP(filename) {
		data, _, err := r.fetchFile(filename)
		if err != nil {
			return nil, err
		}
		return data, nil
	}
	data, err := os.ReadFile(filepath.FromSlash(filename))
	if err != nil {
		if os.IsNotExist(err) || errors.Is(err, syscall.EISDIR) {
			err = ErrModuleFileDoesNotExist
		}
	}
	return data, err
}

func (r *require) loadNodeModule(modPath, start string) (*goja.Object, error) {
	return r.loadAsFileOrDirectory(path.Join(start, modPath))
}

func (r *require) loadNodeModules(modPath, start string) (module *goja.Object, err error) {
	for _, dir := range r.globalFolders {
		if module, err = r.loadNodeModule(modPath, dir); module != nil || err != nil {
			return
		}
	}
	for {
		var p string
		if path.Base(start) != "node_modules" {
			p = path.Join(start, "node_modules")
		} else {
			p = start
		}
		if module, err = r.loadNodeModule(modPath, p); module != nil || err != nil {
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

func (r *require) getCurrentModulePath() string {
	var buf [2]goja.StackFrame
	frames := r.vm.CaptureCallStack(2, buf[:0])
	if len(frames) < 2 {
		return "."
	}
	return path.Dir(frames[1].SrcName())
}

func (r *require) loadModule(path string) (*goja.Object, error) {
	module := r.modules[path]
	if module == nil {
		module = r.vm.NewObject()
		_ = module.Set("exports", r.vm.NewObject())
		r.modules[path] = module
		err := r.loadModuleFile(path, module)
		if err != nil {
			module = nil
			delete(r.modules, path)
			if errors.Is(err, ErrModuleFileDoesNotExist) {
				err = nil
			}
		}
		return module, err
	}
	return module, nil
}

func (r *require) loadModuleFile(p string, jsModule *goja.Object) error {
	buf, err := r.loadSource(p)
	if err != nil {
		return err
	}
	s := string(buf)

	if path.Ext(p) == ".json" {
		s = "module.exports = JSON.parse('" + template.JSEscapeString(s) + "')"
	}

	source := "(function(exports, require, module) {" + s + "\n})"

	return r.compileModule(p, source, jsModule)
}

func (r *require) compileModule(path, source string, jsModule *goja.Object) error {
	parsed, err := goja.Parse(path, source, parser.WithSourceMapLoader(r.loadSource))
	if err != nil {
		return err
	}

	prg, err := goja.CompileAST(parsed, false)
	if err != nil {
		return err
	}

	f, err := r.vm.RunProgram(prg)
	if err != nil {
		return err
	}

	if call, ok := goja.AssertFunction(f); ok {
		jsExports := jsModule.Get("exports")
		jsRequire := r.vm.Get("require")

		// Run the module source, with "jsExports" as "this",
		// "jsExports" as the "exports" variable, "jsRequire"
		// as the "require" variable and "jsModule" as the
		// "module" variable (Nodejs capable).
		_, err = call(jsExports, jsExports, jsRequire, jsModule)
		if err != nil {
			return err
		}
		return nil
	}

	return ErrInvalidModule
}

func isHTTP(name string) bool {
	return strings.HasPrefix(name, "http://") || strings.HasPrefix(name, "https://")
}
