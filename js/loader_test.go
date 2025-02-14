package js

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"testing/fstest"
	_ "unsafe"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/modules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	return rt.ToValue(map[string]string{"globalgomod3": "gomod3"}), nil
}

func (gomod3) Global() {}

type nodeURL struct{}

func (nodeURL) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	ret := rt.NewObject()
	ctor := rt.ToValue(func(call sobek.ConstructorCall) *sobek.Object {
		u, err := url.Parse(call.Argument(0).String())
		if err != nil {
			panic(err)
		}
		instance := rt.ToValue(u).ToObject(rt)
		_ = instance.SetPrototype(call.This.Prototype())
		return instance
	}).ToObject(rt)
	_ = ctor.SetPrototype(rt.NewObject())
	_ = ret.Set("URL", ctor)
	return ret, nil
}

func (nodeURL) Global() {}

func TestModuleLoader(t *testing.T) {
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
			Data: []byte(`{"module": "lib/index.js"}`),
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
		"node_modules/node:file/index.js": &fstest.MapFile{
			Data: []byte(`export default () => "node file module"`),
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
			Data: []byte(`module.exports = () => { return require('module4').default() + "/cjs_script1" };`),
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
		"syntaxError.js": &fstest.MapFile{
			Data: []byte(`a {}`),
		},
	}
	ml := modules.NewLoader(modules.WithFileLoader(func(specifier *url.URL, name string) ([]byte, error) {
		switch specifier.Scheme {
		case "http", "https":
			if specifier.Query().Get("type") == "esm" {
				return []byte(`
				import gomod1 from "ski/gomod1";
				const a = async () => 4;
				export default async () => gomod1.key + 1 + (await a())`), nil
			}
			return []byte(`module.exports = { foo: 'bar' + require('ski/gomod1').key }`), nil
		case "file":
			return mfs.ReadFile(specifier.Path)
		default:
			return nil, fmt.Errorf("unexpected scheme %s", specifier.Scheme)
		}
	}))
	setLoader(t, ml)
	modules.Register("gomod1", new(gomod1))
	modules.Register("gomod2", new(gomod2))
	modules.Register("gomod3", new(gomod3))
	modules.Register("node:url", new(nodeURL))
	vm := NewTestVM(t)

	t.Run("script", func(t *testing.T) {
		scriptCases := []struct{ name, s string }{
			{"gomod1", `assert.equal(require("ski/gomod1").key, "gomod1")`},
			{"gomod2", `assert.equal(require("ski/gomod2").key, "gomod2")`},
			{"gomod3", `assert.equal(globalgomod3, "gomod3")`},
			{"remote cjs", `assert.equal(require("https://foo.com/foo.min.js?type=cjs").foo, "bargomod1")`},
			{"remote esm", `(async () => assert.equal(await require("https://foo.com/foo.min.js?type=esm")(), "gomod114"))()`},
			{"module1", `assert.equal(require("module1")(), "module1")`},
			{"module2", `assert.equal(require("module2").default(), "module1/module2")`},
			{"module3", `assert.equal(require("module3").default(), "module1/module2/module3")`},
			{"module4", `assert.equal(require("module4").default(), "/module4")`},
			{"module5", `assert.equal(require("module5").default(), "/module5/module6")`},
			{"module6", `assert.equal(require("module6").default(), "/module6/module5")`},
			{"module7", `(async () => assert.equal(await require("module7").default(), "dynamic import /module6"))()`},
			{"es_script1", `assert.equal(require("./es_script1").default(), "module1/module2/module3/es_script1")`},
			{"es_script2", `assert.equal(require("./es_script2").value(), 555)`},
			{"cjs_script1", `assert.equal(require("./cjs_script1")(), "/module4/cjs_script1")`},
			{"cjs_script2", `assert.equal(require("./cjs_script2").value(), 555)`},
			{"json1", `assert.equal(require("./json1.json").key, "json1")`},
			{"node url", `
			const NODE_URL = require("node:url").URL;
			assert.equal(new NODE_URL("https://example.com").toString(), "https://example.com");
			assert.true(require("node:url").URL.prototype === URL.prototype, 'prototype not equal');
			`},
			{"node file", `
			assert.equal(require("node:file").default(), "node file module");
			`},
		}

		for _, script := range scriptCases {
			t.Run(fmt.Sprintf("script %s", script.name), func(t *testing.T) {
				_, err := vm.RunString(context.Background(), script.s)
				assert.NoError(t, err)
			})
		}
	})

	t.Run("module", func(t *testing.T) {
		moduleCases := []struct{ name, s string }{
			{"gomod1", `import gomod1 from "ski/gomod1";
			assert.equal(gomod1.key, "gomod1")`},
			{"gomod2", `import gomod2 from "ski/gomod2";
			assert.equal(gomod2.key, "gomod2")`},
			{"gomod3", `assert.equal(globalgomod3, "gomod3")`},
			{"remote cjs", `import foo from "https://foo.com/foo.min.js?type=cjs";
			assert.equal(foo.foo, "bargomod1")`},
			{"remote esm", `import foo from "https://foo.com/foo.min.js?type=esm";
			export default async () => assert.equal(await foo(), "gomod114")`},
			{"module1", `import module1 from "module1";
			assert.equal(module1(), "module1");`},
			{"module2", `import m2 from "module2";
			assert.equal(m2(), "module1/module2");`},
			{"module3", `import module3 from "module3";
			assert.equal(module3(), "module1/module2/module3");`},
			{"module4", `import module4 from "module4";
			assert.equal(module4(), "/module4");`},
			{"module5", `import module5 from "module5";
			assert.equal(module5(), "/module5/module6");`},
			{"module6", `import module6 from "module6";
			assert.equal(module6(), "/module6/module5");`},
			{"module7", `import module7 from "module7";
			export default async () => assert.equal(await module7(), "dynamic import /module6");`},
			{"es_script1", `import es from "./es_script1";
			assert.equal(es(), "module1/module2/module3/es_script1");`},
			{"es_script2", `import { value } from "./es_script2";
			assert.equal(value(), 555);`},
			{"cjs_script1", `import cjs from "./cjs_script1";
			assert.equal(cjs(), "/module4/cjs_script1");`},
			{"cjs_script2", `import { value } from "./cjs_script2";
			assert.equal(value(), 555);`},
			{"json1", `import j from "./json1.json";
			assert.equal(j.key, "json1");`},
			{"node url", `import {URL as NODE_URL} from "node:url";
			assert.equal(new NODE_URL("https://example.com").toString(), "https://example.com");
			assert.true(NODE_URL.prototype === URL.prototype, 'prototype not equal');`},
			{"node file", `import node_file from "node:file";
			assert.equal(node_file(), "node file module");
			`},
		}

		for _, script := range moduleCases {
			t.Run(fmt.Sprintf("module %v", script.name), func(t *testing.T) {
				mod, err := ml.CompileModule("", script.s)
				require.NoError(t, err)
				result, err := vm.RunModule(context.Background(), mod)
				require.NoError(t, err)
				if result == nil {
					return
				}
				promise, ok := result.Export().(*sobek.Promise)
				if !ok {
					return
				}
				switch promise.State() {
				case sobek.PromiseStateRejected:
					require.FailNow(t, fmt.Sprintf("promise rejected: %s", promise.Result().String()))
				case sobek.PromiseStateFulfilled:
				default:
					require.FailNow(t, "unexpected promise state")
				}
			})
		}
	})

	t.Run("error", func(t *testing.T) {
		testCases := []struct {
			name, script string
			expected     string
		}{
			{
				name:     "import syntax",
				script:   `import test from "./syntaxError"`,
				expected: "SyntaxError",
			},
			{
				name:     "require syntax",
				script:   `require("./syntaxError")`,
				expected: `Error: SyntaxError: file://syntaxError.js:`,
			},
			{
				name:     "not found error",
				script:   `import test from "some"`,
				expected: "cannot found module",
			},
			{
				name:     "native not found",
				script:   `import test from "ski/some_module"`,
				expected: "cannot found module",
			},
			{
				name:     "require not found",
				script:   `require("some")`,
				expected: "Error: cannot found module 'some'",
			},
		}

		for _, tt := range testCases {
			t.Run(tt.name, func(t *testing.T) {
				mod, err := ml.CompileModule("", tt.script)
				require.NoError(t, err)
				_, err = vm.RunModule(context.Background(), mod)
				assert.ErrorContains(t, err, tt.expected)
			})
		}
	})
}

func NewTestVM(t *testing.T, opts ...Option) VM {
	vm := NewVM(opts...)
	p := vm.Runtime().NewObject()
	_ = p.Set("equal", func(call sobek.FunctionCall) sobek.Value {
		assert.Equal(t, call.Argument(0).Export(), call.Argument(1).Export(), call.Argument(2).String())
		return sobek.Undefined()
	})
	_ = p.Set("true", func(call sobek.FunctionCall) sobek.Value {
		assert.True(t, call.Argument(0).ToBoolean(), call.Argument(1).String())
		return sobek.Undefined()
	})
	_ = vm.Runtime().Set("assert", p)
	return vm
}

func setLoader(t *testing.T, loader modules.Loader) {
	current := Loader()
	SetLoader(loader)
	t.Cleanup(func() { SetLoader(current) })
}
