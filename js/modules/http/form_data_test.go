package http

import (
	"context"
	"fmt"
	"testing"

	"github.com/shiroyk/cloudcat/js/modulestest"
)

func TestFormData(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	vm := modulestest.New()

	testCase := []string{
		`mp = new FormData();`,
		`try {
			mp = new FormData(0);
		 } catch (e) {
			assert(e.toString().includes('unsupported type'))
		 }`,
		`mp = new FormData({
			'file': new Uint8Array([50]).buffer,
			'name': 'foo'
		 });
		 assert.equal(mp.get('name'), 'foo')`,
		`mp.append('file', new Uint8Array([51]).buffer);
		 assert.equal(mp.getAll('file').length, 2)`,
		`mp.append('name', 'bar');
		 assert.equal(mp.keys().length, 2);
		 assert.equal(mp.get('name'), 'foo');`,
		`assert.equal(mp.entries().length, 2)`,
		`mp.delete('name');
		 assert.equal(mp.getAll('name').length, 0)`,
		`assert(!mp.has('name'))`,
		`mp.set('name', 'foobar');
		 assert.equal(mp.values().length, 2)`,
	}

	for i, s := range testCase {
		t.Run(fmt.Sprintf("Script%v", i), func(t *testing.T) {
			_, err := vm.RunString(ctx, s)
			if err != nil {
				t.Errorf("Script: %s , %s", s, err)
			}
		})
	}
}
