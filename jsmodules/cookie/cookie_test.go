package cookie

import (
	"context"
	"testing"

	"github.com/shiroyk/cloudcat/core"
	"github.com/shiroyk/cloudcat/core/js/modulestest"
	"github.com/stretchr/testify/assert"
)

func TestCookie(t *testing.T) {
	t.Parallel()
	core.Provide[core.Cookie](core.NewCookie())
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
		cookie.set("https://github.com", "max-age=3600;");
		cookie.del("https://github.com");
		assert.true(!cookie.get("https://github.com"), "cookie should be deleted");
		cookie.set("http://localhost", "max-age=3600;");
		assert.equal(cookie.get("http://localhost"), "max-age=3600");
	`)
	assert.NoError(t, err)
}
