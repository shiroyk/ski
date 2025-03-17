package fetch

import (
	"context"
	"testing"

	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponse(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t)
	ctx := context.Background()

	t.Run("constructor", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
		export default () => {
			const response = new Response("hello world", {
				status: 201,
				statusText: "Created",
				headers: {
					"Content-Type": "text/plain",
					"X-Custom": "test"
				}
			});
			return {
				ok: response.ok,
				status: response.status,
				statusText: response.statusText,
				contentType: response.headers.get("content-type"),
				customHeader: response.headers.get("x-custom"),
				type: response.type,
			};
		}
		`)
		require.NoError(t, err)
		obj := result.ToObject(vm.Runtime())
		assert.True(t, obj.Get("ok").ToBoolean())
		assert.Equal(t, int64(201), obj.Get("status").ToInteger())
		assert.Equal(t, "Created", obj.Get("statusText").String())
		assert.Equal(t, "text/plain", obj.Get("contentType").String())
		assert.Equal(t, "test", obj.Get("customHeader").String())
		assert.Equal(t, "default", obj.Get("type").String())
	})

	t.Run("response body methods", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name: "text",
				input: `
				export default async () => {
					const response = new Response("hello world");
					return await response.text();
				}`,
				expected: "hello world",
			},
			{
				name: "json",
				input: `
				export default async () => {
					const response = new Response('{"message":"hello"}');
					const data = await response.json();
					return data.message;
				}`,
				expected: "hello",
			},
			{
				name: "arrayBuffer",
				input: `
				export default async () => {
					const response = new Response("hello");
					const buffer = await response.arrayBuffer();
					return String.fromCharCode.apply(String, new Uint8Array(buffer));
				}`,
				expected: "hello",
			},
			{
				name: "body used error",
				input: `
				export default async () => {
					const response = new Response("test");
					await response.text();
					try {
						await response.text();
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
				if tt.name == "body used error" {
					assert.Contains(t, modulestest.PromiseResult(result).String(), tt.expected)
				} else {
					assert.Equal(t, tt.expected, modulestest.PromiseResult(result).String())
				}
			})
		}
	})

	t.Run("clone", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
		export default async () => {
			const response = new Response("original");
			const clone = response.clone();
			
			const results = {
				original: await response.text(),
				clone: await clone.text(),
			};
			
			try {
				await response.text(); // Should fail
				results.error = "should have failed";
			} catch (e) {
				results.error = e.toString();
			}
			
			return results;
		}
		`)
		require.NoError(t, err)
		obj := modulestest.PromiseResult(result).ToObject(vm.Runtime())
		assert.Equal(t, "original", obj.Get("original").String())
		assert.Equal(t, "original", obj.Get("clone").String())
		assert.Contains(t, obj.Get("error").String(), "body stream already read")
	})

	t.Run("Response.json static method", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
		export default async () => {
			const data = { message: "hello" };
			const response = Response.json(data);
			return {
				contentType: response.headers.get("content-type"),
				body: await response.text(),
			};
		}
		`)
		require.NoError(t, err)
		obj := modulestest.PromiseResult(result).ToObject(vm.Runtime())
		assert.Equal(t, "application/json", obj.Get("contentType").String())
		assert.Equal(t, `{"message":"hello"}`, obj.Get("body").String())
	})

	t.Run("body types", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
		export default async () => {
			const results = {};
			
			const uint8Array = new Uint8Array([104, 101, 108, 108, 111]); // "hello"
			const response1 = new Response(uint8Array);
			results.uint8Array = await response1.text();
			
			const blob = new Blob(["blob test"]);
			const response2 = new Response(blob);
			results.blob = await response2.text();
			
			const formData = new FormData({"field": "form value"});
			const response3 = new Response(formData);
			results.formData = await response3.text();
			
			return results;
		}
		`)
		require.NoError(t, err)
		obj := modulestest.PromiseResult(result).ToObject(vm.Runtime())
		assert.Equal(t, "hello", obj.Get("uint8Array").String())
		assert.Equal(t, "blob test", obj.Get("blob").String())
		assert.Contains(t, obj.Get("formData").String(), "form value")
	})
}
