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

func (c *jsContext) GetString(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	key := call.Argument(0).String()
	content := call.Argument(1).String()
	arg := call.Argument(2).String()

	if p, ok := parser.GetParser(key); ok {
		str, err := p.GetString(c.ctx, content, arg)
		if err != nil {
			panic(vm.ToValue(err))
		}
		return vm.ToValue(str)
	}

	return
}

func (c *jsContext) GetStrings(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	key := call.Argument(0).String()
	content := call.Argument(1).String()
	arg := call.Argument(2).String()

	if p, ok := parser.GetParser(key); ok {
		str, err := p.GetStrings(c.ctx, content, arg)
		if err != nil {
			panic(vm.ToValue(err))
		}
		return vm.ToValue(str)
	}

	return
}

func (c *jsContext) GetElement(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	key := call.Argument(0).String()
	content := call.Argument(1).String()
	arg := call.Argument(2).String()

	if p, ok := parser.GetParser(key); ok {
		str, err := p.GetElement(c.ctx, content, arg)
		if err != nil {
			panic(vm.ToValue(err))
		}
		return vm.ToValue(str)
	}

	return
}

func (c *jsContext) GetElements(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	key := call.Argument(0).String()
	content := call.Argument(1).String()
	arg := call.Argument(2).String()

	if p, ok := parser.GetParser(key); ok {
		str, err := p.GetElements(c.ctx, content, arg)
		if err != nil {
			panic(vm.ToValue(err))
		}
		return vm.ToValue(str)
	}

	return
}
