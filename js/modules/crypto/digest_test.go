package crypto

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
)

func TestHashAlgorithms(t *testing.T) {
	vm := modulestest.New(t, js.WithInitial(func(rt *goja.Runtime) {
		c := new(Crypto)
		instance, _ := c.Instantiate(rt)
		_ = rt.Set("crypto", instance)
	}))

	testCases := []struct {
		algorithm, origin, want string
	}{
		{"md5", "hello md5", "741fc6b1878e208346359af502dd11c5"},
		{"ripemd160", "hello ripemd160", "6fb0548fc1acb266457d6ddae686905295b47a2a"},
		{"sha1", "hello sha1", "64faca92dec81be17500f67d521fbd32bb3a6968"},
		{"sha256", "hello sha256", "433855b7d2b96c23a6f60e70c655eb4305e8806b682a9596a200642f947259b1"},
		{"sha384", "hello sha384", "5a37b3a56f9a5ae7b267d25303801d2a610c329d799e9a61879fe35b8108ccb8a4c1154c420ea69fdb6d177fbf6db8b6"},
		{"sha512", "hello sha512", "ae9ae8f823f9b841bd94062d0af09c2dcffc04a705a89e5415330ed1279f369ea990ca92d63adda838696efe28436c0c14d8e805cd0f04b6c6a0e25127de838c"},
		{"sha512_224", "hello sha512_224", "60765c29a50404c4ff1797540fd5bd38383a24d1232e39030638e647"},
		{"sha512_256", "hello sha512_256", "b5e03d2c411178f6c174370e2f420d274cd20b9635ae7a41e40120d826a4b23b"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.algorithm, func(t *testing.T) {
			_, err := vm.Runtime().RunString(`{
			let correct = "` + testCase.want + `";
			let hash = crypto.` + testCase.algorithm + `("` + testCase.origin + `").hex();
			assert.equal(hash, correct);
			}`)
			assert.NoError(t, err)
		})
	}
}

func TestHMac(t *testing.T) {
	vm := modulestest.New(t, js.WithInitial(func(rt *goja.Runtime) {
		c := new(Crypto)
		instance, _ := c.Instantiate(rt)
		_ = rt.Set("crypto", instance)
	}))

	testCases := []struct {
		algorithm, origin, want string
	}{
		{"md5", "hello hmac md5", "6c241e7c650d8a839aeff9a7a28db599"},
		{"ripemd160", "hello hmac ripemd160", "dfbd49aebc8a7cc33ffd3f6e16ab922a23329c2d"},
		{"sha1", "hello hmac sha1", "754cfe3b0dc73755f9d7cfa90ec979e2c1d42f08"},
		{"sha256", "hello hmac sha256", "1d103c86749c67b0c5531bcf4b1125f32540a3bad4165f4efe804a1a5b4dd9f1"},
		{"sha384", "hello hmac sha384", "bc19f1775949f93a53909fb674c65e6978d6fa80173ead68717543d5e01c229ae0d7f6c5f8901147e9998dd477c701cb"},
		{"sha512", "hello hmac sha512", "1f893eec7580ed74a38053c88d0a380c99213f7cb727984692b25f318e49b3e4f0b9c5ae9c5ba942287738d8d812608c0223e1a599bf4b1429a2972cb2a7844a"},
		{"sha512_224", "hello hmac sha512_224", "5f4a8c8cb6404ad3ff85ccbde756d231ff2544be3be702a4706c8a9b"},
		{"sha512_256", "hello hmac sha512_256", "e466b90580a96d60c34a4fb164afc725840c94d30ce1bdafaa00f8f830771dd8"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.algorithm, func(t *testing.T) {
			_, err := vm.Runtime().RunString(`{
			let correct = "` + testCase.want + `";
			let origin = "` + testCase.origin + `";
			let hasher = crypto.createHMAC("` + testCase.algorithm + `", "some secret");
			assert.equal(hasher.encrypt(origin).hex(), correct);
			}`)
			assert.NoError(t, err)
		})
	}
}
