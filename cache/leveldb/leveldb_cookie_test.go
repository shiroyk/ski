package leveldb

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/shiroyk/cloudcat/utils"
)

func TestCookie(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "cookie")
	defer os.RemoveAll(tempDir)

	c, err := NewCookie(filepath.Join(tempDir, "Db"))
	if err != nil {
		t.Fatalf("New leveldb: %v", err)
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
