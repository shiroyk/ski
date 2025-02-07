package js

import (
	"bytes"
	"encoding/json"
	"log/slog"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski"
)

// console implements the js console
type console struct{}

// EnableConsole enables the console with the slog.Logger
func EnableConsole(rt *sobek.Runtime) {
	_ = rt.Set("console", new(console))
}

func (c *console) log(level slog.Level, call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	ski.Logger(Context(rt)).Log(Context(rt), level, Format(call, rt).String())
	return sobek.Undefined()
}

// Log calls slog.Log.
func (c *console) Log(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return c.log(slog.LevelInfo, call, rt)
}

// Info calls slog.Info.
func (c *console) Info(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return c.log(slog.LevelInfo, call, rt)
}

// Warn calls slog.Warn.
func (c *console) Warn(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return c.log(slog.LevelWarn, call, rt)
}

// Warn calls slog.Error.
func (c *console) Error(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return c.log(slog.LevelError, call, rt)
}

// Debug calls slog.Debug.
func (c *console) Debug(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return c.log(slog.LevelDebug, call, rt)
}

func runeFormat(rt *sobek.Runtime, f rune, val sobek.Value, w *bytes.Buffer) bool {
	switch f {
	case 's':
		w.WriteString(val.String())
	case 'd':
		w.WriteString(val.ToNumber().String())
	case 'j':
		if json, ok := rt.Get("JSON").(*sobek.Object); ok {
			if stringify, ok := sobek.AssertFunction(json.Get("stringify")); ok {
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

func bufferFormat(vm *sobek.Runtime, b *bytes.Buffer, f string, args ...sobek.Value) {
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
func Format(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	var f string

	if arg := call.Argument(0); !sobek.IsUndefined(arg) {
		m, ok := arg.(json.Marshaler)
		if ok {
			data, err := m.MarshalJSON()
			if err != nil {
				Throw(rt, err)
			}
			f = string(data)
		} else {
			f = arg.String()
		}
	}

	if len(call.Arguments) > 1 {
		var b bytes.Buffer
		bufferFormat(rt, &b, f, call.Arguments[1:]...)
		f = b.String()
	}

	return rt.ToValue(f)
}
