package http

import (
	"context"
	"net/http"
	pkgurl "net/url"
	"testing"
	"time"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCookieJar struct {
	cookies []*http.Cookie
}

func (m *mockCookieJar) SetCookies(u *pkgurl.URL, cookies []*http.Cookie) {
	m.cookies = cookies
}

func (m *mockCookieJar) Cookies(u *pkgurl.URL) []*http.Cookie {
	return m.cookies
}

func (m *mockCookieJar) RemoveCookie(u *pkgurl.URL) {
	m.cookies = nil
}

func TestCookieJarModule(t *testing.T) {
	t.Parallel()
	jar := &mockCookieJar{}
	vm := modulestest.New(t, js.WithInitial(func(rt *sobek.Runtime) {
		module := &CookieJarModule{CookieJar: jar}
		cookieJar, err := module.Instantiate(rt)
		assert.NoError(t, err)
		_ = rt.Set("cookieJar", cookieJar)
	}))
	ctx := context.Background()

	t.Run("set and get", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const url = "https://example.com";
				cookieJar.set(url, {
					name: "test",
					value: "value",
					domain: "example.com",
					path: "/",
					secure: true,
					httpOnly: true,
					sameSite: "strict"
				});

				return cookieJar.get(url, "test");
			}
		`)
		require.NoError(t, err)
		obj := result.ToObject(vm.Runtime())
		assert.Equal(t, "test", obj.Get("name").String())
		assert.Equal(t, "value", obj.Get("value").String())
		assert.Equal(t, "example.com", obj.Get("domain").String())
		assert.Equal(t, "/", obj.Get("path").String())
		assert.True(t, obj.Get("secure").ToBoolean())
		assert.True(t, obj.Get("httpOnly").ToBoolean())
		assert.Equal(t, "strict", obj.Get("sameSite").String())
	})

	t.Run("get all", func(t *testing.T) {
		jar.cookies = []*http.Cookie{
			{Name: "test1", Value: "value1"},
			{Name: "test2", Value: "value2"},
		}

		result, err := vm.RunModule(ctx, `
			export default () => {
				const url = "https://example.com";
				return cookieJar.getAll(url);
			}
		`)
		require.NoError(t, err)
		arr := result.ToObject(vm.Runtime())
		assert.Equal(t, int64(2), arr.Get("length").ToInteger())
		assert.Equal(t, "test1", arr.Get("0").ToObject(vm.Runtime()).Get("name").String())
		assert.Equal(t, "test2", arr.Get("1").ToObject(vm.Runtime()).Get("name").String())
	})

	t.Run("get all with name", func(t *testing.T) {
		jar.cookies = []*http.Cookie{
			{Name: "test1", Value: "value1"},
			{Name: "test2", Value: "value2"},
			{Name: "test1", Value: "value3"},
		}

		result, err := vm.RunModule(ctx, `
			export default () => {
				const url = "https://example.com";
				return cookieJar.getAll(url, "test1");
			}
		`)
		require.NoError(t, err)
		arr := result.ToObject(vm.Runtime())
		assert.Equal(t, int64(2), arr.Get("length").ToInteger())
		assert.Equal(t, "value1", arr.Get("0").ToObject(vm.Runtime()).Get("value").String())
		assert.Equal(t, "value3", arr.Get("1").ToObject(vm.Runtime()).Get("value").String())
	})

	t.Run("delete", func(t *testing.T) {
		jar.cookies = []*http.Cookie{
			{Name: "test1", Value: "value1"},
			{Name: "test2", Value: "value2"},
		}

		result, err := vm.RunModule(ctx, `
			export default () => {
				const url = "https://example.com";
				cookieJar.del(url);
				return cookieJar.getAll(url);
			}
		`)
		require.NoError(t, err)
		arr := result.ToObject(vm.Runtime())
		assert.Equal(t, int64(0), arr.Get("length").ToInteger())
	})

	t.Run("expires", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const url = "https://example.com";
				const expires = Date.now() + 3600000; // 1 hour from now
				cookieJar.set(url, {
					name: "test",
					value: "value",
					expires: expires
				});
				return cookieJar.get(url, "test").expires;
			}
		`)
		require.NoError(t, err)
		assert.InDelta(t, time.Now().Add(time.Hour).UnixMilli(), result.ToInteger(), float64(time.Second.Milliseconds()))
	})
}
