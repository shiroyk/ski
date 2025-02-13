package base64

import (
	"encoding/base64"
	"strings"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/modules"
)

func init() {
	modules.Register("base64", new(Base64))
}

// Base64 encoding and decoding
type Base64 struct{}

func (b Base64) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	ret := rt.NewObject()
	_ = ret.Set("encode", b.Encode)
	_ = ret.Set("encodeURI", b.EncodeURI)
	_ = ret.Set("decode", b.Decode)
	return ret, nil
}

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
func (Base64) Decode(call sobek.FunctionCall, rt *sobek.Runtime) (ret sobek.Value) {
	input := call.Argument(0).Export()
	toBuffer := call.Argument(1).ToBoolean()

	data, err := js.ToBytes(input)
	if err != nil {
		js.Throw(rt, err)
	}
	bytes, err := base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(unURI(string(data)))
	if err != nil {
		js.Throw(rt, err)
	}
	if toBuffer {
		return rt.ToValue(rt.NewArrayBuffer(bytes))
	}

	return rt.ToValue(string(bytes))
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
