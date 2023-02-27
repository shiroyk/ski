package console

import (
	"context"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/js/common"
	"github.com/shiroyk/cloudcat/js/modules"
	"golang.org/x/exp/slog"
)

// Module js module
type Module struct{}

// Exports returns module instance
func (*Module) Exports() any {
	return &Console{}
}

// Global returns is it is a global module
func (*Module) Global() {}

func init() {
	modules.Register("console", &Module{})
}

// Console implements the js Console
type Console struct{}

func (c *Console) log(level slog.Level, call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	slog.Log(context.Background(), level, common.Format(call, vm).String())
	return goja.Undefined()
}

// Log calls Logger.Log.
func (c *Console) Log(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return c.log(slog.LevelInfo, call, vm)
}

// Info calls Logger.Info.
func (c *Console) Info(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return c.log(slog.LevelInfo, call, vm)
}

// Warn calls Logger.Warn.
func (c *Console) Warn(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return c.log(slog.LevelWarn, call, vm)
}

// Warn calls Logger.Error.
func (c *Console) Error(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return c.log(slog.LevelError, call, vm)
}
