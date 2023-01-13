package js

import (
	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/parser"
)

type jsContext struct {
	ctx                  *parser.Context
	BaseUrl, RedirectUrl string
	Content              any
	Http                 jsHttp
	Cache                jsCache
	Cookie               jsCookie
	Shortener            jsShortener
}

func (c *jsContext) GetVar(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return vm.ToValue(c.ctx.Value(call.Argument(0).String()))
}

func (c *jsContext) SetVar(call goja.FunctionCall) (ret goja.Value) {
	c.ctx.SetValue(call.Argument(0).String(), call.Argument(1).Export())
	return
}

func (c *jsContext) ClearVar(_ goja.FunctionCall) (ret goja.Value) {
	c.ctx.ClearValue()
	return
}
