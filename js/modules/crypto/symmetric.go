package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/shiroyk/cloudcat/js/common"
)

// Aes returns a new AES cipher
func Aes(key, iv any, algorithm string) (*Cipher, error) {
	if algorithm == "" || !strings.HasPrefix(algorithm, "AES") {
		algorithm = "AES/ECB/PKCS5"
	}
	c, err := CreateCipher(algorithm, key, iv)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// Des returns a new DES cipher
func Des(key, iv any, algorithm string) (*Cipher, error) {
	if algorithm == "" || !strings.HasPrefix(algorithm, "DES") {
		algorithm = "DES/CBC/PKCS5"
	}
	c, err := CreateCipher(algorithm, key, iv)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// TripleDES returns a new TDes cipher
func TripleDES(key, iv any, algorithm string) (*Cipher, error) {
	if algorithm == "" || !strings.HasPrefix(algorithm, "TripleDES") {
		algorithm = "TripleDES/CBC/PKCS5"
	}
	c, err := CreateCipher(algorithm, key, iv)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// CreateCipher returns a new Cipher
func CreateCipher(algorithm string, key, iv any) (*Cipher, error) {
	keyByte, err := common.ToBytes(key)
	if err != nil {
		return nil, err
	}

	symmetric := strings.Split(algorithm, "/")
	if len(symmetric) != 3 {
		return nil, fmt.Errorf("invalid algorithm: %s", algorithm)
	}

	var block cipher.Block

	switch symmetric[0] {
	case "AES":
		block, err = aes.NewCipher(keyByte)
	case "DES":
		block, err = des.NewCipher(keyByte)
	case "TripleDES":
		block, err = des.NewTripleDESCipher(keyByte)
	default:
		return nil, fmt.Errorf("invalid algorithm: %s", algorithm)
	}
	if err != nil {
		return nil, err
	}
	ivByte, _ := common.ToBytes(iv)
	if ivByte == nil {
		ivByte = make([]byte, block.BlockSize())
		if _, err = rand.Read(ivByte); err != nil {
			return nil, err
		}
	}

	var padding Padding = none
	switch symmetric[2] {
	case "ZERO":
		padding = zero
	case "PKCS5":
		padding = pkcs5
	case "PKCS7":
		padding = pkcs7
	}

	return &Cipher{ivByte, block, symmetric[1], padding}, nil
}

// Cipher A Block represents an implementation of block cipher
// using a given key. It provides the capability to encrypt
// or decrypt individual blocks.
type Cipher struct {
	iv      []byte
	block   cipher.Block
	mode    string
	padding Padding
}

// Encrypt encrypts the first block in src into dst.
func (c *Cipher) Encrypt(input any) (*Encoder, error) {
	src, err := common.ToBytes(input)
	if err != nil {
		return nil, err
	}
	blockSize := c.block.BlockSize()
	src = c.padding.Padding(src, blockSize)
	dst := make([]byte, len(src))
	switch c.mode {
	case "ECB":
		dstTemp := dst
		for len(src) > 0 {
			c.block.Encrypt(dstTemp, src[:blockSize])
			src = src[blockSize:]
			dstTemp = dstTemp[blockSize:]
		}
	case "CBC":
		cipher.NewCBCEncrypter(c.block, c.iv).CryptBlocks(dst, src)
	case "CFB":
		cipher.NewCFBEncrypter(c.block, c.iv).XORKeyStream(dst, src)
	case "OFB":
		cipher.NewOFB(c.block, c.iv).XORKeyStream(dst, src)
	case "CTR":
		cipher.NewCTR(c.block, c.iv).XORKeyStream(dst, src)
	case "GCM":
		gcm, err := cipher.NewGCMWithNonceSize(c.block, len(c.iv))
		if err != nil {
			return nil, err
		}
		dst = gcm.Seal(c.iv, c.iv, src, nil)
	default:
		return nil, fmt.Errorf("unsupported encryption mode %s", c.mode)
	}
	return &Encoder{dst}, nil
}

// Decrypt decrypts the first block in src into dst.
func (c *Cipher) Decrypt(input any) (*Encoder, error) {
	src, err := common.ToBytes(input)
	if err != nil {
		return nil, err
	}
	blockSize := c.block.BlockSize()
	dst := make([]byte, len(src))
	switch c.mode {
	case "ECB":
		dstTemp := dst
		for len(src) > 0 {
			c.block.Decrypt(dstTemp, src[:blockSize])
			src = src[blockSize:]
			dstTemp = dstTemp[blockSize:]
		}
	case "CBC":
		cipher.NewCBCDecrypter(c.block, c.iv).CryptBlocks(dst, src)
	case "CFB":
		cipher.NewCFBDecrypter(c.block, c.iv).XORKeyStream(dst, src)
	case "OFB":
		cipher.NewOFB(c.block, c.iv).XORKeyStream(dst, src)
	case "CTR":
		cipher.NewCTR(c.block, c.iv).XORKeyStream(dst, src)
	case "GCM":
		gcm, err := cipher.NewGCMWithNonceSize(c.block, len(c.iv))
		if err != nil {
			return nil, err
		}
		var nonce []byte
		nonce, src = src[:gcm.NonceSize()], src[gcm.NonceSize():]
		dst, err = gcm.Open(nil, nonce, src, nil)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported decryption mode %s", c.mode)
	}
	return &Encoder{c.padding.UnPadding(dst)}, nil
}

// Padding the blocks size
type Padding interface {
	// Padding the blocks with bytes
	Padding([]byte, int) []byte
	// UnPadding the data
	UnPadding([]byte) []byte
}

var (
	pkcs7 PKCS7
	pkcs5 PKCS5
	zero  Zero
	none  None
)

// None padding the blocks size
type None struct{}

// Padding the blocks with bytes
func (None) Padding(data []byte, _ int) []byte {
	return data
}

// UnPadding the data
func (None) UnPadding(data []byte) []byte {
	return data
}

// PKCS7 padding the blocks size
type PKCS7 struct{}

// Padding the blocks with bytes
func (PKCS7) Padding(data []byte, blockSize int) []byte {
	size := blockSize - len(data)%blockSize
	paddingText := bytes.Repeat([]byte{byte(size)}, size)
	return append(data, paddingText...)
}

// UnPadding the data
func (PKCS7) UnPadding(data []byte) []byte {
	length := len(data)
	unPadding := int(data[length-1])
	return data[:(length - unPadding)]
}

// PKCS5 padding the blocks size
type PKCS5 struct{}

// Padding the blocks with bytes
func (PKCS5) Padding(data []byte, blockSize int) []byte {
	return pkcs7.Padding(data, 8)
}

// UnPadding the data
func (PKCS5) UnPadding(data []byte) []byte {
	return pkcs7.UnPadding(data)
}

// Zero padding the blocks size
type Zero struct{}

// Padding the blocks with bytes
func (Zero) Padding(data []byte, blockSize int) []byte {
	size := blockSize - len(data)%blockSize
	paddingText := bytes.Repeat([]byte{byte(0)}, size)
	return append(data, paddingText...)
}

// UnPadding the data
func (Zero) UnPadding(data []byte) []byte {
	return bytes.TrimRightFunc(data, func(r rune) bool {
		return r == 0
	})
}
