package js

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/js/common"
	"github.com/stretchr/testify/assert"
)

func TestConsole(t *testing.T) {
	t.Parallel()
	vm := goja.New()
	vm.SetFieldNameMapper(common.FieldNameMapper{})
	EnableConsole(vm)

	_, err := vm.RunString(`
		console.log("hello %s", "cloudcat");
	`)
	assert.NoError(t, err)
}
