package http

import (
	"context"
	"fmt"
	"testing"

	"github.com/shiroyk/cloudcat/core/js/modulestest"
	"github.com/stretchr/testify/assert"
)

func TestURLSearchParams(t *testing.T) {
	ctx := context.Background()
	vm := modulestest.New(t)

	testCase := []string{
		`form = new URLSearchParams();form.sort();`,
		`try {
			form = new URLSearchParams(0);
		 } catch (e) {
			assert.true(e.toString().includes('unsupported type'))
		 }`,
		`form = new URLSearchParams({'name': 'foo'});
		 form.forEach((v, k) => assert.true(v.length == 1))
		 assert.equal(form.get('name'), 'foo')`,
		`form.append('name', 'bar');
		 assert.equal(form.getAll('name').length, 2)`,
		`assert.equal(form.toString(), 'name=foo&name=bar')`,
		`form.append('value', 'zoo');
		 assert.true(form.keys(), ['name', 'value'])`,
		`assert.equal(form.entries().length, 3)`,
		`form.delete('name');
		 assert.equal(form.getAll('name').length, 0)`,
		`assert.true(!form.has('name'))`,
		`form.set('name', 'foobar');
		 assert.equal(form.values().length, 2)`,
	}

	for i, s := range testCase {
		t.Run(fmt.Sprintf("Script%v", i), func(t *testing.T) {
			_, err := vm.RunString(ctx, s)
			assert.NoError(t, err)
		})
	}
}
