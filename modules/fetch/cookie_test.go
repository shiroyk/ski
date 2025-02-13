package fetch

import (
	"net/http"
	pkgurl "net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCookieJar(t *testing.T) {
	t.Parallel()
	c := NewCookieJar()

	u, _ := pkgurl.Parse("https://github.com")

	cookies := []*http.Cookie{{Name: "has_recent_activity", Value: "1", Path: "/", Secure: true}}
	c.SetCookies(u, cookies)
	assert.NotNil(t, c.Cookies(u))
	c.RemoveCookie(u)
	assert.Nil(t, c.Cookies(u))
}
