package http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dop251/goja"
	"github.com/shiroyk/ski"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
)

func TestCookie(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t, js.WithInitial(func(rt *goja.Runtime) {
		jar := CookieJar{ski.NewCookieJar()}
		instantiate, err := jar.Instantiate(rt)
		if err != nil {
			t.Fatal(err)
		}
		_ = rt.Set("cookieJar", instantiate)
		client := http.Client{Jar: jar}
		instance, _ := (&Http{&client}).Instantiate(rt)
		_ = rt.Set("http", instance)
	}))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie("foo"); err == nil {
			_, err = fmt.Fprint(w, cookie.String())
			assert.NoError(t, err)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	_ = vm.Runtime().Set("url", ts.URL)

	_, err := vm.RunString(context.Background(), `
		cookieJar.set("https://github.com", { name: "foo", value: "bar", path: "/", maxAge: 7200 });
		assert.equal("bar", cookieJar.get({ url: "https://github.com" }).value);
		cookieJar.del("https://github.com");
		assert.true(!cookieJar.get({ url: "https://github.com" }), "cookie should be deleted");
		cookieJar.set(url, { name: "foo", value: "bar", path: "/", maxAge: 7200 });
		const res1 = http.get(url);
		assert.equal(res1.text(), "foo=bar");
		cookieJar.del(url);
		const res2 = http.get(url);
		assert.equal(res2.text(), "");
	`)
	assert.NoError(t, err)
}
