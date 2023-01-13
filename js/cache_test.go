package js

import (
	"testing"
)

func TestCache(t *testing.T) {
	{
		_, err := testVM.RunString(`
			go.cache.set("cache1", "1");
			go.cache.del("cache1");
			assert(!go.cache.get("cache1"), "cache should be deleted");
			go.cache.set("cache2", "2");
			go.cache.get("cache2");
			assert.equal(go.cache.get("cache2"), "2")
		`)
		if err != nil {
			t.Error(err)
		}
	}

	{
		_, err := testVM.RunString(`
			go.cache.setBytes("cache3", new Uint8Array([50]).buffer);
			assert.equal(new Uint8Array(go.cache.getBytes("cache3"))[0], 50)
		`)
		if err != nil {
			t.Error(err)
		}
	}
}
