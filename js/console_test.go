package js

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
)

func TestConsole(t *testing.T) {
	t.Parallel()
	vm := goja.New()
	vm.SetFieldNameMapper(FieldNameMapper{})
	EnableConsole(vm)

	_, err := vm.RunString(`
		console.log("hello %s", "cloudcat");
		console.log("json %j", {'foo': 'bar'});
	`)
	assert.NoError(t, err)
}
