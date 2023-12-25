package js

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/plugin/jsmodule"
	"github.com/stretchr/testify/assert"
)

type testModuleFetch struct{}

func (*testModuleFetch) Do(req *http.Request) (*http.Response, error) {
	source := `module.exports = { foo: 'bar' + require('cloudcat/gomod1').key }`
	if req.URL.Query().Get("type") == "esm" {
		source = `
import gomod1 from "cloudcat/gomod1";
const a = async () => 4;
export default async () => gomod1.key + 1 + (await a())`
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
	t.Parallel()
	fetch := new(testModuleFetch)
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
			Data: []byte(`exports.default = () => { return require('module4').default() + "/cjs_script1" };`),
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
	resolver := NewModuleLoader(WithFileLoader(func(specifier *url.URL, name string) ([]byte, error) {
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
	jsmodule.Register("gomod1", new(gomod1))
	jsmodule.Register("gomod2", new(gomod2))
	jsmodule.Register("gomod3", new(gomod3))
	vm := NewTestVM(t, resolver)

	{
		scriptCases := []struct{ name, s string }{
			{"gomod1", `assert.equal(require("cloudcat/gomod1").key, "gomod1")`},
			{"gomod2", `assert.equal(require("cloudcat/gomod2").key, "gomod2")`},
			{"gomod3", `assert.equal(gomod3.key, "gomod3")`},
			{"remote cjs", `assert.equal(require("https://foo.com/foo.min.js?type=cjs").foo, "bargomod1")`},
			{"remote esm", `async () => assert.equal(await require("https://foo.com/foo.min.js?type=esm").default(), "gomod114")`},
			{"module1", `assert.equal(require("module1").default(), "module1")`},
			{"module2", `assert.equal(require("module2").default(), "module1/module2")`},
			{"module3", `assert.equal(require("module3").default(), "module1/module2/module3")`},
			{"module4", `assert.equal(require("module4").default(), "/module4")`},
			{"module5", `assert.equal(require("module5").default(), "/module5/module6")`},
			{"module6", `assert.equal(require("module6").default(), "/module6/module5")`},
			{"module7", `async () => assert.equal(await require("module7").default(), "dynamic import /module6")`},
			{"es_script1", `assert.equal(require("./es_script1").default(), "module1/module2/module3/es_script1")`},
			{"es_script2", `assert.equal(require("./es_script2").value(), 555)`},
			{"cjs_script1", `assert.equal(require("./cjs_script1").default(), "/module4/cjs_script1")`},
			{"cjs_script2", `assert.equal(require("./cjs_script2").value(), 555)`},
			{"json1", `assert.equal(require("./json1.json").key, "json1")`},
		}

		for _, script := range scriptCases {
			t.Run(fmt.Sprintf("script %s", script.name), func(t *testing.T) {
				_, err := vm.RunString(context.Background(), script.s)
				assert.NoError(t, err)
			})
		}
	}
	{
		moduleCases := []struct{ name, s string }{
			{"gomod1", `import gomod1 from "cloudcat/gomod1";
			 export default () => assert.equal(gomod1.key, "gomod1")`},
			{"gomod2", `import gomod2 from "cloudcat/gomod2";
			 export default () => assert.equal(gomod2.key, "gomod2")`},
			{"gomod3", `export default () => assert.equal(gomod3.key, "gomod3")`},
			{"remote cjs", `import foo from "https://foo.com/foo.min.js?type=cjs";
			 export default () => assert.equal(foo.foo, "bargomod1")`},
			{"remote esm", `import foo from "https://foo.com/foo.min.js?type=esm";
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
				module, err := goja.ParseModule("", script.s, resolver.ResolveModule)
				if assert.NoError(t, err) {
					_, err = vm.RunModule(context.Background(), module)
					assert.NoError(t, err)
				}
			})
		}
	}
}
