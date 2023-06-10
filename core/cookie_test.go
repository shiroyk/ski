package cloudcat

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCookie(t *testing.T) {
	t.Parallel()
	c := NewCookie()

	u, _ := url.Parse("https://github.com")

	if len(c.Cookies(u)) > 0 {
		t.Fatal("retrieved cookie before adding it")
	}

	raw := "has_recent_activity=1; path=/; secure; HttpOnly; SameSite=Lax"
	c.SetCookies(u, ParseSetCookie(raw))
	assert.Equal(t, []string{"has_recent_activity=1"}, c.CookieString(u))
	c.DeleteCookie(u)
	assert.Nil(t, c.Cookies(u))
}
