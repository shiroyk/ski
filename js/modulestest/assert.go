package modulestest

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/types"
)

// throwMatch is match error message
var throwMatch bool

// Assert returns the assert implementation used by the module tests, it is a
// callable object like the node `assert` module with the node methods, the go
// side helpers (equal, contains, regexp and true) are attached as well.
func Assert(rt *sobek.Runtime) *sobek.Object {
	errorCtor := rt.Get("Error").ToObject(rt)
	errorPrototype := errorCtor.Get("prototype").ToObject(rt)

	assertionError := rt.ToValue(func(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
		object, err := rt.New(errorCtor, call.Argument(0))
		if err != nil {
			panic(err)
		}
		_ = object.Set("name", "AssertionError")
		_ = object.Set("code", "ERR_ASSERTION")
		_ = object.Set("actual", call.Argument(1))
		_ = object.Set("expected", call.Argument(2))
		_ = object.Set("operator", call.Argument(3))
		_ = object.Set("generatedMessage", true)
		_ = object.SetPrototype(call.This.Prototype())
		return object
	}).ToObject(rt)
	prototype := rt.NewObject()
	_ = prototype.SetPrototype(errorPrototype)
	_ = prototype.Set("name", "AssertionError")
	_ = prototype.Set("code", "ERR_ASSERTION")
	_ = prototype.Set("constructor", assertionError)
	_ = assertionError.Set("prototype", prototype)
	_ = assertionError.SetPrototype(errorCtor)

	// fail throws an AssertionError, it is the panic value so the runtime
	// throws a javascript error object
	fail := func(text string, actual, expected sobek.Value, operator string) {
		object, err := rt.New(assertionError, rt.ToValue(text), actual, expected, rt.ToValue(operator))
		if err != nil {
			panic(err)
		}
		panic(object)
	}

	failure := func(call sobek.FunctionCall, operator string, actual, expected sobek.Value) {
		text := fmt.Sprintf("%s failed\nactual: %s\nexpected: %s",
			operator, InspectValue(actual), InspectValue(expected))
		if msg := call.Argument(2); !sobek.IsUndefined(msg) {
			text = msg.String() + ": " + text
		}
		fail(text, actual, expected, operator)
	}

	deepStrictEqual := func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
		actual, expected := call.Argument(0), call.Argument(1)
		if !DeepEqual(rt, actual, expected, true, make(map[*sobek.Object]map[*sobek.Object]bool)) {
			failure(call, "deepStrictEqual", actual, expected)
		}
		return sobek.Undefined()
	}

	deepLooseEqual := func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
		actual, expected := call.Argument(0), call.Argument(1)
		if !DeepEqual(rt, actual, expected, false, make(map[*sobek.Object]map[*sobek.Object]bool)) {
			failure(call, "deepEqual", actual, expected)
		}
		return sobek.Undefined()
	}

	notDeepStrictEqual := func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
		actual, expected := call.Argument(0), call.Argument(1)
		if DeepEqual(rt, actual, expected, true, make(map[*sobek.Object]map[*sobek.Object]bool)) {
			failure(call, "notDeepStrictEqual", actual, expected)
		}
		return sobek.Undefined()
	}

	strictEqual := func(call sobek.FunctionCall, _ *sobek.Runtime) sobek.Value {
		actual, expected := call.Argument(0), call.Argument(1)
		if !actual.SameAs(expected) {
			failure(call, "strictEqual", actual, expected)
		}
		return sobek.Undefined()
	}

	notStrictEqual := func(call sobek.FunctionCall, _ *sobek.Runtime) sobek.Value {
		actual, expected := call.Argument(0), call.Argument(1)
		if actual.SameAs(expected) {
			failure(call, "notStrictEqual", actual, expected)
		}
		return sobek.Undefined()
	}

	throws := func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
		function, ok := sobek.AssertFunction(call.Argument(0))
		if !ok {
			fail(`The "fn" argument must be of type function`, call.Argument(0), sobek.Undefined(), "throws")
		}
		_, err := function(sobek.Undefined())
		if err == nil {
			text := "Missing expected exception"
			if msg := call.Argument(2); !sobek.IsUndefined(msg) {
				text = msg.String()
			}
			fail(text, call.Argument(1), sobek.Undefined(), "throws")
		}
		actual := thrownValue(rt, err)
		checkThrown(rt, actual, call.Argument(1), fail)
		return actual
	}

	doesNotThrow := func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
		function, ok := sobek.AssertFunction(call.Argument(0))
		if !ok {
			fail(`The "fn" argument must be of type function`, call.Argument(0), sobek.Undefined(), "doesNotThrow")
		}
		if _, err := function(sobek.Undefined()); err != nil {
			text := "Got unwanted exception"
			if msg := call.Argument(1); !sobek.IsUndefined(msg) {
				text = msg.String()
			}
			fail(text, thrownValue(rt, err), sobek.Undefined(), "doesNotThrow")
		}
		return sobek.Undefined()
	}

	assertFunc := rt.ToValue(func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
		if !call.Argument(0).ToBoolean() {
			text := fmt.Sprintf("%s == true", InspectValue(call.Argument(0)))
			if msg := call.Argument(1); !sobek.IsUndefined(msg) {
				text = msg.String()
			}
			fail(text, call.Argument(0), rt.ToValue(true), "==")
		}
		return sobek.Undefined()
	}).ToObject(rt)

	_ = assertFunc.Set("AssertionError", assertionError)
	_ = assertFunc.Set("ok", assertFunc)
	_ = assertFunc.Set("fail", func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
		text := "Failed"
		if msg := call.Argument(0); !sobek.IsUndefined(msg) {
			text = msg.String()
		}
		fail(text, sobek.Undefined(), sobek.Undefined(), "fail")
		return sobek.Undefined()
	})
	_ = assertFunc.Set("ifError", func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
		value := call.Argument(0)
		if value != nil && !sobek.IsUndefined(value) && !sobek.IsNull(value) {
			fail("ifError got unwanted exception", value, sobek.Undefined(), "ifError")
		}
		return sobek.Undefined()
	})
	_ = assertFunc.Set("strictEqual", strictEqual)
	_ = assertFunc.Set("notStrictEqual", notStrictEqual)
	_ = assertFunc.Set("and", strictEqual)
	_ = assertFunc.Set("deepStrictEqual", deepStrictEqual)
	_ = assertFunc.Set("notDeepStrictEqual", notDeepStrictEqual)
	_ = assertFunc.Set("deepEqual", deepLooseEqual)
	_ = assertFunc.Set("notDeepEqual", notDeepStrictEqual)
	_ = assertFunc.Set("throws", throws)
	_ = assertFunc.Set("doesNotThrow", doesNotThrow)
	_ = assertFunc.Set("match", func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
		text := call.Argument(0).String()
		regexp := call.Argument(1)
		test, ok := sobek.AssertFunction(regexp.ToObject(rt).Get("test"))
		if !ok {
			fail(`The "regexp" argument must be an instance of RegExp`, regexp, sobek.Undefined(), "match")
		}
		result, err := test(regexp, rt.ToValue(text))
		if err != nil {
			js.Throw(rt, err)
		}
		if !result.ToBoolean() {
			failure(call, "match", call.Argument(0), call.Argument(1))
		}
		return sobek.Undefined()
	})
	// node keeps the strict methods in the assert.strict namespace
	_ = assertFunc.Set("strict", assertFunc)
	_ = assertFunc.Set("equal", deepLooseEqual)
	_ = assertFunc.Set("true", assertFunc)
	_ = assertFunc.Set("contains", func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
		text := call.Argument(0).String()
		sub := call.Argument(1).String()
		if !strings.Contains(text, sub) {
			failure(call, "contains", call.Argument(0), call.Argument(1))
		}
		return sobek.Undefined()
	})
	return assertFunc
}

