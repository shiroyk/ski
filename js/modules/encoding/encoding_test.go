package encoding

import (
	"fmt"
	"testing"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
)

func TestEncodingBase64(t *testing.T) {
	t.Parallel()

	vm := modulestest.New(t, js.WithInitial(func(rt *sobek.Runtime) {
		instantiate, _ := new(Encoding).Instantiate(rt)
		_ = rt.Set("encoding", instantiate)
	}))

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
			_, err := vm.Runtime().RunString(code + "(raw, padding);assert.equal(want, result);")
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

			_, err := vm.Runtime().RunString(`
			assert.equal(want, encoding.base64.decode(raw, toBuffer));
			`)
			assert.NoError(t, err)
		})
	}
}
