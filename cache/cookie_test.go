package cache

import (
	"net/url"
	"testing"

	"github.com/shiroyk/cloudcat/cache/memory"
	"github.com/shiroyk/cloudcat/lib/utils"
	"github.com/stretchr/testify/assert"
)

func TestCookie(t *testing.T) {
	t.Parallel()
	c := memory.NewCookie()

	u, _ := url.Parse("http://localhost")

	if len(c.Cookies(u)) > 0 {
		t.Fatal("retrieved cookie before adding it")
	}

	{
		maxAge := "MaxAge=3600;"
		c.SetCookies(u, utils.ParseCookie(maxAge))
		assert.Equal(t, "MaxAge=3600", c.CookieString(u))
	}

	{
		maxAge := "MaxAge=7200;"
		c.SetCookieString(u, maxAge)
		assert.Equal(t, "MaxAge=7200;", c.CookieString(u))
	}
}
