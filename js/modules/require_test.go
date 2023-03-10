package modules

import (
	"strconv"
	"testing"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/fetch"
	"github.com/shiroyk/cloudcat/js/common"
	"github.com/stretchr/testify/assert"
)

type testModule struct{}

func (testModule) Exports() any {
	return map[string]string{"key": "test"}
}

func TestRequire(t *testing.T) {
	di.Provide(fetch.NewFetcher(fetch.Options{}), false)

	vm := goja.New()
	vm.SetFieldNameMapper(goja.UncapFieldNameMapper())
	EnableRequire(vm)
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
			`const lodash = require("https://cdn.jsdelivr.net/npm/lodash@4.17.21/lodash.min.js");
		 assert.equal(lodash.VERSION, "4.17.21")`,
		},
		{
			`const base64 = require("https://cdn.jsdelivr.net/npm/js-base64@3.7.5/base64.min.js");
		 assert.equal(base64.version, "3.7.5")`,
		},
	}

	for i, testCase := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			_, err := vm.RunString(testCase.script)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
