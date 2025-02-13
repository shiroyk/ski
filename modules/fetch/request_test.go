package fetch

import (
	"context"
	"testing"

	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequest(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t)
	ctx := context.Background()

	t.Run("constructor", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
		export default () => {
			const request = new Request("https://example.com/api", {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
					"X-Custom": "test"
				},
				body: JSON.stringify({ data: "test" }),
				mode: "cors",
				credentials: "include",
				cache: "no-cache",
				redirect: "follow",
				referrer: "https://example.com",
				integrity: "sha256-hash"
			});
			
			return {
				url: request.url,
				method: request.method,
				contentType: request.headers.get("content-type"),
				customHeader: request.headers.get("x-custom"),
				mode: request.mode,
				credentials: request.credentials,
				cache: request.cache,
				redirect: request.redirect,
				referrer: request.referrer,
				integrity: request.integrity
			};
		}
		`)
		require.NoError(t, err)
		obj := modulestest.PromiseResult(result).ToObject(vm.Runtime())

		assert.Equal(t, "https://example.com/api", obj.Get("url").String())
		assert.Equal(t, "POST", obj.Get("method").String())
		assert.Equal(t, "application/json", obj.Get("contentType").String())
		assert.Equal(t, "test", obj.Get("customHeader").String())
		assert.Equal(t, "cors", obj.Get("mode").String())
		assert.Equal(t, "include", obj.Get("credentials").String())
		assert.Equal(t, "no-cache", obj.Get("cache").String())
		assert.Equal(t, "follow", obj.Get("redirect").String())
		assert.Equal(t, "https://example.com", obj.Get("referrer").String())
		assert.Equal(t, "sha256-hash", obj.Get("integrity").String())
	})

	t.Run("request body methods", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name: "text",
				input: `
				export default async () => {
					const request = new Request("https://example.com", {
						method: "POST",
						body: "hello world"
					});
					return await request.text();
				}`,
				expected: "hello world",
			},
			{
				name: "json",
				input: `
				export default async () => {
					const request = new Request("https://example.com", {
						method: "POST",
						body: JSON.stringify({message: "hello"})
					});
					const data = await request.json();
					return data.message;
				}`,
				expected: "hello",
			},
			{
				name: "arrayBuffer",
				input: `
				export default async () => {
					const request = new Request("https://example.com", {
						method: "POST",
						body: "buffer test"
					});
					const buffer = await request.arrayBuffer();
					return String.fromCharCode.apply(String, new Uint8Array(buffer));
				}`,
				expected: "buffer test",
			},
			{
				name: "body used error",
				input: `
				export default async () => {
					const request = new Request("https://example.com", {
						body: "test data"
					});
					await request.text();
					try {
						await request.text();
						return "should not reach here";
					} catch (e) {
						return e.toString();
					}
				}`,
				expected: "body stream already read",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := vm.RunModule(ctx, tt.input)
				require.NoError(t, err)
				assert.Contains(t, modulestest.PromiseResult(result).String(), tt.expected)
			})
		}
	})

	t.Run("clone", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
		export default async () => {
			const original = new Request("https://example.com", {
				method: "POST",
				body: "original body"
			});
			const clone = original.clone();
			
			return {
				originalBody: await original.text(),
				cloneBody: await clone.text(),
				originalMethod: original.method,
				cloneMethod: clone.method,
				originalUrl: original.url,
				cloneUrl: clone.url
			};
		}
		`)
		require.NoError(t, err)
		obj := modulestest.PromiseResult(result).ToObject(vm.Runtime())

		assert.Equal(t, "original body", obj.Get("originalBody").String())
		assert.Equal(t, "original body", obj.Get("cloneBody").String())
		assert.Equal(t, "POST", obj.Get("originalMethod").String())
		assert.Equal(t, "POST", obj.Get("cloneMethod").String())
		assert.Equal(t, "https://example.com", obj.Get("originalUrl").String())
		assert.Equal(t, "https://example.com", obj.Get("cloneUrl").String())
	})
}
