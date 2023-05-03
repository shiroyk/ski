package core

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCookie(t *testing.T) {
	t.Parallel()
	c := NewCookie()

	u, _ := url.Parse("http://localhost")

	if len(c.Cookies(u)) > 0 {
		t.Fatal("retrieved cookie before adding it")
	}

	{
		maxAge := "MaxAge=3600;"
		c.SetCookies(u, ParseCookie(maxAge))
		assert.Equal(t, "MaxAge=3600", c.CookieString(u))
	}

	{
		maxAge := "MaxAge=7200;"
		c.SetCookieString(u, maxAge)
		assert.Equal(t, "MaxAge=7200;", c.CookieString(u))
	}

	{
		maxAge := "ID=1; MaxAge=7200"
		c.SetCookies(u, ParseCookie(maxAge))
		assert.Equal(t, maxAge, c.CookieString(u))
	}
}
