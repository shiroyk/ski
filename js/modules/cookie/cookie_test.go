package cookie

import (
	"context"
	"testing"

	"github.com/shiroyk/cloudcat"
	"github.com/shiroyk/cloudcat/js/modulestest"
	"github.com/stretchr/testify/assert"
)

func TestCookie(t *testing.T) {
	t.Parallel()
	cloudcat.Provide[cloudcat.Cookie](cloudcat.NewCookie())
	ctx := context.Background()
	vm := modulestest.New(t)

	_, _ = vm.RunString(ctx, `const cookie = require('cloudcat/cookie')`)

	var err error
	errScript := []string{`cookie.set('\x0000', "");`, `cookie.get('\x0000');`, `cookie.del('\x0000');`}
	for _, s := range errScript {
		_, err = vm.RunString(ctx, s)
		assert.ErrorContains(t, err, "net/url: invalid control character in URL")
	}

	_, err = vm.RunString(ctx, `
		cookie.set("https://github.com", ["test=1; path=/; secure; HttpOnly;"]);
		cookie.del("https://github.com");
		assert.true(!cookie.get("https://github.com").length, "cookie should be deleted");
		cookie.set("https://github.com", ["has_recent_activity=1; path=/; secure; HttpOnly; SameSite=Lax"]);
		assert.equal("has_recent_activity=1", cookie.get("https://github.com")[0]);
	`)
	assert.NoError(t, err)
}
