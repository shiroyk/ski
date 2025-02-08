package crypto

import (
	"testing"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
)

func TestCipherAlgorithm(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		return
	}

	vm := modulestest.New(t, js.WithInitial(func(rt *sobek.Runtime) {
		c := new(Crypto)
		instance, _ := c.Instantiate(rt)
		_ = rt.Set("crypto", instance)
	}))

	t.Run("Cipher", func(t *testing.T) {
		_, err := vm.Runtime().RunString(`{
			let key = "1111111111111111";
			let iv = "1111111111111111";
			let text = "hello aes";
			let aes = crypto.createCipher("AES/ECB/ZERO", key, iv);
			let result = aes.encrypt(text);
			let decrypt = aes.decrypt(result.binary()).string();
			assert.equal(text, decrypt);
			}`)
		assert.NoError(t, err)
	})

	t.Run("AES", func(t *testing.T) {
		mode := []string{"ECB", "CBC", "CFB", "OFB", "CTR", "GCM"}
		padding := []string{"ZERO", "PKCS5", "PKCS7"}
		for _, m := range mode {
			_ = vm.Runtime().Set("M", m)
			for _, p := range padding {
				_ = vm.Runtime().Set("P", p)
				t.Run(m+"/"+p, func(t *testing.T) {
					_, err := vm.Runtime().RunString(`{
					let key = "1111111111111111";
					let iv = "1111111111111111";
					let text = "hello aes";
					let aes = crypto.aes(key, iv, 'AES'+'/'+M+'/'+P);
					let result = aes.encrypt(text);
					let decrypt = aes.decrypt(result.binary()).string();
					assert.equal(text, decrypt);
					}`)
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
					_, err := vm.Runtime().RunString(`{
					let key = "11111111";
					let iv = "11111111";
					let text = "hello des";
					let des = crypto.des(key, iv, 'DES'+'/'+M+'/'+P);
					let result = des.encrypt(text);
					let decrypt = des.decrypt(result.binary()).string();
					assert.equal(text, decrypt);
					}`)
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
					_, err := vm.Runtime().RunString(`{
					let key = "111111111111111111111111";
					let text = "hello des";
					let des = crypto.tripleDes(key, null, 'TripleDes'+'/'+M+'/'+P);
					let result = des.encrypt(text);
					let decrypt = des.decrypt(result.binary()).string();
					assert.equal(text, decrypt);
					}`)
					assert.NoError(t, err)
				})
			}
		}
	})
}
