package js

import (
	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/js/common"
	"github.com/shiroyk/cloudcat/parser"
)

// Cat an analyzer context
type Cat struct {
	ctx                  *parser.Context
	BaseURL, RedirectURL string
}

// NewCat returns a new Cat instance
func NewCat(ctx *parser.Context) *Cat {
	return &Cat{ctx, ctx.BaseURL(), ctx.RedirectURL()}
}

// Log print the msg to logger
func (c *Cat) Log(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	c.ctx.Logger().Info(common.Format(call, vm).String())
	return goja.Undefined()
}

// GetVar returns the value associated with this context for key, or nil
// if no value is associated with key.
func (c *Cat) GetVar(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return vm.ToValue(c.ctx.Value(call.Argument(0).String()))
}

// SetVar value associated with key is val.
func (c *Cat) SetVar(call goja.FunctionCall) (ret goja.Value) {
	c.ctx.SetValue(call.Argument(0).String(), call.Argument(1).Export())
	return
}

// ClearVar clean all values
func (c *Cat) ClearVar(_ goja.FunctionCall) (ret goja.Value) {
	c.ctx.ClearValue()
	return
}

// GetString gets the string of the content with the given arguments
func (c *Cat) GetString(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	key := call.Argument(0).String()
	content := call.Argument(1).String()
	arg := call.Argument(2).String()

	if p, ok := parser.GetParser(key); ok {
		str, err := p.GetString(c.ctx, content, arg)
		if err != nil {
			common.Throw(vm, err)
		}
		return vm.ToValue(str)
	}

	return
}

// GetStrings gets the string of the content with the given arguments
func (c *Cat) GetStrings(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	key := call.Argument(0).String()
	content := call.Argument(1).String()
	arg := call.Argument(2).String()

	if p, ok := parser.GetParser(key); ok {
		str, err := p.GetStrings(c.ctx, content, arg)
		if err != nil {
			common.Throw(vm, err)
		}
		return vm.ToValue(str)
	}

	return
}

// GetElement gets the string of the content with the given arguments
func (c *Cat) GetElement(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	key := call.Argument(0).String()
	content := call.Argument(1).String()
	arg := call.Argument(2).String()

	if p, ok := parser.GetParser(key); ok {
		str, err := p.GetElement(c.ctx, content, arg)
		if err != nil {
			common.Throw(vm, err)
		}
		return vm.ToValue(str)
	}

	return
}

// GetElements gets the string of the content with the given arguments
func (c *Cat) GetElements(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	key := call.Argument(0).String()
	content := call.Argument(1).String()
	arg := call.Argument(2).String()

	if p, ok := parser.GetParser(key); ok {
		str, err := p.GetElements(c.ctx, content, arg)
		if err != nil {
			common.Throw(vm, err)
		}
		return vm.ToValue(str)
	}

	return
}