// thrownValue returns the value thrown by the function.
func thrownValue(rt *sobek.Runtime, err error) sobek.Value {
	if exception, ok := errors.AsType[*sobek.Exception](err); ok {
		return exception.Value()
	}
	return rt.NewGoError(err)
}

// checkThrown validates the thrown value against the expected value, the
// expected value follows the node assert.throws() rules.
func checkThrown(rt *sobek.Runtime, actual, expected sobek.Value, fail func(string, sobek.Value, sobek.Value, string)) {
	if expected == nil || sobek.IsUndefined(expected) {
		return
	}

	if object, ok := expected.(*sobek.Object); ok {
		if prototype := object.Get("prototype"); prototype != nil && !sobek.IsUndefined(prototype) {
			if rt.InstanceOf(actual, object) {
				return
			}
		}
		if function, ok := sobek.AssertFunction(object); ok {
			if isPrototypeOf(rt.Get("Error").ToObject(rt), object) {
				fail(fmt.Sprintf("The error class is not a subclass of Error: %s", object.String()),
					actual, expected, "throws")
			}
			result, err := function(rt.NewObject(), actual)
			if err != nil {
				js.Throw(rt, err)
			}
			if result.ToBoolean() {
				return
			}
			fail(`The validation function is expected to return "true"`, actual, expected, "throws")
		}
		if rt.InstanceOf(object, rt.Get("RegExp").ToObject(rt)) {
			test, ok := sobek.AssertFunction(object.Get("test"))
			if ok {
				result, err := test(object, rt.ToValue(actual.String()))
				if err != nil {
					js.Throw(rt, err)
				}
				if result.ToBoolean() {
					return
				}
			}
			fail(fmt.Sprintf(`The error message "%s" does not match %s`, actual.String(), object.String()),
				actual, expected, "throws")
		}
		if throwMatch {
			for _, key := range object.Keys() {
				validated := errorProperty(actual, key)
				expected := object.Get(key)
				// a regexp value matches the property, like node's assert.throws()
				if regexp, ok := expected.(*sobek.Object); ok && rt.InstanceOf(regexp, rt.Get("RegExp").ToObject(rt)) {
					test, ok := sobek.AssertFunction(regexp.Get("test"))
					if ok {
						result, err := test(regexp, rt.ToValue(validated.String()))
						if err != nil {
							js.Throw(rt, err)
						}
						if result.ToBoolean() {
							continue
						}
					}
				}
				if !DeepEqual(rt, validated, expected, true, make(map[*sobek.Object]map[*sobek.Object]bool)) {
					fail(fmt.Sprintf(`The error "%s" property does not match: %s != %s`,
						key, InspectValue(validated), InspectValue(expected)),
						validated, expected, "throws")
				}
			}
		}
		return
	}

	fail("The error type is not supported", actual, expected, "throws")
}

