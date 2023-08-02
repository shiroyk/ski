package http

import (
	"context"
	"fmt"
	"testing"

	"github.com/shiroyk/cloudcat/js/modulestest"
	"github.com/stretchr/testify/assert"
)

func TestFormData(t *testing.T) {
	ctx := context.Background()
	vm := modulestest.New(t)

	_, _ = vm.Runtime().RunString(`const mp = new FormData({
			'file': new Uint8Array([50]),
			'name': 'foo'
		 });`)

	testCase := []string{
		`try {
			new FormData(0);
		 } catch (e) {
			assert.true(e.toString().includes('unsupported type'))
		 }`,
		`assert.equal(mp.get('name'), 'foo')`,
		`mp.append('file', new Uint8Array([51]).buffer);
		 assert.equal(mp.getAll('file').length, 2)`,
		`mp.append('name', 'bar');
		 assert.equal(mp.keys().length, 2);
		 assert.equal(mp.get('name'), 'foo');`,
		`assert.equal(mp.entries().length, 2)`,
		`mp.delete('name');
		 assert.equal(mp.getAll('name').length, 0)`,
		`assert.true(!mp.has('name'))`,
		`mp.set('name', 'foobar');
		 assert.equal(mp.values().length, 2)`,
	}

	for i, s := range testCase {
		t.Run(fmt.Sprintf("Script%v", i), func(t *testing.T) {
			_, err := vm.RunString(ctx, s)
			assert.NoError(t, err)
		})
	}
}
