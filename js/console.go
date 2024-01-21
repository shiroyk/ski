package js

import (
	"bytes"
	"log/slog"

	"github.com/dop251/goja"
	"github.com/shiroyk/ski"
)

// console implements the js console
type console struct{}

// EnableConsole enables the console with the slog.Logger
func EnableConsole(rt *goja.Runtime) {
	_ = rt.Set("console", new(console))
}

func (c *console) log(level slog.Level, call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	ski.Logger(Context(rt)).Log(Context(rt), level, Format(call, rt).String())
	return goja.Undefined()
}

// Log calls slog.Log.
func (c *console) Log(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	return c.log(slog.LevelInfo, call, rt)
}

// Info calls slog.Info.
func (c *console) Info(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	return c.log(slog.LevelInfo, call, rt)
}

// Warn calls slog.Warn.
func (c *console) Warn(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	return c.log(slog.LevelWarn, call, rt)
}

// Warn calls slog.Error.
func (c *console) Error(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	return c.log(slog.LevelError, call, rt)
}

// Debug calls slog.Debug.
func (c *console) Debug(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	return c.log(slog.LevelDebug, call, rt)
}

func runeFormat(rt *goja.Runtime, f rune, val goja.Value, w *bytes.Buffer) bool {
	switch f {
	case 's':
		w.WriteString(val.String())
	case 'd':
		w.WriteString(val.ToNumber().String())
	case 'j':
		if json, ok := rt.Get("JSON").(*goja.Object); ok {
			if stringify, ok := goja.AssertFunction(json.Get("stringify")); ok {
				res, err := stringify(json, val)
				if err != nil {
					panic(err)
				}
				w.WriteString(res.String())
			}
		}
	case '%':
		w.WriteByte('%')
		return false
	default:
		w.WriteByte('%')
		w.WriteRune(f)
		return false
	}
	return true
}

func bufferFormat(vm *goja.Runtime, b *bytes.Buffer, f string, args ...goja.Value) {
	pct := false
	argNum := 0
	for _, chr := range f {
		if pct { //nolint:nestif
			if argNum < len(args) {
				if runeFormat(vm, chr, args[argNum], b) {
					argNum++
				}
			} else {
				b.WriteByte('%')
				b.WriteRune(chr)
			}
			pct = false
		} else {
			if chr == '%' {
				pct = true
			} else {
				b.WriteRune(chr)
			}
		}
	}

	for _, arg := range args[argNum:] {
		b.WriteByte(' ')
		b.WriteString(arg.String())
	}
}

// Format implements js format
func Format(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	var b bytes.Buffer
	var f string

	if arg := call.Argument(0); !goja.IsUndefined(arg) {
		f = arg.String()
	}

	var args []goja.Value
	if len(call.Arguments) > 0 {
		args = call.Arguments[1:]
	}
	bufferFormat(rt, &b, f, args...)

	return rt.ToValue(b.String())
}
