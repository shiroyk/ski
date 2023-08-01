package js

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/shiroyk/cloudcat"
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
	cloudcat.Provide[cloudcat.Fetch](new(testFetcher))
	jsmodule.Register("testr", new(testRModule))
	jsmodule.Register("testrg", new(testRGModule))
	vm := NewTestVM(t)

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