// errorProperty returns the property of the thrown value, undefined when the
// value is not an object.
func errorProperty(value sobek.Value, key string) sobek.Value {
	if object, ok := value.(*sobek.Object); ok {
		return object.Get(key)
	}
	return sobek.Undefined()
}

// isPrototypeOf reports whether the prototype is in the prototype chain of the
// object, it is the javascript Object.prototype.isPrototypeOf().
func isPrototypeOf(prototype, object *sobek.Object) bool {
	for current := object.Prototype(); current != nil; current = current.Prototype() {
		if current == prototype {
			return true
		}
	}
	return false
}

// DeepEqual is the node deepStrictEqual/deepEqual comparison, strict compares
// the prototype chain and the typed array constructors.
func DeepEqual(rt *sobek.Runtime, a, b sobek.Value, strict bool, seen map[*sobek.Object]map[*sobek.Object]bool) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if a.SameAs(b) {
		return true
	}
	if types.ValueType(a) != types.ValueType(b) {
		if !strict {
			return a.Equals(b)
		}
		return false
	}
	if types.ValueType(a) != "object" && types.ValueType(a) != "function" {
		if !strict {
			return a.Equals(b)
		}
		return false
	}

	aObject, aOk := a.(*sobek.Object)
	bObject, bOk := b.(*sobek.Object)
	if !aOk || !bOk {
		return false
	}
	if strict && aObject.Prototype() != bObject.Prototype() {
		return false
	}
	if seen[aObject] != nil && seen[aObject][bObject] {
		return true
	}
	if seen[aObject] == nil {
		seen[aObject] = make(map[*sobek.Object]bool)
	}
	seen[aObject][bObject] = true

	if types.IsTypedArray(rt, a) || types.IsTypedArray(rt, b) {
		if !types.IsTypedArray(rt, a) || !types.IsTypedArray(rt, b) {
			return false
		}
		if strict && aObject.Get("constructor") != bObject.Get("constructor") {
			return false
		}
		return typedArrayEqual(aObject, bObject)
	}

	if aObject.ClassName() == "Date" || bObject.ClassName() == "Date" {
		return aObject.String() == bObject.String()
	}
	if rt.InstanceOf(a, rt.Get("Error").ToObject(rt)) || rt.InstanceOf(b, rt.Get("Error").ToObject(rt)) {
		return rt.InstanceOf(a, rt.Get("Error").ToObject(rt)) &&
			rt.InstanceOf(b, rt.Get("Error").ToObject(rt)) &&
			aObject.Get("name").String() == bObject.Get("name").String() &&
			aObject.Get("message").String() == bObject.Get("message").String()
	}

	if aObject.ClassName() == "Array" || bObject.ClassName() == "Array" {
		if aObject.ClassName() != "Array" || bObject.ClassName() != "Array" {
			return false
		}
		if aObject.Get("length").ToInteger() != bObject.Get("length").ToInteger() {
			return false
		}
	}

	aKeys, bKeys := aObject.Keys(), bObject.Keys()
	if len(aKeys) != len(bKeys) {
		return false
	}
	for _, key := range aKeys {
		if !slices.Contains(bKeys, key) {
			return false
		}
		if !DeepEqual(rt, aObject.Get(key), bObject.Get(key), strict, seen) {
			return false
		}
	}
	aSymbols, bSymbols := aObject.Symbols(), bObject.Symbols()
	if len(aSymbols) != len(bSymbols) {
		return false
	}
	for _, symbol := range aSymbols {
		var found *sobek.Symbol
		for _, item := range bSymbols {
			if item == symbol {
				found = item
				break
			}
		}
		if found == nil {
			return false
		}
		if !DeepEqual(rt, aObject.GetSymbol(symbol), bObject.GetSymbol(found), strict, seen) {
			return false
		}
	}
	return true
}

