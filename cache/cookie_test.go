package cache

import (
	"net/url"
	"testing"

	"github.com/shiroyk/cloudcat/cache/memory"
	"github.com/shiroyk/cloudcat/utils"
)

func TestCookie(t *testing.T) {
	c := memory.NewCookie()

	u, _ := url.Parse("http://localhost")

	if len(c.Cookies(u)) > 0 {
		t.Fatal("retrieved cookie before adding it")
	}

	{
		maxAge := "MaxAge=3600;"
		c.SetCookies(u, utils.ParseCookie(maxAge))
		if c.CookieString(u) != "MaxAge=3600" {
			t.Fatalf("unexpected cookie %s", c.CookieString(u))
		}
	}

	{
		maxAge := "MaxAge=7200;"
		c.SetCookieString(u, maxAge)
		if c.CookieString(u) != "MaxAge=7200" {
			t.Fatalf("unexpected cookie %s", c.CookieString(u))
		}
	}
}
