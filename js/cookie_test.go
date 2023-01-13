package js

import (
	"testing"
)

func TestCookie(t *testing.T) {
	var err error
	errScript := []string{`go.cookie.set('\x0000', "");`, `go.cookie.get('\x0000');`, `go.cookie.del('\x0000');`}
	for _, s := range errScript {
		_, err = testVM.RunString(s)
		if err == nil {
			t.Error("error should not be nil")
		}
	}

	_, err = testVM.RunString(`
		go.cookie.set("https://github.com", "max-age=3600;");
		go.cookie.del("https://github.com");
		assert(!go.cookie.get("https://github.com"), "cookie should be deleted");
		go.cookie.set("http://localhost", "max-age=3600;");
		assert.equal(go.cookie.get("http://localhost"), "max-age=3600;");
	`)
	if err != nil {
		t.Error(err)
	}
}
