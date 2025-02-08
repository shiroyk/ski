package crypto

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"hash"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"golang.org/x/crypto/ripemd160"
)

// RandomBytes returns a random ArrayBuffer of the given size.
func RandomBytes(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	size := int(call.Argument(0).ToInteger())
	if size < 1 {
		js.Throw(rt, errors.New("invalid size"))
	}
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		js.Throw(rt, err)
	}
	return rt.ToValue(rt.NewArrayBuffer(bytes))
}

// Md5 returns the MD5 Hash of input in the given encoding.
func Md5(input any) (any, error) { return Hash("md5", input) }

// Sha1 returns the SHA1 Hash of input in the given encoding.
func Sha1(input any) (any, error) { return Hash("sha1", input) }

// Sha256 returns the SHA256 Hash of input in the given encoding.
func Sha256(input any) (any, error) { return Hash("sha256", input) }

// Sha384 returns the SHA384 Hash of input in the given encoding.
func Sha384(input any) (any, error) { return Hash("sha384", input) }

// Sha512 returns the SHA512 Hash of input in the given encoding.
func Sha512(input any) (any, error) { return Hash("sha512", input) }

// Sha512_224 returns the SHA512/224 Hash of input in the given encoding.
func Sha512_224(input any) (any, error) { return Hash("sha512_224", input) }

// Sha512_256 returns the SHA512/256 Hash of input in the given encoding.
func Sha512_256(input any) (any, error) { return Hash("sha512_256", input) }

// Ripemd160 returns the RIPEMD160 Hash of input in the given encoding.
func Ripemd160(input any) (any, error) { return Hash("ripemd160", input) }

// CreateHash returns a Hasher instance that uses the given algorithm.
func CreateHash(algorithm string) (*Hasher, error) {
	h := hashFunc(algorithm)
	if h == nil {
		return nil, fmt.Errorf("invalid algorithm: %s", algorithm)
	}
	return &Hasher{h()}, nil
}

// Hash returns a new Encoder using the given algorithm and key.
func Hash(algorithm string, input any) (*Encoder, error) {
	hasher, err := CreateHash(algorithm)
	if err != nil {
		return nil, err
	}
	return hasher.Encrypt(input)
}

// CreateHMAC returns a new HMAC Hash using the given algorithm and key.
func CreateHMAC(algorithm string, key any) (*Hasher, error) {
	h := hashFunc(algorithm)
	if h == nil {
		return nil, fmt.Errorf("invalid algorithm: %s", algorithm)
	}

	data, err := js.ToBytes(key)
	if err != nil {
		return nil, err
	}
	return &Hasher{hmac.New(h, data)}, nil
}

// Hmac returns a new Encoder of input using the given algorithm and key.
func Hmac(algorithm string, key, input any) (*Encoder, error) {
	hasher, err := CreateHMAC(algorithm, key)
	if err != nil {
		return nil, err
	}
	return hasher.Encrypt(input)
}

func hashFunc(name string) func() hash.Hash {
	switch name {
	case "md5":
		return md5.New
	case "sha1":
		return sha1.New
	case "sha256":
		return sha256.New
	case "sha384":
		return sha512.New384
	case "sha512":
		return sha512.New
	case "sha512_224":
		return sha512.New512_224
	case "sha512_256":
		return sha512.New512_256
	case "ripemd160":
		return ripemd160.New
	default:
		return nil
	}
}

// Hasher wraps a hash.Hash.
type Hasher struct {
	hash hash.Hash
}

// Update the Hash with the input data.
func (hasher *Hasher) Update(input any) error {
	d, err := js.ToBytes(input)
	if err != nil {
		return err
	}
	_, err = hasher.hash.Write(d)
	if err != nil {
		return err
	}
	return nil
}

// Digest returns the Hash value in the given encoding.
func (hasher *Hasher) Digest() *Encoder {
	return &Encoder{hasher.hash.Sum(nil)}
}

// Reset resets the Hash value to initial values.
func (hasher *Hasher) Reset() {
	hasher.hash.Reset()
}

// Encrypt returns the Encoder.
func (hasher *Hasher) Encrypt(input any) (*Encoder, error) {
	if err := hasher.Update(input); err != nil {
		return nil, err
	}
	return hasher.Digest(), nil
}
