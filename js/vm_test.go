package js

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat"
	"github.com/shiroyk/cloudcat/js/loader"
	"github.com/shiroyk/cloudcat/plugin"
	"github.com/shiroyk/cloudcat/plugin/jsmodule"
	"github.com/stretchr/testify/assert"
)

func TestVMRunString(t *testing.T) {
	t.Parallel()
	vm := NewTestVM(t)

	testCases := []struct {
		script string
		want   any
	}{
		{"2", 2},
		{"let a = 1; a + 2", 3},
		{"(() => 4)()", 4},
		{"[5]", []any{int64(5)}},
		{"let a = {'key':'foo'}; a", map[string]any{"key": "foo"}},
		{"JSON.stringify({'key':7})", `{"key":7}`},
		{"JSON.stringify([8])", `[8]`},
		{"(async () => 9)()", 9},
	}

	for _, c := range testCases {
		t.Run(c.script, func(t *testing.T) {
			v, err := vm.RunString(context.Background(), c.script)
			assert.NoError(t, err)
			vv, err := Unwrap(v)
			assert.NoError(t, err)
			assert.EqualValues(t, c.want, vv)
		})
	}
}

func TestVMRunModule(t *testing.T) {
	t.Parallel()
	moduleLoader := loader.NewModuleLoader()
	cloudcat.Provide(moduleLoader)
	vm := NewTestVM(t)

	{
		testCases := []struct {
			script string
			want   any
		}{
			{"export default () => 1", 1},
			{"export default function () {let a = 1; return a + 1}", 2},
			//{"export default async () => 3", 3},
			{"const a = async () => 5; let b = await a(); export default () => b - 1", 4},
			{"export default 3 + 2", 5},
		}

		for i, c := range testCases {
			module, err := goja.ParseModule(strconv.Itoa(i), c.script, moduleLoader.ResolveModule)
			assert.NoError(t, err)
			t.Run(c.script, func(t *testing.T) {
				v, err := vm.RunModule(context.Background(), module)
				assert.NoError(t, err)
				vv, err := Unwrap(v)
				assert.NoError(t, err)
				assert.EqualValues(t, c.want, vv)
			})
		}
	}
	{
		ctx := plugin.NewContext(plugin.ContextOptions{Values: map[any]any{
			"v1": 1,
			"v2": []string{"2"},
			"v3": map[string]any{"key": 3},
		}})
		testCases := []struct {
			script string
			want   any
		}{
			{"export default (ctx) => ctx.get('v1')", 1},
			{"export default function (ctx) {return ctx.get('v2')[0]}", "2"},
			//{"export default async (ctx) => ctx.get('v3').key", 3},
			{"const a = async () => 5; let b = await a(); export default (ctx) => b - ctx.get('v1')", 4},
		}

		for i, c := range testCases {
			module, err := goja.ParseModule(strconv.Itoa(i), c.script, moduleLoader.ResolveModule)
			assert.NoError(t, err)
			t.Run(c.script, func(t *testing.T) {
				v, err := vm.RunModule(ctx, module)
				assert.NoError(t, err)
				vv, err := Unwrap(v)
				assert.NoError(t, err)
				assert.EqualValues(t, c.want, vv)
			})
		}
	}
}

func TestTimeout(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*200)
	defer cancel()

	_, err := NewTestVM(t).RunString(ctx, `while(true){}`)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestVMRunWithContext(t *testing.T) {
	t.Parallel()
	{
		vm := NewTestVM(t)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = vm.Runtime().Set("testContext", func(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
			return vm.ToValue(VMContext(vm))
		})
		v, err := vm.RunString(ctx, "testContext()")
		assert.NoError(t, err)
		assert.Equal(t, ctx, v.Export())
		assert.Equal(t, context.Background(), VMContext(vm.Runtime()))
	}
	{
		vm := NewTestVM(t)
		ctx := plugin.NewContext(plugin.ContextOptions{})
		_ = vm.Runtime().Set("testContext", func(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
			return vm.ToValue(VMContext(vm))
		})
		v, err := vm.RunString(ctx, "testContext()")
		assert.NoError(t, err)
		assert.Equal(t, ctx, v.Export())
		assert.Equal(t, context.Background(), VMContext(vm.Runtime()))
	}
}

