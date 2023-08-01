package crypto

import (
	"context"
	"testing"

	"github.com/shiroyk/cloudcat/js/modulestest"
	"github.com/stretchr/testify/assert"
)

func TestCipherAlgorithm(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		return
	}

	vm := modulestest.New(t)
	_, _ = vm.Runtime().RunString(`
		const crypto = require('cloudcat/crypto');
	`)

	t.Run("AES", func(t *testing.T) {
		mode := []string{"ECB", "CBC", "CFB", "OFB", "CTR", "GCM"}
		padding := []string{"ZERO", "PKCS5", "PKCS7"}
		for _, m := range mode {
			_ = vm.Runtime().Set("M", m)
			for _, p := range padding {
				_ = vm.Runtime().Set("P", p)
				t.Run(m+"/"+p, func(t *testing.T) {
					_, err := vm.RunString(context.Background(), `
					var key = "1111111111111111";
					var iv = "1111111111111111";
					var text = "hello aes";
					var aes = crypto.aes(key, iv, 'AES'+'/'+M+'/'+P);
					var result = aes.encrypt(text);
					var decrypt = aes.decrypt(result.binary()).string();
					assert.equal(text, decrypt);
					`)
					assert.NoError(t, err)
				})
			}
		}
	})

	t.Run("DES", func(t *testing.T) {
		mode := []string{"ECB", "CBC", "CFB", "OFB", "CTR"}
		padding := []string{"ZERO", "PKCS5", "PKCS7"}
		for _, m := range mode {
			_ = vm.Runtime().Set("M", m)
			for _, p := range padding {
				_ = vm.Runtime().Set("P", p)
				t.Run(m+"/"+p, func(t *testing.T) {
					_, err := vm.RunString(context.Background(), `
					var key = "11111111";
					var iv = "11111111";
					var text = "hello des";
					var des = crypto.des(key, iv, 'DES'+'/'+M+'/'+P);
					var result = des.encrypt(text);
					var decrypt = des.decrypt(result.binary()).string();
					assert.equal(text, decrypt);
					`)
					assert.NoError(t, err)
				})
			}
		}
	})

	t.Run("TripleDES", func(t *testing.T) {
		mode := []string{"ECB", "CBC", "CFB", "OFB", "CTR", "GCM"}
		padding := []string{"ZERO", "PKCS5", "PKCS7"}
		for _, m := range mode {
			_ = vm.Runtime().Set("M", m)
			for _, p := range padding {
				_ = vm.Runtime().Set("P", p)
				t.Run(m+"/"+p, func(t *testing.T) {
					_, err := vm.RunString(context.Background(), `
					var key = "111111111111111111111111";
					var text = "hello des";
					var des = crypto.tripleDes(key, null, 'TripleDes'+'/'+M+'/'+P);
					var result = des.encrypt(text);
					var decrypt = des.decrypt(result.binary()).string();
					assert.equal(text, decrypt);
					`)
					assert.NoError(t, err)
				})
			}
		}
	})
}
