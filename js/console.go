package js

import (
	"bytes"
	"context"
	"log/slog"

	"github.com/grafana/sobek"
)

func EnableConsole(rt *sobek.Runtime, attr ...slog.Attr) {
	v, _ := console(attr).Instantiate(rt)
	_ = rt.Set("console", v)
}

// console implements the js console
type console []slog.Attr

func (c console) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	obj := rt.NewObject()
	_ = obj.Set("log", c.log)
	_ = obj.Set("info", c.info)
	_ = obj.Set("warn", c.warn)
	_ = obj.Set("error", c.error)
	return obj, nil
}

func (c console) output(level slog.Level, call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	ctx := Context(rt)
	Logger(ctx).LogAttrs(ctx, level, Format(rt, call.Arguments...), c...)
	return sobek.Undefined()
}

// log calls slog.Log.
func (c console) log(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return c.output(slog.LevelInfo, call, rt)
}

// info calls slog.Info.
func (c console) info(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return c.output(slog.LevelInfo, call, rt)
}

// warn calls slog.Warn.
func (c console) warn(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return c.output(slog.LevelWarn, call, rt)
}

// error calls slog.Error.
func (c console) error(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return c.output(slog.LevelError, call, rt)
}

// debug calls slog.Debug.
func (c console) debug(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return c.output(slog.LevelDebug, call, rt)
}

func runeFormat(rt *sobek.Runtime, f rune, val sobek.Value, w *bytes.Buffer) bool {
	switch f {
	case 's':
		w.WriteString(val.String())
	case 'd':
		w.WriteString(val.ToNumber().String())
	case 'j':
		if j, ok := rt.Get("JSON").(*sobek.Object); ok {
			if stringify, ok := sobek.AssertFunction(j.Get("stringify")); ok {
				res, err := stringify(j, val)
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
		b.WriteString(valueString(arg))
	}
}

func valueString(v sobek.Value) string {
	if obj, ok := v.(*sobek.Object); ok {
		switch obj.ClassName() {
		case "Error":
			if stack := obj.Get("stack"); stack != nil {
				return stack.String()
			}
		default:
			data, err := obj.MarshalJSON()
			if err == nil {
				return string(data)
			}
		}
	}
	return v.String()
}

// Format js console format
func Format(rt *sobek.Runtime, args ...sobek.Value) string {
	var s string
	if len(args) > 0 {
		s = valueString(args[0])
	}
	if len(args) > 1 {
		var b bytes.Buffer
		bufferFormat(rt, &b, s, args[1:]...)
		s = b.String()
	}
	return s
}

type loggerKey struct{}

// Logger get slog.Logger from the context
func Logger(ctx context.Context) *slog.Logger {
	if logger := ctx.Value(loggerKey{}); logger != nil {
		return logger.(*slog.Logger)
	}
	return slog.Default()
}

// WithLogger set the slog.Logger to context
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	if c, ok := ctx.(interface{ SetValue(key, value any) }); ok {
		c.SetValue(loggerKey{}, logger)
		return ctx
	}
	return context.WithValue(ctx, loggerKey{}, logger)
}
