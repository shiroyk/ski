package js

import (
	"net/url"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/cache"
)

type jsCookie struct {
	cookie cache.Cookie
}

func (c *jsCookie) Get(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	u, err := url.Parse(call.Argument(0).String())
	if err != nil {
		panic(vm.ToValue(err))
	}
	return vm.ToValue(c.cookie.CookieString(u))
}

func (c *jsCookie) Set(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	u, err := url.Parse(call.Argument(0).String())
	if err != nil {
		panic(vm.ToValue(err))
	}
	c.cookie.SetCookieString(u, call.Argument(1).String())
	return
}

func (c *jsCookie) Del(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	u, err := url.Parse(call.Argument(0).String())
	if err != nil {
		panic(vm.ToValue(err))
	}
	c.cookie.DeleteCookie(u)
	return
}
