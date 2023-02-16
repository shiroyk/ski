package bolt

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/shiroyk/cloudcat/lib/utils"
)

func TestCookie(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "test_cache")
	os.MkdirAll(tempDir, os.ModePerm)
	defer os.RemoveAll(tempDir)

	c, err := NewCookie(tempDir)
	if err != nil {
		t.Fatal(err)
	}

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
