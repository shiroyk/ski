package common

import (
	"context"
	"testing"

	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
)

func TestDOMException(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t)

	_, err := vm.RunModule(context.Background(), `
		const err = new DOMException('msg', 'AbortError');
		assert.equal(err.name, 'AbortError');
		assert.equal(err.message, 'msg');
		assert.equal(err.code, 20);
		assert.equal(err.toString(), 'AbortError: msg');
		assert.equal(String(err), 'AbortError: msg');
		assert.true(err instanceof DOMException);
		assert.true(err instanceof Error);
		assert.true(Object.getPrototypeOf(DOMException.prototype) === Error.prototype);
		assert.equal(Object.prototype.toString.call(err), '[object DOMException]');
		assert.equal(typeof err.stack, 'string');
		
		const empty = new DOMException();
		assert.equal(empty.name, 'Error');
		assert.equal(empty.message, '');
		assert.equal(empty.code, 0);
		assert.equal(new DOMException('m', 'TimeoutError').code, 23);
		assert.equal(new DOMException('m', 'NopeError').code, 0);
		
		assert.equal(DOMException.ABORT_ERR, 20);
		assert.equal(DOMException.TIMEOUT_ERR, 23);
		assert.equal(DOMException.prototype.ABORT_ERR, 20);
		
		// The DOMException instances stay plain and extensible.
		const plain = new DOMException('m', 'AbortError');
		assert.equal(Object.getOwnPropertyNames(plain).join(','), 'stack');
		assert.true(Object.isExtensible(plain));
		plain.expando = 1;
		assert.equal(plain.expando, 1);
	`)
	assert.NoError(t, err)

	rt := vm.Runtime()
	exc := NewDOMException(rt, "go", "TimeoutError")
	assert.Equal(t, "TimeoutError", exc.Get("name").String())
	assert.Equal(t, "go", exc.Get("message").String())
	assert.Equal(t, int64(23), exc.Get("code").ToInteger())
}