func TestNewPromise(t *testing.T) {
	t.Parallel()
	vm := NewTestVM(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	goFunc := func(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
		return rt.ToValue(NewPromise(rt, func() (any, error) {
			time.Sleep(time.Second)
			return max(call.Argument(0).ToInteger(), call.Argument(1).ToInteger()), nil
		}))
	}
	_ = vm.Runtime().Set("max", goFunc)

	start := time.Now()

	result, err := vm.RunString(ctx, `max(1, 2)`)
	if err != nil {
		assert.NoError(t, err)
	}
	value, err := Unwrap(result)
	if err != nil {
		assert.NoError(t, err)
	}
	assert.EqualValues(t, 2, value)
	assert.EqualValues(t, 1, int(time.Now().Sub(start).Seconds()))
}

func NewTestVM(t *testing.T) VM {
	vm := NewVM()

	assertObject := vm.Runtime().NewObject()
	_ = assertObject.Set("equal", func(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
		a, err := Unwrap(call.Argument(0))
		if err != nil {
			Throw(vm, err)
		}
		b, err := Unwrap(call.Argument(1))
		if err != nil {
			Throw(vm, err)
		}
		var msg string
		if !goja.IsUndefined(call.Argument(2)) {
			msg = call.Argument(2).String()
		}
		if !assert.Equal(t, b, a, msg) {
			Throw(vm, errors.New("not equal"))
		}
		return
	})
	_ = assertObject.Set("true", func(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
		var msg string
		if !goja.IsUndefined(call.Argument(1)) {
			msg = call.Argument(1).String()
		}
		if !assert.True(t, call.Argument(0).ToBoolean(), msg) {
			Throw(vm, errors.New("should be true"))
		}
		return
	})
	_ = vm.Runtime().Set("assert", assertObject)

	return vm
}

type testFetch struct{}

func (*testFetch) Do(req *http.Request) (*http.Response, error) {
	source := `module.exports = { foo: 'bar' + require('cloudcat/gomod1').key }`
	if req.URL.Query().Get("type") == "esm" {
		source = `
import gomod1 from "cloudcat/gomod1";
const a = async () => 4; 
let b = await a(); 
export default () => gomod1.key + 1 + b`
	}
	return &http.Response{Body: io.NopCloser(strings.NewReader(source))}, nil
}

type gomod1 struct{}

func (gomod1) Exports() any { return map[string]string{"key": "gomod1"} }

type gomod2 struct{}

func (gomod2) Exports() any {
	return struct {
		Key string `js:"key"`
	}{Key: "gomod2"}
}

type gomod3 struct{}

func (gomod3) Exports() any { return map[string]string{"key": "gomod3"} }

func (gomod3) Global() {}

func TestModule(t *testing.T) {
	fetch := new(testFetch)
	mfs := fstest.MapFS{
		"node_modules/module1/index.js": &fstest.MapFile{
			Data: []byte(`export default function() { return "module1" };`),
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
	resolver := loader.NewModuleLoader(loader.WithFileLoader(func(specifier *url.URL, name string) ([]byte, error) {
		switch specifier.Scheme {
		case "http", "https":
			res, err := fetch.Do(&http.Request{URL: specifier})
			if err != nil {
				return nil, err
			}
			body, err := io.ReadAll(res.Body)
			return body, err
		case "file":
			return fs.ReadFile(mfs, specifier.Path)
		default:
			return nil, fmt.Errorf("unexpected scheme %s", specifier.Scheme)
		}
	}))
	cloudcat.Provide(resolver)
	jsmodule.Register("gomod1", new(gomod1))
	jsmodule.Register("gomod2", new(gomod2))
	jsmodule.Register("gomod3", new(gomod3))
	vm := NewTestVM(t)

	{
		testCases := map[string]string{
			"gomod1":      `assert.equal(require("cloudcat/gomod1").key, "gomod1")`,
			"gomod2":      `assert.equal(require("cloudcat/gomod2").key, "gomod2")`,
			"gomod3":      `assert.equal(gomod3.key, "gomod3")`,
			"remote cjs":  `assert.equal(require("https://foo.com/foo.min.js?type=cjs").foo, "bargomod1")`,
			"remote esm":  `assert.equal(require("https://foo.com/foo.min.js?type=esm")(), "gomod114")`,
			"module1":     `assert.equal(require("module1")(), "module1")`,
			"module2":     `assert.equal(require("module2")(), "module1/module2")`,
			"module3":     `assert.equal(require("module3")(), "module1/module2/module3")`,
			"module4":     `assert.equal(require("module4")(), "/module4")`,
			"es_script1":  `assert.equal(require("./es_script1")(), "module1/module2/module3/es_script1")`,
			"es_script2":  `assert.equal(require("./es_script2").value(), 555)`,
			"cjs_script1": `assert.equal(require("./cjs_script1")(), "/module4/cjs_script1")`,
			"cjs_script2": `assert.equal(require("./cjs_script2").value(), 555)`,
			"json1":       `assert.equal(require("./json1.json").key, "json1")`,
		}

		for k, script := range testCases {
			t.Run(fmt.Sprintf("script %s", k), func(t *testing.T) {
				_, err := vm.RunString(context.Background(), script)
				assert.NoError(t, err)
			})
		}
	}
	{
		testCases := map[string]string{
			"gomod1": `import gomod1 from "cloudcat/gomod1";
			 export default () => assert.equal(gomod1.key, "gomod1")`,
			"gomod2": `import gomod2 from "cloudcat/gomod2";
			 export default () => assert.equal(gomod2.key, "gomod2")`,
			"gomod3": `export default () => assert.equal(gomod3.key, "gomod3")`,
			"remote cjs": `import foo from "https://foo.com/foo.min.js?type=cjs";
    		 export default () => assert.equal(foo.foo, "bargomod1")`,
			"remote esm": `import foo from "https://foo.com/foo.min.js?type=esm";
    		 export default () => assert.equal(foo(), "gomod114")`,
			"module1": `import module1 from "module1";
			 export default () => assert.equal(module1(), "module1");`,
			"module2": `import m2 from "module2";
			 export default () => assert.equal(m2(), "module1/module2");`,
			"module3": `import module3 from "module3";
			 export default () => assert.equal(module3(), "module1/module2/module3");`,
			"module4": `import module4 from "module4";
			 export default () => assert.equal(module4(), "/module4");`,
			"es_script1": `import es from "./es_script1";
			 export default () => assert.equal(es(), "module1/module2/module3/es_script1");`,
			"es_script2": `import { value } from "./es_script2";
			 export default () => assert.equal(value(), 555);`,
			"cjs_script1": `import cjs from "./cjs_script1";
			 export default () => assert.equal(cjs(), "/module4/cjs_script1");`,
			"cjs_script2": `import { value } from "./cjs_script2";
			 export default () => assert.equal(value(), 555);`,
			"json1": `import j from "./json1.json";
			 export default () => assert.equal(j.key, "json1");`,
		}

		for i, script := range testCases {
			t.Run(fmt.Sprintf("module %v", i), func(t *testing.T) {
				module, err := goja.ParseModule("", script, resolver.ResolveModule)
				assert.NoError(t, err)
				_, err = vm.RunModule(context.Background(), module)
				assert.NoError(t, err)
			})
		}
	}
}
