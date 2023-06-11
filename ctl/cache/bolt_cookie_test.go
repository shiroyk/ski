package cache

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	cloudcat "github.com/shiroyk/cloudcat/core"
	"github.com/stretchr/testify/assert"
)

func TestCookie(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "test_cache")
	assert.NoError(t, os.MkdirAll(tempDir, os.ModePerm))
	defer assert.NoError(t, os.RemoveAll(tempDir))

	c, err := NewCookie(Options{Path: tempDir})
	if err != nil {
		t.Fatal(err)
	}

	u, _ := url.Parse("https://github.com")

	if len(c.Cookies(u)) > 0 {
		t.Fatal("retrieved cookie before adding it")
	}

	raw := "has_recent_activity=1; path=/; secure; HttpOnly; SameSite=Lax"
	c.SetCookies(u, cloudcat.ParseSetCookie(raw))
	assert.Equal(t, []string{"has_recent_activity=1; Path=/; HttpOnly; Secure; SameSite=Lax"}, c.CookieString(u))
	c.DeleteCookie(u)
	assert.Empty(t, c.Cookies(u))
}
