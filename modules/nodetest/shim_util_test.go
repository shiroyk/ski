package nodetest

import (
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/modules/buffer"
)

// newUtilShim implements the util.inspect() subset used by the node buffer
// test suite, the output follows node for buffers and typed arrays.
func newUtilShim(rt *sobek.Runtime) *sobek.Object {
	inspectFn := func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
		depth := 2
		if options := call.Argument(1); options != nil && !sobek.IsUndefined(options) {
			if d := options.ToObject(rt).Get("depth"); d != nil && !sobek.IsUndefined(d) {
				depth = int(d.ToInteger())
			}
		}
		return rt.ToValue(inspectValue(rt, call.Argument(0), depth, make(map[*sobek.Object]bool)))
	}
	inspect := rt.ToValue(inspectFn).ToObject(rt)
	if symbolFor, ok := sobek.AssertFunction(rt.Get("Symbol").ToObject(rt).Get("for")); ok {
		if custom, err := symbolFor(sobek.Undefined(), rt.ToValue("nodejs.util.inspect.custom")); err == nil {
			_ = inspect.Set("custom", custom)
		}
	}

	util := rt.NewObject()
	_ = util.Set("inspect", inspect)
	_ = util.Set("types", map[string]any{
		"isUint8Array": func(value sobek.Value) bool {
			if value == nil {
				return false
			}
			ctor, ok := rt.Get("Uint8Array").(*sobek.Object)
			return ok && rt.InstanceOf(value, ctor)
		},
		"isAnyArrayBuffer": func(value sobek.Value) bool {
			if value == nil {
				return false
			}
			_, ok := value.Export().(sobek.ArrayBuffer)
			return ok
		},
	})
	return util
}

func inspectValue(rt *sobek.Runtime, value sobek.Value, depth int, seen map[*sobek.Object]bool) string {
	switch {
	case value == nil || sobek.IsUndefined(value):
		return "undefined"
	case sobek.IsNull(value):
		return "null"
	case sobek.IsString(value):
		return "'" + value.String() + "'"
	case sobek.IsBigInt(value):
		return value.String() + "n"
	case sobek.IsNumber(value), value.ExportType() == reflect.TypeOf(false):
		return value.String()
	}

	object, ok := value.(*sobek.Object)
	if !ok {
		return value.String()
	}
	if _, callable := sobek.AssertFunction(object); callable {
		name := ""
		if value := object.Get("name"); value != nil {
			name = value.String()
		}
		return fmt.Sprintf("[Function: %s]", name)
	}
	if buffer.IsBuffer(rt, object) {
		return inspectBuffer(rt, object, depth, seen)
	}
	if isTypedArray(rt, object) {
		return inspectTypedArray(rt, object)
	}
	if depth < 0 {
		return "[Object]"
	}
	if seen[object] {
		return "[Circular]"
	}
	seen[object] = true
	defer delete(seen, object)

	keys := object.Keys()
	if object.ExportType().Kind() == reflect.Slice || object.ExportType().Kind() == reflect.Array {
		items := make([]string, 0, len(keys))
		for _, key := range keys {
			items = append(items, inspectValue(rt, object.Get(key), depth-1, seen))
		}
		return "[ " + strings.Join(items, ", ") + " ]"
	}

	items := make([]string, 0, len(keys))
	for _, key := range keys {
		items = append(items, key+": "+inspectValue(rt, object.Get(key), depth-1, seen))
	}
	return "{ " + strings.Join(items, ", ") + " }"
}

func inspectBuffer(rt *sobek.Runtime, value *sobek.Object, depth int, seen map[*sobek.Object]bool) string {
	length := value.Get("length").ToInteger()
	limit := length
	if ctor, ok := rt.Get("Buffer").(*sobek.Object); ok {
		if defined := ctor.Get("INSPECT_MAX_BYTES"); defined != nil && !sobek.IsUndefined(defined) {
			if max := defined.ToFloat(); !math.IsNaN(max) && max < float64(limit) {
				limit = int64(max)
			}
		}
	}
	bytes := make([]string, 0, limit)
	for i := range limit {
		bytes = append(bytes, fmt.Sprintf("%02x", value.Get(fmt.Sprint(i)).ToInteger()&0xff))
	}
	body := strings.Join(bytes, " ")
	if length > limit {
		more := length - limit
		noun := "bytes"
		if more == 1 {
			noun = "byte"
		}
		body += fmt.Sprintf(" ... %d more %s", more, noun)
	}
	for _, key := range value.Keys() {
		if isIndex(key) {
			continue
		}
		if body != "" {
			body += ", "
		}
		body += key + ": " + inspectValue(rt, value.Get(key), depth-1, seen)
	}
	return "<Buffer " + body + ">"
}

func inspectTypedArray(rt *sobek.Runtime, value *sobek.Object) string {
	length := value.Get("length").ToInteger()
	items := make([]string, 0, length)
	for i := range length {
		item := value.Get(fmt.Sprint(i))
		if sobek.IsBigInt(item) {
			items = append(items, item.String()+"n")
		} else {
			items = append(items, item.String())
		}
	}
	if len(items) == 0 {
		return fmt.Sprintf("%s(0) []", className(rt, value))
	}
	return fmt.Sprintf("%s(%d) [ %s ]", className(rt, value), length, strings.Join(items, ", "))
}

// className returns the constructor name of the value.
func className(rt *sobek.Runtime, value *sobek.Object) string {
	if constructor := value.Get("constructor"); constructor != nil && !sobek.IsUndefined(constructor) {
		if name := constructor.ToObject(rt).Get("name"); name != nil {
			return name.String()
		}
	}
	return "Object"
}

func isTypedArray(rt *sobek.Runtime, value sobek.Value) bool {
	for _, name := range []string{"Int8Array", "Uint8Array", "Uint8ClampedArray", "Int16Array",
		"Uint16Array", "Int32Array", "Uint32Array", "Float32Array", "Float64Array",
		"BigInt64Array", "BigUint64Array"} {
		if ctor, ok := rt.Get(name).(*sobek.Object); ok && rt.InstanceOf(value, ctor) {
			return true
		}
	}
	return false
}

// isIndex reports whether the key is an array index, sobek exposes the typed
// array indices as own enumerable keys while node does not.
func isIndex(key string) bool {
	if key == "" {
		return false
	}
	for _, r := range key {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
