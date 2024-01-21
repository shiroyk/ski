package cache

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/shiroyk/ski"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t, js.WithInitial(func(rt *goja.Runtime) {
		cache := Cache{ski.NewCache()}
		instantiate, err := cache.Instantiate(rt)
		if err != nil {
			t.Fatal(err)
		}
		_ = rt.Set("cache", instantiate)
	}))

	_, err := vm.Runtime().RunString(`
			cache.set("cache1", "1");
			cache.del("cache1");
			assert.true(!cache.get("cache1"), "cache should be deleted");
			cache.set("cache2", "2", "1s");
			assert.equal(cache.get("not exists"), undefined);
			assert.equal(cache.get("not exists"), undefined);
			assert.equal(cache.get("cache2"), "2");
			cache.setBytes("cache3", new Uint8Array([50]));
			assert.equal(new Uint8Array(cache.getBytes("cache3"))[0], 50);
			cache.setBytes("cache4", new Uint8Array([60]).buffer);
			assert.equal(new Uint8Array(cache.getBytes("cache4"))[0], 60);
		`)
	assert.NoError(t, err)
}
