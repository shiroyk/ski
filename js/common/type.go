package common

import (
	"reflect"
	"strings"
)

// Program The js program
type Program struct {
	Code string
	Args map[string]any
}

// FieldNameMapper provides custom mapping between Go and JavaScript property names.
type FieldNameMapper struct{}

// FieldName returns a JavaScript name for the given struct field in the given type.
// If this method returns "" the field becomes hidden.
func (FieldNameMapper) FieldName(_ reflect.Type, f reflect.StructField) string {
	if v, ok := f.Tag.Lookup("js"); ok {
		if v == "-" {
			return ""
		}
		return v
	}
	return strings.ToLower(f.Name[0:1]) + f.Name[1:]
}

// MethodName returns a JavaScript name for the given method in the given type.
// If this method returns "" the method becomes hidden.
func (FieldNameMapper) MethodName(_ reflect.Type, m reflect.Method) string {
	return strings.ToLower(m.Name[0:1]) + m.Name[1:]
}
