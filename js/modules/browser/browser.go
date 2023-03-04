package browser

import (
	"encoding/json"
	"reflect"

	"github.com/dop251/goja"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/js/common"
	"github.com/shiroyk/cloudcat/js/modules"
)

// Module js module
type Module struct{}

// Exports returns module instance
func (*Module) Exports() any {
	return &Browser{di.MustResolve[*rod.Browser]()}
}

func init() {
	modules.Register("browser", &Module{})
}

// Browser module represents the browser. It doesn't depends on file system,
// it should work with remote browser seamlessly.
type Browser struct { //nolint:var-naming
	browser *rod.Browser
}

// Page returns a new page
func (b *Browser) Page(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if call.Argument(0).ExportType().Kind() == reflect.String {
		page := b.browser.MustPage(call.Argument(0).String())
		return vm.ToValue(Page{page})
	}

	target := jsonValue[proto.TargetCreateTarget](call.Argument(0), vm)
	page, err := b.browser.Page(target)
	if err != nil {
		common.Throw(vm, err)
	}
	return vm.ToValue(Page{page})
}

// jsonValue serializes the js object to golang struct.
func jsonValue[T any](value goja.Value, vm *goja.Runtime) (t T) {
	bytes, err := value.ToObject(vm).MarshalJSON()
	if err != nil {
		common.Throw(vm, err)
	}
	err = json.Unmarshal(bytes, &t)
	if err != nil {
		common.Throw(vm, err)
	}
	return
}
