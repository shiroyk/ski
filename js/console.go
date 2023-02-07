package js

import (
	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/js/common"
	"golang.org/x/exp/slog"
)

func EnableConsole(vm *goja.Runtime) {
	_ = vm.Set("console", &Console{})
}

type Console struct{}

func (c *Console) log(level slog.Level, call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	slog.Log(level, common.Format(call, vm).String())
	return goja.Undefined()
}

func (c *Console) Log(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return c.log(slog.LevelInfo, call, vm)
}

func (c *Console) Info(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return c.log(slog.LevelInfo, call, vm)
}

func (c *Console) Warn(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return c.log(slog.LevelWarn, call, vm)
}

func (c *Console) Error(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return c.log(slog.LevelError, call, vm)
}
