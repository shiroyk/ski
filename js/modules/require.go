package modules

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"text/template"

	"github.com/dop251/goja"
	"github.com/dop251/goja/parser"
	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/ext"
	"github.com/shiroyk/cloudcat/fetch"
)

// NodeJS module search algorithm described by
// https://nodejs.org/api/modules.html#modules_all_together

type require struct {
	vm          *goja.Runtime
	nodeModules map[string]*goja.Object

	globalFolders []string
}

// Require load a js module from path or URL
func (r *require) Require(name string) goja.Value {
	if e, ok := ext.Get(ext.JSExtension)[name]; ok {
		if module, ok := e.Module.(Module); ok {
			return r.vm.ToValue(module.Exports())
		}
	}
	module, err := r.resolve(name)
	if err != nil {
		return goja.Undefined()
	}
	return module.Get("exports")
}

func (r *require) resolve(name string) (*goja.Object, error) {
	if name == "" {
		return nil, ErrIllegalModuleName
	}

	if strings.Contains(name, "://") {
		u, err := url.Parse(name)
		if err != nil {
			return nil, err
		}
		if u.Scheme == "http" || u.Scheme == "https" {
			return r.resolveRemote(u)
		}
	}

	return r.resolveFile(name)
}

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
	if module = r.nodeModules[p]; module != nil {
		return
	}
	module, err = r.loadNodeModules(modPath, start)
	if err == nil && module != nil {
		r.nodeModules[p] = module
	}

	if module == nil && err == nil {
		err = ErrInvalidModule
	}
	return
}

func (r *require) resolveRemote(u *url.URL) (module *goja.Object, err error) {
	p := u.String()
	fetcher, err := di.Resolve[fetch.Fetch]()
	if err != nil {
		return nil, err
	}
	res, err := fetcher.Get(p, nil)
	if err != nil {
		return nil, err
	}

	module = r.vm.NewObject()
	_ = module.Set("exports", r.vm.NewObject())
	if err = r.compileModule(p, res.String(), module); err != nil {
		return nil, err
	}

	return
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

	return nil, ErrInvalidModule
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
	module := r.nodeModules[path]
	if module == nil {
		module = r.vm.NewObject()
		_ = module.Set("exports", r.vm.NewObject())
		r.nodeModules[path] = module
		err := r.loadModuleFile(path, module)
		if err != nil {
			module = nil
			delete(r.nodeModules, path)
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
	} else {
		return ErrInvalidModule
	}

	return nil
}
