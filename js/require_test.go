package js

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/plugin/jsmodule"
	"github.com/stretchr/testify/assert"
)

type testFetcher struct{}

func (*testFetcher) Do(*http.Request) (*http.Response, error) {
	return &http.Response{Body: io.NopCloser(strings.NewReader("module.exports = { foo: 'bar' }"))}, nil
}

type testRModule struct{}

func (testRModule) Exports() any { return map[string]string{"key": "testr"} }

type testRGModule struct{}

func (testRGModule) Exports() any { return map[string]string{"key": "testrg"} }

func (testRGModule) Global() {}

func TestRequire(t *testing.T) {
	jsmodule.Register("testr", new(testRModule))
	jsmodule.Register("testrg", new(testRGModule))
	vm := createTestVM(t)

	testCases := []struct {
		script string
	}{
		{
			`const testr = require("cloudcat/testr");
		 assert.equal(testr.key, "testr")`,
		},
		{
			`assert.equal(testrg.key, "testrg")`,
		},
		{
			`const foo = require("https://foo.com/foo.min.js");
		 assert.equal(foo.foo, "bar")`,
		},
	}

	for i, testCase := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			_, err := vm.RunString(context.Background(), testCase.script)
			assert.NoError(t, err)
		})
	}
}

func createTestVM(t *testing.T) VM {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.UncapFieldNameMapper())

	req := &require{
		vm:          vm,
		modules:     make(map[string]*goja.Object),
		nodeModules: make(map[string]*goja.Object),
		fetcher:     &testFetcher{},
	}

	_ = vm.Set("require", req.Require)
	InitGlobalModule(vm)

	assertObject := vm.NewObject()
	_ = assertObject.Set("equal", func(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
		a, err := Unwrap(call.Argument(0))
		if err != nil {
			Throw(vm, err)
		}
		b, err := Unwrap(call.Argument(1))
		if err != nil {
			Throw(vm, err)
		}
		return vm.ToValue(assert.Equal(t, a, b, call.Argument(2).String()))
	})
	_ = assertObject.Set("true", func(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
		return vm.ToValue(assert.True(t, call.Argument(0).ToBoolean(), call.Argument(2).String()))
	})
	_ = vm.Set("assert", assertObject)
	return &vmImpl{vm, make(chan struct{}, 1), false}
}
