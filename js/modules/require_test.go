package modules

import (
	"strconv"
	"testing"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/fetch"
	"github.com/shiroyk/cloudcat/js/common"
	"github.com/stretchr/testify/assert"
)

type testFetcher struct{}

func (*testFetcher) DoRequest(*fetch.Request) (*fetch.Response, error) {
	return &fetch.Response{Body: []byte("module.exports = { foo: 'bar' }")}, nil
}

type testModule struct{}

func (testModule) Exports() any {
	return map[string]string{"key": "test"}
}

func TestRequire(t *testing.T) {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.UncapFieldNameMapper())
	req := &require{
		vm:          vm,
		modules:     make(map[string]*goja.Object),
		nodeModules: make(map[string]*goja.Object),
		fetcher:     &testFetcher{},
	}
	_ = vm.Set("require", req.Require)
	Register("test", &testModule{})

	assertObject := vm.NewObject()
	_ = assertObject.Set("equal", func(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
		a, err := common.Unwrap(call.Argument(0))
		if err != nil {
			common.Throw(vm, err)
		}
		b, err := common.Unwrap(call.Argument(1))
		if err != nil {
			common.Throw(vm, err)
		}
		return vm.ToValue(assert.Equal(t, a, b, call.Argument(2).String()))
	})
	_ = assertObject.Set("true", func(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
		return vm.ToValue(assert.True(t, call.Argument(0).ToBoolean(), call.Argument(2).String()))
	})
	_ = vm.Set("assert", assertObject)

	testCases := []struct {
		script string
	}{
		{
			`const test = require("cloudcat/test");
		 assert.equal(test.key, "test")`,
		},
		{
			`const foo = require("https://foo.com/foo.min.js");
		 assert.equal(foo.foo, "bar")`,
		},
	}

	for i, testCase := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			_, err := vm.RunString(testCase.script)
			assert.NoError(t, err)
		})
	}
}
