package cookie

import (
	"net/url"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/cache"
	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/js/common"
	"github.com/shiroyk/cloudcat/js/modules"
)

// Module js module
type Module struct{}

// Exports returns module instance
func (*Module) Exports() any {
	return &Cookie{di.MustResolve[cache.Cookie]()}
}

func init() {
	modules.Register("cookie", &Module{})
}

// Cookie manages storage and use of cookies in HTTP requests.
type Cookie struct {
	cookie cache.Cookie
}

// Get returns the cookies string for the given URL.
func (c *Cookie) Get(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	u, err := url.Parse(call.Argument(0).String())
	if err != nil {
		common.Throw(vm, err)
	}
	return vm.ToValue(c.cookie.CookieString(u))
}

// Set handles the receipt of the cookies strung in a reply for the given URL.
func (c *Cookie) Set(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	u, err := url.Parse(call.Argument(0).String())
	if err != nil {
		common.Throw(vm, err)
	}
	c.cookie.SetCookieString(u, call.Argument(1).String())
	return
}

// Del handles the receipt of the cookies in a reply for the given URL.
func (c *Cookie) Del(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	u, err := url.Parse(call.Argument(0).String())
	if err != nil {
		common.Throw(vm, err)
	}
	c.cookie.DeleteCookie(u)
	return
}
