package http

import (
	"fmt"
	"testing"

	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
)

func TestURLSearchParams(t *testing.T) {
	vm := modulestest.New(t)

	_, _ = vm.Runtime().RunString(`const params = new URLSearchParams({'name': 'foo'});params.sort();`)

	testCase := []string{
		`try {
			new URLSearchParams(0);
		 } catch (e) {
			assert.true(e.toString().includes('unsupported type'))
		 }`,
		`assert.equal(new URLSearchParams('foo=1&bar=2').toString(), 'foo=1&bar=2')`,
		`assert.equal(new URLSearchParams('?foo=1&bar=2').toString(), 'foo=1&bar=2')`,
		`assert.equal(new URLSearchParams('https://example.com?foo=1&bar=2').toString(), 'https%3A%2F%2Fexample.com%3Ffoo=1&bar=2')`,
		`params.forEach((v, k) => assert.true(v.length == 1))
		 assert.equal(params.get('name'), 'foo')`,
		`params.append('name', 'bar');
		 assert.equal(params.getAll('name').length, 2)`,
		`assert.equal(params.toString(), 'name=foo&name=bar')`,
		`params.append('value', 'zoo');
		 assert.true(params.keys(), ['name', 'value'])`,
		`assert.equal(params.entries().length, 2)`,
		`params.delete('name');
		 assert.equal(params.getAll('name').length, 0)`,
		`assert.true(!params.has('name'))`,
		`params.set('name', 'foobar');
		 assert.equal(params.values().length, 2)`,
		`params.append('000', '114');
		 params.sort();
		 assert.equal(params.toString(), '000=114&name=foobar&value=zoo')`,
		`let str = "";
		 for (const [key, value] of params) {
			str += key + "=" + value + ",";
		 }
		 assert.equal(str, '000=114,name=foobar,value=zoo,')`,
	}

	for i, s := range testCase {
		t.Run(fmt.Sprintf("Script%v", i), func(t *testing.T) {
			_, err := vm.Runtime().RunString(s)
			assert.NoError(t, err)
		})
	}
}
