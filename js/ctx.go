package js

import (
	"fmt"
	"log/slog"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/plugin"
	"github.com/shiroyk/cloudcat/plugin/parser"
)

var attr = slog.String("source", "js")

// ctxWrapper an analyzer context
type ctxWrapper struct {
	ctx     *plugin.Context
	BaseURL string
	URL     string `js:"url"`
}

// NewCtxWrapper returns a new ctxWrapper instance
func NewCtxWrapper(vm VM, ctx *plugin.Context) goja.Value {
	return vm.Runtime().ToValue(&ctxWrapper{ctx, ctx.BaseURL(), ctx.URL()})
}

// Log print the msg to logger
func (c *ctxWrapper) Log(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	c.ctx.Logger().Info(Format(call, vm).String(), attr)
	return goja.Undefined()
}

// Get returns the value associated with this context for key, or nil
// if no value is associated with key.
func (c *ctxWrapper) Get(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return vm.ToValue(c.ctx.Value(call.Argument(0).String()))
}

// Set value associated with key is val.
func (c *ctxWrapper) Set(key string, value goja.Value) error {
	v, err := Unwrap(value)
	if err != nil {
		return err
	}
	c.ctx.SetValue(key, v)
	return nil
}

// ClearVar clean all values
func (c *ctxWrapper) ClearVar() {
	c.ctx.ClearValue()
}

// Cancel this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this Context complete.
func (c *ctxWrapper) Cancel() {
	c.ctx.Cancel()
}

// GetString gets the string of the content with the given arguments
func (c *ctxWrapper) GetString(key string, rule string, content any) (ret string, err error) {
	str, err := ToStrings(content)
	if err != nil {
		return
	}

	if p, ok := parser.GetParser(key); ok {
		return p.GetString(c.ctx, str, rule)
	}

	return ret, fmt.Errorf("parser %s not found", key)
}

// GetStrings gets the string of the content with the given arguments
func (c *ctxWrapper) GetStrings(key string, rule string, content any) (ret []string, err error) {
	str, err := ToStrings(content)
	if err != nil {
		return
	}

	if p, ok := parser.GetParser(key); ok {
		return p.GetStrings(c.ctx, str, rule)
	}

	return ret, fmt.Errorf("parser %s not found", key)
}

// GetElement gets the string of the content with the given arguments
func (c *ctxWrapper) GetElement(key string, rule string, content any) (ret string, err error) {
	str, err := ToStrings(content)
	if err != nil {
		return
	}

	if p, ok := parser.GetParser(key); ok {
		return p.GetElement(c.ctx, str, rule)
	}

	return ret, fmt.Errorf("parser %s not found", key)
}

// GetElements gets the string of the content with the given arguments
func (c *ctxWrapper) GetElements(key string, rule string, content any) (ret []string, err error) {
	str, err := ToStrings(content)
	if err != nil {
		return
	}

	if p, ok := parser.GetParser(key); ok {
		return p.GetElements(c.ctx, str, rule)
	}

	return ret, fmt.Errorf("parser %s not found", key)
}
