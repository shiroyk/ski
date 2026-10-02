package common

import (
	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js/types"
)

var symDOMException = sobek.NewSymbol("Symbol.DOMException")

// DOMException is the JavaScript `DOMException` constructor.
// https://webidl.spec.whatwg.org/#idl-DOMException
type DOMException struct{}

func (e *DOMException) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	proto := e.prototype(rt)
	ctor := rt.ToValue(e.constructor).ToObject(rt)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.SetPrototype(proto)
	_ = ctor.Set("prototype", proto)
	_ = proto.SetPrototype(types.Prototype(rt, "Error"))
	_ = proto.SetSymbol(sobek.SymToStringTag, rt.ToValue("DOMException"))
	setDOMExceptionConstants(ctor)
	setDOMExceptionConstants(proto)
	return ctor, nil
}

func (e *DOMException) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	var message string
	if v := call.Argument(0); !types.IsNil(v) {
		message = v.String()
	}
	name := "Error"
	if v := call.Argument(1); !types.IsNil(v) {
		name = v.String()
	}
	// Keep JavaScript subclassing working.
	return newDOMException(rt, message, name, call.This.ToObject(rt).Prototype())
}

func (e *DOMException) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()
	_ = p.DefineAccessorProperty("name", rt.ToValue(e.name), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("message", rt.ToValue(e.message), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("code", rt.ToValue(e.code), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	return p
}

func (*DOMException) name(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toDOMException(rt, call.This).name)
}

func (*DOMException) message(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toDOMException(rt, call.This).message)
}

func (*DOMException) code(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(domExceptionCode(toDOMException(rt, call.This).name))
}

// NewDOMException creates a new DOMException with the given message and name.
func NewDOMException(rt *sobek.Runtime, message, name string) *sobek.Object {
	if name == "" {
		name = "Error"
	}
	return newDOMException(rt, message, name, types.Prototype(rt, "DOMException"))
}

type _domException struct {
	message string
	name    string
}

// newDOMException creates a DOMException, a plain object so that its instances
// stay extensible like every other JavaScript object.
func newDOMException(rt *sobek.Runtime, message, name string, proto *sobek.Object) *sobek.Object {
	obj := rt.NewObject()
	_ = obj.SetPrototype(proto)
	_ = obj.SetSymbol(symDOMException, &_domException{message: message, name: name})
	_ = obj.Set("stack", name+": "+message)
	return obj
}

func toDOMException(rt *sobek.Runtime, value sobek.Value) *_domException {
	if o, ok := value.(*sobek.Object); ok {
		if brand := o.GetSymbol(symDOMException); brand != nil {
			if exc, ok := brand.Export().(*_domException); ok {
				return exc
			}
		}
	}
	panic(NewTypeErrorf(rt, ErrInvalidThis, `Value of "this" must be of type DOMException`))
}

// domExceptionCodes maps DOMException names to their legacy codes.
var domExceptionCodes = map[string]int{
	"IndexSizeError":             1,
	"DOMStringSizeError":         2,
	"HierarchyRequestError":      3,
	"WrongDocumentError":         4,
	"InvalidCharacterError":      5,
	"NoDataAllowedError":         6,
	"NoModificationAllowedError": 7,
	"NotFoundError":              8,
	"NotSupportedError":          9,
	"InUseAttributeError":        10,
	"InvalidStateError":          11,
	"SyntaxError":                12,
	"InvalidModificationError":   13,
	"NamespaceError":             14,
	"InvalidAccessError":         15,
	"ValidationError":            16,
	"TypeMismatchError":          17,
	"SecurityError":              18,
	"NetworkError":               19,
	"AbortError":                 20,
	"URLMismatchError":           21,
	"QuotaExceededError":         22,
	"TimeoutError":               23,
	"InvalidNodeTypeError":       24,
	"DataCloneError":             25,
}

func domExceptionCode(name string) int {
	return domExceptionCodes[name]
}

// domExceptionConstants are the legacy DOMException codes.
var domExceptionConstants = map[string]int{
	"INDEX_SIZE_ERR":              1,
	"DOMSTRING_SIZE_ERR":          2,
	"HIERARCHY_REQUEST_ERR":       3,
	"WRONG_DOCUMENT_ERR":          4,
	"INVALID_CHARACTER_ERR":       5,
	"NO_DATA_ALLOWED_ERR":         6,
	"NO_MODIFICATION_ALLOWED_ERR": 7,
	"NOT_FOUND_ERR":               8,
	"NOT_SUPPORTED_ERR":           9,
	"INUSE_ATTRIBUTE_ERR":         10,
	"INVALID_STATE_ERR":           11,
	"SYNTAX_ERR":                  12,
	"INVALID_MODIFICATION_ERR":    13,
	"NAMESPACE_ERR":               14,
	"INVALID_ACCESS_ERR":          15,
	"VALIDATION_ERR":              16,
	"TYPE_MISMATCH_ERR":           17,
	"SECURITY_ERR":                18,
	"NETWORK_ERR":                 19,
	"ABORT_ERR":                   20,
	"URL_MISMATCH_ERR":            21,
	"QUOTA_EXCEEDED_ERR":          22,
	"TIMEOUT_ERR":                 23,
	"INVALID_NODE_TYPE_ERR":       24,
	"DATA_CLONE_ERR":              25,
}

func setDOMExceptionConstants(o *sobek.Object) {
	for name, code := range domExceptionConstants {
		_ = o.Set(name, code)
	}
}
