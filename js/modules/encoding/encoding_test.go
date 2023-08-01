package encoding

import (
	"context"
	"fmt"
	"testing"

	"github.com/shiroyk/cloudcat/js/modulestest"
	"github.com/stretchr/testify/assert"
)

func TestEncodingBase64(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		return
	}

	vm := modulestest.New(t)
	_, _ = vm.Runtime().RunString(`
		const encoding = require('cloudcat/encoding');
	`)

	buffer := vm.Runtime().NewArrayBuffer([]byte{100, 97, 110, 107, 111, 103, 97, 105})

	encodeTestCases := []struct {
		raw          any
		want         string
		padding, url bool
	}{
		{"dankogai", "ZGFua29nYWk=", false, false},
		{"dankogai", "ZGFua29nYWk", true, false},
		{"dankogai", "ZGFua29nYWk", false, true},
		{"dankogai", "ZGFua29nYWk", false, true},
		{buffer, "ZGFua29nYWk=", false, false},
		{buffer, "ZGFua29nYWk", true, false},
		{buffer, "ZGFua29nYWk", false, true},
		{"小飼弾", "5bCP6aO85by+", false, false},
		{"小飼弾", "5bCP6aO85by+", true, false},
		{"小飼弾", "5bCP6aO85by-", false, true},
	}

	for i, testCase := range encodeTestCases {
		t.Run(fmt.Sprintf("encode %v", i), func(t *testing.T) {
			_ = vm.Runtime().Set("raw", testCase.raw)
			_ = vm.Runtime().Set("want", testCase.want)
			_ = vm.Runtime().Set("padding", testCase.padding)

			code := "result = encoding.base64.encode"
			if testCase.url {
				code += "URI"
			}
			_, err := vm.RunString(context.Background(), code+"(raw, padding);assert.equal(want, result);")
			assert.NoError(t, err)
		})
	}

	decodeTestCases := []struct {
		raw      string
		want     any
		toBuffer bool
	}{
		{"ZGFua29nYWk=", "dankogai", false},
		{"ZGFua29nYWk", "dankogai", false},
		{"ZGFua29nYWk", buffer, true},
		{"5bCP6aO85by+", "小飼弾", false},
		{"5bCP6aO85by-", "小飼弾", false},
	}

	for i, testCase := range decodeTestCases {
		t.Run(fmt.Sprintf("decode %v", i), func(t *testing.T) {
			_ = vm.Runtime().Set("raw", testCase.raw)
			_ = vm.Runtime().Set("want", testCase.want)
			_ = vm.Runtime().Set("toBuffer", testCase.toBuffer)

			_, err := vm.RunString(context.Background(), `
			assert.equal(want, encoding.base64.decode(raw, toBuffer));
			`)
			assert.NoError(t, err)
		})
	}
}
