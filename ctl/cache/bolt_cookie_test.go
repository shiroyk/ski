package cache

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/shiroyk/cloudcat/core"
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

	u, _ := url.Parse("http://localhost")

	if len(c.Cookies(u)) > 0 {
		t.Fatal("retrieved cookie before adding it")
	}

	{
		maxAge := "MaxAge=3600;"
		c.SetCookies(u, cloudcat.ParseCookie(maxAge))
		assert.Equal(t, "MaxAge=3600", c.CookieString(u))
	}

	{
		maxAge := "MaxAge=7200;"
		c.SetCookieString(u, maxAge)
		assert.Equal(t, "MaxAge=7200", c.CookieString(u))
	}
}
