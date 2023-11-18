// Package crypto the crypto JS implementation
package crypto

import (
	"encoding/base64"
	"encoding/hex"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/plugin/jsmodule"
)

// Module js module
type Module struct{}

// Exports returns module instance
func (*Module) Exports() any {
	return map[string]any{
		"aes":          Aes,
		"createCipher": CreateCipher,
		"createHash":   CreateHash,
		"createHMAC":   CreateHMAC,
		"des":          Des,
		"hmac":         Hmac,
		"md4":          Md4,
		"md5":          Md5,
		"randomBytes":  RandomBytes,
		"ripemd160":    Ripemd160,
		"tripleDes":    TripleDES,
		"sha1":         Sha1,
		"sha256":       Sha256,
		"sha384":       Sha384,
		"sha512":       Sha512,
		"sha512_224":   Sha512_224,
		"sha512_256":   Sha512_256,
	}
}

func init() {
	jsmodule.Register("crypto", new(Module))
}

// Encoder the encoded
type Encoder struct {
	data []byte
}

// Base64 encode to base64
func (e *Encoder) Base64() string {
	return base64.StdEncoding.EncodeToString(e.data)
}

// Base64url encode to base64url
func (e *Encoder) Base64url() string {
	return base64.URLEncoding.EncodeToString(e.data)
}

// Base64rawurl encode to base64rawurl
func (e *Encoder) Base64rawurl() string {
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(e.data)
}

// Hex encode to hex
func (e *Encoder) Hex() string {
	return hex.EncodeToString(e.data)
}

// String encode to string
func (e *Encoder) String() string {
	return string(e.data)
}

// Binary encode to arraybuffer
func (e *Encoder) Binary(_ goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return vm.ToValue(vm.NewArrayBuffer(e.data))
}