func typedArrayEqual(a, b *sobek.Object) bool {
	length := a.Get("length").ToInteger()
	if length != b.Get("length").ToInteger() {
		return false
	}
	for i := range length {
		key := strconv.FormatInt(i, 10)
		if !a.Get(key).SameAs(b.Get(key)) {
			return false
		}
	}
	return true
}

// InspectValue returns a short description of the value for the assertion
// messages.
func InspectValue(value sobek.Value) string {
	if value == nil || sobek.IsUndefined(value) {
		return "undefined"
	}
	if sobek.IsNull(value) {
		return "null"
	}
	switch {
	case types.IsString(value):
		return "'" + value.String() + "'"
	case types.IsNumber(value), types.IsBigInt(value), value.ExportType() == types.TypeBool:
		return value.String()
	}
	if object, ok := value.(*sobek.Object); ok && isTypedArrayObject(object) {
		length := object.Get("length").ToInteger()
		items := make([]string, 0, length)
		for i := int64(0); i < length && i < 16; i++ {
			items = append(items, object.Get(strconv.FormatInt(i, 10)).String())
		}
		ending := ""
		if length > 16 {
			ending = ", ..."
		}
		return fmt.Sprintf("%s(%d) [%s%s]", object.ClassName(), length, strings.Join(items, ", "), ending)
	}
	return value.String()
}

func isTypedArrayObject(object *sobek.Object) bool {
	switch object.ClassName() {
	case "Int8Array", "Uint8Array", "Uint8ClampedArray", "Int16Array", "Uint16Array",
		"Int32Array", "Uint32Array", "Float32Array", "Float64Array", "BigInt64Array",
		"BigUint64Array":
		return true
	}
	return false
}
