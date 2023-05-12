package cloudcat

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
		cookie := ParseCookie(maxAge)
		c.SetCookies(u, cookie)
		assert.EqualValues(t, cookie, c.Cookies(u))
		c.DeleteCookie(u)
		assert.Nil(t, c.Cookies(u))
	}

	{
		maxAge := "Name=test; MaxAge=7200"
		cookie := ParseCookie(maxAge)
		c.SetCookies(u, cookie)
		assert.EqualValues(t, cookie, c.Cookies(u))
		c.DeleteCookie(u)
		assert.Nil(t, c.Cookies(u))
	}
}
