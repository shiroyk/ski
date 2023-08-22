package cache

import (
	"context"
	"testing"

	"github.com/shiroyk/cloudcat"
	"github.com/shiroyk/cloudcat/js/modulestest"
	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	t.Parallel()
	cloudcat.Provide[cloudcat.Cache](cloudcat.NewCache())
	ctx := context.Background()
	vm := modulestest.New(t)

	_, err := vm.RunString(ctx, `
			const cache = require('cloudcat/cache');
			cache.set("cache1", "1");
			cache.del("cache1");
			assert.true(!cache.get("cache1"), "cache should be deleted");
			cache.set("cache2", "2", "1s");
			cache.get("cache2");
			assert.equal(cache.get("not exists"), undefined);
			assert.equal(cache.get("cache2"), "2");
			cache.setBytes("cache3", new Uint8Array([50]));
			assert.equal(new Uint8Array(cache.getBytes("cache3"))[0], 50);
			cache.setBytes("cache4", new Uint8Array([60]).buffer);
			assert.equal(new Uint8Array(cache.getBytes("cache4"))[0], 60);
		`)
	assert.NoError(t, err)
}
