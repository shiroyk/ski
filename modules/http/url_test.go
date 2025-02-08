package http

import (
	"context"
	"testing"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURL(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t)
	ctx := context.Background()

	t.Run("constructor", func(t *testing.T) {
		tests := []struct {
			name     string
			url      string
			base     string
			expected string
			wantErr  bool
		}{
			{
				name:     "absolute url",
				url:      "https://example.com/path?query=1#hash",
				expected: "https://example.com/path?query=1#hash",
			},
			{
				name:     "relative url with base",
				url:      "/path?query=1#hash",
				base:     "https://example.com",
				expected: "https://example.com/path?query=1#hash",
			},
			{
				name:    "invalid url",
				url:     "://invalid",
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var result sobek.Value
				var err error
				if tt.base == "" {
					result, err = vm.RunString(ctx, `new URL("`+tt.url+`")`)
				} else {
					result, err = vm.RunString(ctx, `new URL("`+tt.url+`", "`+tt.base+`")`)
				}

				if tt.wantErr {
					assert.Error(t, err)
					return
				}

				require.NoError(t, err)
				assert.Equal(t, tt.expected, result.String())
			})
		}
	})

	t.Run("properties", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const url = new URL('https://user:pass@example.com:8080/path?query=1#hash');
				return ({
					hash: url.hash,
					host: url.host,
					hostname: url.hostname,
					href: url.href,
					origin: url.origin,
					password: url.password,
					pathname: url.pathname,
					port: url.port,
					protocol: url.protocol,
					username: url.username,
					search: url.searchParams.toString()
				})
			}
		`)

		require.NoError(t, err)
		obj := result.ToObject(vm.Runtime())
		assert.Equal(t, "#hash", obj.Get("hash").String())
		assert.Equal(t, "example.com:8080", obj.Get("host").String())
		assert.Equal(t, "example.com", obj.Get("hostname").String())
		assert.Equal(t, "https://user:pass@example.com:8080/path?query=1#hash", obj.Get("href").String())
		assert.Equal(t, "https://example.com:8080", obj.Get("origin").String())
		assert.Equal(t, "pass", obj.Get("password").String())
		assert.Equal(t, "/path", obj.Get("pathname").String())
		assert.Equal(t, "8080", obj.Get("port").String())
		assert.Equal(t, "https:", obj.Get("protocol").String())
		assert.Equal(t, "user", obj.Get("username").String())
		assert.Equal(t, "query=1", obj.Get("search").String())
	})

	t.Run("setters", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const url = new URL('https://example.com');
				url.hash = '#newhash';
				url.host = 'newhost:9090';
				url.hostname = 'newhost';
				url.password = 'newpass';
				url.pathname = '/newpath';
				url.port = '9091';
				url.protocol = 'http:';
				url.username = 'newuser';
				return url.href
			}
		`)
		require.NoError(t, err)
		assert.Equal(t, "http://newuser:newpass@newhost:9091/newpath#newhash", result.String())
	})

	t.Run("searchParams", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const url = new URL('https://example.com?a=1&b=2');
				url.searchParams.append('c', '3');
				url.searchParams.set('b', '22');
				url.searchParams.delete('a');
				return url.toString()
			}
		`)
		require.NoError(t, err)
		assert.Equal(t, "https://example.com?b=22&c=3", result.String())
	})

	t.Run("toString", func(t *testing.T) {
		result, err := vm.RunString(ctx, `
			const url = new URL('https://example.com');
			[url.toString(), url.toJSON(), String(url)]
		`)
		require.NoError(t, err)
		arr := result.ToObject(vm.Runtime())
		assert.Equal(t, "https://example.com", arr.Get("0").String())
		assert.Equal(t, "https://example.com", arr.Get("1").String())
		assert.Equal(t, "https://example.com", arr.Get("2").String())
	})
}
