package js

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/plugin"
	"github.com/shiroyk/cloudcat/plugin/parser"
	"golang.org/x/exp/slog"
)

var attr = slog.String("type", "js")

// Cat an analyzer context
type Cat struct {
	ctx     *plugin.Context
	BaseURL string
	URL     string `js:"url"`
}

// NewCat returns a new Cat instance
func NewCat(ctx *plugin.Context) *Cat {
	return &Cat{ctx, ctx.BaseURL(), ctx.URL()}
}

// Log print the msg to logger
func (c *Cat) Log(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	c.ctx.Logger().Info(Format(call, vm).String(), attr)
	return goja.Undefined()
}

// GetVar returns the value associated with this context for key, or nil
// if no value is associated with key.
func (c *Cat) GetVar(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return vm.ToValue(c.ctx.Value(call.Argument(0).String()))
}

// SetVar value associated with key is val.
func (c *Cat) SetVar(key string, value goja.Value) error {
	v, err := Unwrap(value)
	if err != nil {
		return err
	}
	c.ctx.SetValue(key, v)
	return nil
}

// ClearVar clean all values
func (c *Cat) ClearVar() {
	c.ctx.ClearValue()
}

// Cancel this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this Context complete.
func (c *Cat) Cancel() {
	c.ctx.Cancel()
}

// GetString gets the string of the content with the given arguments
func (c *Cat) GetString(key string, rule string, content any) (ret string, err error) {
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
func (c *Cat) GetStrings(key string, rule string, content any) (ret []string, err error) {
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
func (c *Cat) GetElement(key string, rule string, content any) (ret string, err error) {
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
func (c *Cat) GetElements(key string, rule string, content any) (ret []string, err error) {
	str, err := ToStrings(content)
	if err != nil {
		return
	}

	if p, ok := parser.GetParser(key); ok {
		return p.GetElements(c.ctx, str, rule)
	}

	return ret, fmt.Errorf("parser %s not found", key)
}
