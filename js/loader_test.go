package js

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	_ "unsafe"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski"
	"github.com/stretchr/testify/assert"
)

type fetch struct{}

func (*fetch) Do(req *http.Request) (*http.Response, error) {
	source := `module.exports = { foo: 'bar' + require('ski/gomod1').key }`
	if req.URL.Query().Get("type") == "esm" {
		source = `
import gomod1 from "ski/gomod1";
const a = async () => 4;
export default async () => gomod1.key + 1 + (await a())`
	}
	return &http.Response{Body: io.NopCloser(strings.NewReader(source))}, nil
}

type gomod1 struct{}

func (gomod1) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	return rt.ToValue(map[string]string{"key": "gomod1"}), nil
}

type gomod2 struct{}

func (gomod2) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	return rt.ToValue(struct {
		Key string `js:"key"`
	}{Key: "gomod2"}), nil
}

type gomod3 struct{}

func (gomod3) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	return rt.ToValue(map[string]string{"key": "gomod3"}), nil
}

func (gomod3) Global() {}

func TestModuleLoader(t *testing.T) {
	t.Parallel()
	fetch := new(fetch)
	mfs := fstest.MapFS{
		"node_modules/module1/index.js": &fstest.MapFile{
			Data: []byte(`module.exports = function() { return "module1" };`),
		},
		"node_modules/module2/index.js": &fstest.MapFile{
			Data: []byte(`
				import m1 from "module1";
				export default function() { return m1() + "/module2" };
				`),
		},
		"node_modules/module3/index.js": &fstest.MapFile{
			Data: []byte(`
				import module2 from "module2";
				import { module3 } from "./module3";
				export default function() { return module2() + module3() };
				`),
		},
		"node_modules/module3/module3.js": &fstest.MapFile{
			Data: []byte(`export function module3() { return "/module3" };`),
		},
		"node_modules/module4/lib/module4.js": &fstest.MapFile{
			Data: []byte(`export default () => { return "/module4" };`),
		},
		"node_modules/module4/package.json": &fstest.MapFile{
			Data: []byte(`{"main": "lib/module4.js"}`),
		},
		"node_modules/module5/lib/index.js": &fstest.MapFile{
			Data: []byte(`
				import { msg as msg6 } from "module6";
				export const msg = "/module5";
				export default () => msg + msg6;`),
		},
		"node_modules/module5/package.json": &fstest.MapFile{
			Data: []byte(`{"main": "lib/index.js"}`),
		},
		"node_modules/module6/lib/index.js": &fstest.MapFile{
			Data: []byte(`
				import { msg as msg5 } from "module5";
				export const msg = "/module6";
				export default () => msg + msg5;`),
		},
		"node_modules/module6/package.json": &fstest.MapFile{
			Data: []byte(`{"main": "lib/index.js"}`),
		},
		"node_modules/module7/index.js": &fstest.MapFile{
			Data: []byte(`export default async () => "dynamic import " + (await import('module6')).msg;`),
		},
		"es_script1.js": &fstest.MapFile{
			Data: []byte(`
				import module3 from "module3";
				export default function() { return module3() + "/es_script1" };
				`),
		},
		"es_script2.js": &fstest.MapFile{
			Data: []byte(`export const value = () => 555;`),
		},
		"cjs_script1.js": &fstest.MapFile{
			Data: []byte(`module.exports = () => { return require('module4')() + "/cjs_script1" };`),
		},
		"cjs_script2.js": &fstest.MapFile{
			Data: []byte(`
				const { value } = require('./es_script2');
				exports.value = () => value();
				`),
		},
		"json1.json": &fstest.MapFile{
			Data: []byte(`{"key": "json1"}`),
		},
	}
	loader := NewModuleLoader(WithFileLoader(func(specifier *url.URL, name string) ([]byte, error) {
		switch specifier.Scheme {
		case "http", "https":
			res, err := fetch.Do(&http.Request{URL: specifier})
			if err != nil {
				return nil, err
			}
			body, err := io.ReadAll(res.Body)
			return body, err
		case "file":
			return mfs.ReadFile(specifier.Path)
		default:
			return nil, fmt.Errorf("unexpected scheme %s", specifier.Scheme)
		}
	}))
	Register("gomod1", new(gomod1))
	Register("gomod2", new(gomod2))
	Register("gomod3", new(gomod3))
	vm := NewTestVM(t, WithModuleLoader(loader))

	{
		scriptCases := []struct{ name, s string }{
			{"gomod1", `assert.equal(require("ski/gomod1").key, "gomod1")`},
			{"gomod2", `assert.equal(require("ski/gomod2").key, "gomod2")`},
			{"gomod3", `assert.equal(gomod3.key, "gomod3")`},
			{"remote cjs", `assert.equal(require("http://foo.com/foo.min.js?type=cjs").foo, "bargomod1")`},
			{"remote esm", `(async () => assert.equal(await require("http://foo.com/foo.min.js?type=esm")(), "gomod114"))()`},
			{"module1", `assert.equal(require("module1")(), "module1")`},
			{"module2", `assert.equal(require("module2")(), "module1/module2")`},
			{"module3", `assert.equal(require("module3")(), "module1/module2/module3")`},
			{"module4", `assert.equal(require("module4")(), "/module4")`},
			{"module5", `assert.equal(require("module5")(), "/module5/module6")`},
			{"module6", `assert.equal(require("module6")(), "/module6/module5")`},
			{"module7", `(async () => assert.equal(await require("module7")(), "dynamic import /module6"))()`},
			{"es_script1", `assert.equal(require("./es_script1")(), "module1/module2/module3/es_script1")`},
			{"es_script2", `assert.equal(require("./es_script2").value(), 555)`},
			{"cjs_script1", `assert.equal(require("./cjs_script1")(), "/module4/cjs_script1")`},
			{"cjs_script2", `assert.equal(require("./cjs_script2").value(), 555)`},
			{"json1", `assert.equal(require("./json1.json").key, "json1")`},
		}

		for _, script := range scriptCases {
			t.Run(fmt.Sprintf("script %s", script.name), func(t *testing.T) {
				vm.Run(context.Background(), func() {
					_, err := vm.Runtime().RunString(script.s)
					assert.NoError(t, err)
				})
			})
		}
	}
	{
		moduleCases := []struct{ name, s string }{
			{"gomod1", `import gomod1 from "ski/gomod1";
			 export default () => assert.equal(gomod1.key, "gomod1")`},
			{"gomod2", `import gomod2 from "ski/gomod2";
			 export default () => assert.equal(gomod2.key, "gomod2")`},
			{"gomod3", `export default () => assert.equal(gomod3.key, "gomod3")`},
			{"remote cjs", `import foo from "http://foo.com/foo.min.js?type=cjs";
			 export default () => assert.equal(foo.foo, "bargomod1")`},
			{"remote esm", `import foo from "http://foo.com/foo.min.js?type=esm";
			 export default async () => assert.equal(await foo(), "gomod114")`},
			{"module1", `import module1 from "module1";
			 export default () => assert.equal(module1(), "module1");`},
			{"module2", `import m2 from "module2";
			 export default () => assert.equal(m2(), "module1/module2");`},
			{"module3", `import module3 from "module3";
			 export default () => assert.equal(module3(), "module1/module2/module3");`},
			{"module4", `import module4 from "module4";
			 export default () => assert.equal(module4(), "/module4");`},
			{"module5", `import module5 from "module5";
			 export default () => assert.equal(module5(), "/module5/module6");`},
			{"module6", `import module6 from "module6";
			 export default () => assert.equal(module6(), "/module6/module5");`},
			{"module7", `import module7 from "module7";
			 export default async () => assert.equal(await module7(), "dynamic import /module6");`},
			{"es_script1", `import es from "./es_script1";
			 export default () => assert.equal(es(), "module1/module2/module3/es_script1");`},
			{"es_script2", `import { value } from "./es_script2";
			 export default () => assert.equal(value(), 555);`},
			{"cjs_script1", `import cjs from "./cjs_script1";
			 export default () => assert.equal(cjs(), "/module4/cjs_script1");`},
			{"cjs_script2", `import { value } from "./cjs_script2";
			 export default () => assert.equal(value(), 555);`},
			{"json1", `import j from "./json1.json";
			 export default () => assert.equal(j.key, "json1");`},
		}

		for _, script := range moduleCases {
			t.Run(fmt.Sprintf("module %v", script.name), func(t *testing.T) {
				mod, err := loader.CompileModule("", script.s)
				if assert.NoError(t, err) {
					_, err = vm.RunModule(context.Background(), mod)
					assert.NoError(t, err)
				}
			})
		}
	}
}

