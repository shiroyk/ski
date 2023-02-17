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

func TestRequire(t *testing.T) {
	di.Override(fetch.NewFetcher(fetch.Options{}))
	vm := goja.New()
	vm.SetFieldNameMapper(goja.UncapFieldNameMapper())
	EnableRequire(vm)

	testCases := []struct {
		script, want string
	}{
		{`const lodash = require("https://cdn.jsdelivr.net/npm/lodash@4.17.21/lodash.min.js");
		lodash.VERSION;`,
			"4.17.21",
		},
		{`const base64 = require("https://cdn.jsdelivr.net/npm/js-base64@3.7.5/base64.min.js");
		base64.version;`,
			"3.7.5",
		},
	}

	for i, testCase := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			value, err := vm.RunString(testCase.script)
			if err != nil {
				t.Error(err)
			}
			v, err := common.Unwrap(value)
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, testCase.want, v)
		})
	}
}
