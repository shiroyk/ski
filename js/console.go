package js

import (
	"context"

	"github.com/dop251/goja"
	"log/slog"
)

// console implements the js console
type console struct{}

// EnableConsole enables the console
func EnableConsole(vm *goja.Runtime) {
	_ = vm.Set("console", new(console))
}

func (c *console) log(level slog.Level, call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	slog.Log(context.Background(), level, Format(call, vm).String())
	return goja.Undefined()
}

// Log calls Logger.Log.
func (c *console) Log(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return c.log(slog.LevelInfo, call, vm)
}

// Info calls Logger.Info.
func (c *console) Info(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return c.log(slog.LevelInfo, call, vm)
}

// Warn calls Logger.Warn.
func (c *console) Warn(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return c.log(slog.LevelWarn, call, vm)
}

// Warn calls Logger.Error.
func (c *console) Error(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return c.log(slog.LevelError, call, vm)
}