func TestConcurrentLoader(t *testing.T) {
	t.Parallel()
	num := 8

	mfs := make(fstest.MapFS, num)
	for i := 0; i < num; i++ {
		mfs[fmt.Sprintf("module%d.js", i)] = &fstest.MapFile{Data: []byte(`export default () => ` + strconv.Itoa(i))}
	}

	fileLoader := WithFileLoader(func(specifier *url.URL, name string) ([]byte, error) {
		return fs.ReadFile(mfs, specifier.Path)
	})
	scheduler := NewScheduler(SchedulerOptions{
		InitialVMs: 2,
		Loader:     NewModuleLoader(fileLoader),
	})

	var wg sync.WaitGroup

	for i := 0; i < num; i++ {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()

			vm, err := scheduler.Get()
			if assert.NoError(t, err) {
				mod, err := scheduler.Loader().CompileModule("", fmt.Sprintf(`
				import m from './module%d.js';
				export default () => m()`, j))
				if assert.NoError(t, err) {
					v, err := vm.RunModule(context.Background(), mod)
					if assert.NoError(t, err) {
						assert.Equal(t, int64(j), v.ToInteger())
					}
				}
			}
		}(i)
	}

	wg.Wait()
}

type testExec struct{ v any }

func new_testExec(arg ...ski.Executor) (ski.Executor, error) {
	return testExec{ski.ExecToString(arg[0])}, nil
}

func (t testExec) Exec(context.Context, any) (any, error) { return t.v, nil }

func TestJSExecutor(t *testing.T) {
	ski.Register("loader_executor", new_testExec)
	ski.Register("loader_executor.other", new_testExec)
	vm := NewTestVM(t, WithModuleLoader(NewModuleLoader()))

	for i, s := range []string{
		`assert.equal(require("executor/loader_executor")('foo').exec(''), 'foo');`,
		`assert.equal(require("executor/loader_executor").other('bar').exec(''), 'bar');`,
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			_, err := vm.Runtime().RunString(s)
			assert.NoError(t, err)
		})
	}
}

func TestESMExecutor(t *testing.T) {
	exec, err := new_executor()(ski.String(`export default (ctx) => ctx.get('content') + 1`))
	if assert.NoError(t, err) {
		v, err := exec.Exec(context.Background(), "a")
		if assert.NoError(t, err) {
			assert.Equal(t, "a1", v)
		}
	}
}

func NewTestVM(t *testing.T, opts ...Option) VM {
	vm := NewVM(opts...)
	p := vm.Runtime().NewObject()
	_ = p.Set("equal", func(call sobek.FunctionCall) sobek.Value {
		assert.Equal(t, call.Argument(1).Export(), call.Argument(0).Export(), call.Argument(2).String())
		return sobek.Undefined()
	})
	_ = vm.Runtime().Set("assert", p)
	return vm
}
