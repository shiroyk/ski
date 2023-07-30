// Package encoding the encoding JS implementation
package encoding

import (
	"encoding/base64"
	"strings"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/js"
	"github.com/shiroyk/cloudcat/plugin/jsmodule"
)

// Module js module
type Module struct{}

// Exports returns module instance
func (*Module) Exports() any {
	return map[string]any{
		"base64": &Base64{},
	}
}

func init() {
	jsmodule.Register("encoding", &Module{})
}

// Base64 encoding and decoding
type Base64 struct{}

// Encode returns the base64 encoding of input.
func (Base64) Encode(input any, skipPadding bool) (string, error) {
	data, err := js.ToBytes(input)
	if err != nil {
		return "", err
	}
	if skipPadding {
		return base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(data), nil
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// EncodeURI returns the base64URI encoding of input.
func (Base64) EncodeURI(input any) (string, error) {
	data, err := js.ToBytes(input)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(data), nil
}

// Decode returns the string decoding of input.
func (Base64) Decode(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	input := call.Argument(0).Export()
	toBuffer := call.Argument(1).ToBoolean()

	data, err := js.ToBytes(input)
	if err != nil {
		js.Throw(vm, err)
	}
	bytes, err := base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(unURI(string(data)))
	if err != nil {
		js.Throw(vm, err)
	}
	if toBuffer {
		return vm.ToValue(vm.NewArrayBuffer(bytes))
	}

	return vm.ToValue(string(bytes))
}

func unURI(input string) string {
	return strings.Map(func(r rune) rune {
		if r == '-' {
			return '+'
		}
		if r == '_' {
			return '/'
		}
		if (r >= 'A' && r <= 'Z') ||
			(r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') ||
			r == '+' || r == '/' {
			return r
		}
		return -1
	}, input)
}
