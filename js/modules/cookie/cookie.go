package cookie

import (
	"net/url"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/cache"
	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/js/common"
	"github.com/shiroyk/cloudcat/js/modules"
)

type Module struct{}

func (*Module) Exports() any {
	return &Cookie{di.MustResolve[cache.Cookie]()}
}

func init() {
	modules.Register("cookie", &Module{})
}

type Cookie struct {
	cookie cache.Cookie
}

func (c *Cookie) Get(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	u, err := url.Parse(call.Argument(0).String())
	if err != nil {
		common.Throw(vm, err)
	}
	return vm.ToValue(c.cookie.CookieString(u))
}

func (c *Cookie) Set(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	u, err := url.Parse(call.Argument(0).String())
	if err != nil {
		common.Throw(vm, err)
	}
	c.cookie.SetCookieString(u, call.Argument(1).String())
	return
}

func (c *Cookie) Del(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	u, err := url.Parse(call.Argument(0).String())
	if err != nil {
		common.Throw(vm, err)
	}
	c.cookie.DeleteCookie(u)
	return
}
