// Package crypto the crypto JS implementation
package crypto

import (
	"encoding/base64"
	"encoding/hex"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/modules"
)

func init() {
	modules.Register("crypto", new(Crypto))
}

// Crypto js module
type Crypto struct{}

// Instantiate returns module instance
func (*Crypto) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	return rt.ToValue(map[string]any{
		"aes":          Aes,
		"createCipher": CreateCipher,
		"createHash":   CreateHash,
		"createHMAC":   CreateHMAC,
		"des":          Des,
		"hmac":         Hmac,
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
	}), nil
}

// Encoder the encoded
type Encoder struct{ data []byte }

// Base64 encode to base64
func (e *Encoder) Base64() string { return base64.StdEncoding.EncodeToString(e.data) }

// Hex encode to hex
func (e *Encoder) Hex() string { return hex.EncodeToString(e.data) }

// String encode to string
func (e *Encoder) String() string { return string(e.data) }

// Binary encode to arraybuffer
func (e *Encoder) Binary(_ sobek.FunctionCall, vm *sobek.Runtime) sobek.Value {
	return vm.ToValue(vm.NewArrayBuffer(e.data))
}
