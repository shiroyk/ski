package cookie

import (
	"context"
	"testing"

	"github.com/shiroyk/cloudcat/cache"
	"github.com/shiroyk/cloudcat/cache/memory"
	"github.com/shiroyk/cloudcat/internal/di"
	"github.com/shiroyk/cloudcat/js/modulestest"
)

func TestCookie(t *testing.T) {
	t.Parallel()
	di.Override[cache.Cookie](memory.NewCookie())
	ctx := context.Background()
	vm := modulestest.New()

	_, _ = vm.RunString(ctx, `const cookie = require('cloudcat/cookie')`)

	var err error
	errScript := []string{`cookie.set('\x0000', "");`, `cookie.get('\x0000');`, `cookie.del('\x0000');`}
	for _, s := range errScript {
		_, err = vm.RunString(ctx, s)
		if err == nil {
			t.Error("error should not be nil")
		}
	}

	_, err = vm.RunString(ctx, `
		cookie.set("https://github.com", "max-age=3600;");
		cookie.del("https://github.com");
		assert(!cookie.get("https://github.com"), "cookie should be deleted");
		cookie.set("http://localhost", "max-age=3600;");
		assert.equal(cookie.get("http://localhost"), "max-age=3600;");
	`)
	if err != nil {
		t.Error(err)
	}
}
