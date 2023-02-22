package cache

import (
	"context"
	"testing"

	"github.com/shiroyk/cloudcat/cache"
	"github.com/shiroyk/cloudcat/cache/memory"
	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/js/modulestest"
)

func TestCache(t *testing.T) {
	t.Parallel()
	di.Provide[cache.Cache](memory.NewCache(), false)
	ctx := context.Background()
	vm := modulestest.New(t)

	_, err := vm.RunString(ctx, `
			const cache = require('cloudcat/cache');
			cache.set("cache1", "1");
			cache.del("cache1");
			assert.true(!cache.get("cache1"), "cache should be deleted");
			cache.set("cache2", "2", "1s");
			cache.get("cache2");
			assert.equal(cache.get("cache2"), "2");
			cache.setBytes("cache3", new Uint8Array([50]).buffer);
			assert.equal(new Uint8Array(cache.getBytes("cache3"))[0], 50);
		`)
	if err != nil {
		t.Error(err)
	}
}
