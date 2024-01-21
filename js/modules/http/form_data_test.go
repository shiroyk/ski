package http

import (
	"context"
	"testing"

	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
)

func TestFormData(t *testing.T) {
	vm := modulestest.New(t)

	_, err := vm.RunString(context.Background(), `
		const form = new FormData({
			'file': new Uint8Array([50]),
			'name': 'foo'
		});
		try {
			new FormData(0);
		} catch (e) {
			assert.true(e.toString().includes('unsupported type'))
		}
		assert.equal(form.get('name'), 'foo')
		form.delete('name');
		assert.equal(form.get('name'), null);
		form.append('file', new Uint8Array([51]).buffer);
		assert.equal(form.getAll('file').length, 2)
		form.append('name', 'bar');
		assert.equal(form.keys().length, 2);
		assert.equal(form.get('name'), 'bar');
		assert.equal(form.entries().length, 2)
		form.delete('name');
		assert.equal(form.getAll('name').length, 0)
		assert.true(!form.has('name'))
		form.set('name', 'foobar');
		assert.equal(form.values().length, 2)
		let str = "";
		for (const [key, value] of form) {
			str += key + ",";
		}
		assert.equal(str, 'file,name,')`)
	assert.NoError(t, err)
}
