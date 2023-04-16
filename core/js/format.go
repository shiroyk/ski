package js

import (
	"bytes"

	"github.com/dop251/goja"
)

func runeFormat(vm *goja.Runtime, f rune, val goja.Value, w *bytes.Buffer) bool {
	switch f {
	case 's':
		w.WriteString(val.String())
	case 'd':
		w.WriteString(val.ToNumber().String())
	case 'j':
		if json, ok := vm.Get("JSON").(*goja.Object); ok {
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
func Format(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	var b bytes.Buffer
	var fmt string

	if arg := call.Argument(0); !goja.IsUndefined(arg) {
		fmt = arg.String()
	}

	var args []goja.Value
	if len(call.Arguments) > 0 {
		args = call.Arguments[1:]
	}
	bufferFormat(vm, &b, fmt, args...)

	return vm.ToValue(b.String())
}
