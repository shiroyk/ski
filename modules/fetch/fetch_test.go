package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/shiroyk/ski/js/modulestest"
	"github.com/shiroyk/ski/js/promise"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetch(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t)
	ctx := context.Background()

	t.Run("basic request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			w.Write([]byte("hello world"))
		}))
		defer server.Close()

		result, err := vm.RunModule(ctx, `
		export default async (url) => {
			const response = await fetch(url);
			return {
				ok: response.ok,
				status: response.status,
				statusText: response.statusText,
				text: await response.text(),
			};
		}
		`, server.URL)
		require.NoError(t, err)
		obj := modulestest.PromiseResult(result).ToObject(vm.Runtime())
		assert.True(t, obj.Get("ok").ToBoolean())
		assert.Equal(t, int64(200), obj.Get("status").ToInteger())
		assert.Equal(t, "200 OK", obj.Get("statusText").String())
		assert.Equal(t, "hello world", obj.Get("text").String())
	})

	t.Run("new request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			w.Write([]byte("hello world"))
		}))
		defer server.Close()

		result, err := vm.RunModule(ctx, `
		export default async (url) => {
			const response = await fetch(new Request(url, {
				method: "GET",
			}));
			return {
				ok: response.ok,
				status: response.status,
				statusText: response.statusText,
				text: await response.text(),
			};
		}
		`, server.URL)
		require.NoError(t, err)
		obj := modulestest.PromiseResult(result).ToObject(vm.Runtime())
		assert.True(t, obj.Get("ok").ToBoolean())
		assert.Equal(t, int64(200), obj.Get("status").ToInteger())
		assert.Equal(t, "200 OK", obj.Get("statusText").String())
		assert.Equal(t, "hello world", obj.Get("text").String())
	})

	t.Run("request methods", func(t *testing.T) {
		tests := []struct {
			name   string
			method string
			body   string
		}{
			{name: "GET", method: "GET"},
			{name: "POST", method: "POST", body: "post data"},
			{name: "PUT", method: "PUT", body: "put data"},
			{name: "DELETE", method: "DELETE"},
			{name: "HEAD", method: "HEAD"},
			{name: "OPTIONS", method: "OPTIONS"},
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			i, err := strconv.Atoi(r.URL.Query().Get("i"))
			require.NoError(t, err)
			tt := tests[i]
			assert.Equal(t, tt.method, r.Method)
			if tt.body != "" {
				body, err := io.ReadAll(r.Body)
				if assert.NoError(t, err) {
					assert.Equal(t, tt.body, string(body))
				}
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		for i, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := vm.RunModule(ctx, `
				export default async (url, method, body) => {
					const response = await fetch(url, { method, body });
					return response.status;
				}
				`, fmt.Sprintf("%s?i=%d", server.URL, i), tt.method, tt.body)
				require.NoError(t, err)
				assert.Equal(t, int64(200), modulestest.PromiseResult(result).ToInteger())
			})
		}
	})

	t.Run("request headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "custom value", r.Header.Get("X-Custom"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		result, err := vm.RunModule(ctx, `
		export default async (url) => {
			const response = await fetch(url, {
				headers: {
					"Content-Type": "application/json",
					"X-Custom": "custom value"
				}
			});
			return response.status;
		}
		`, server.URL)
		require.NoError(t, err)
		assert.Equal(t, int64(200), modulestest.PromiseResult(result).ToInteger())
	})

	t.Run("request body types", func(t *testing.T) {
		tests := []struct {
			name         string
			input        string
			expectedBody string
			contentType  string
		}{
			{
				name:         "string body",
				input:        `body: "hello world"`,
				expectedBody: "hello world",
			},
			{
				name:         "Blob body",
				input:        `body: new Blob(["hello world"])`,
				expectedBody: "hello world",
			},
			{
				name:         "ArrayBuffer body",
				input:        `body: new ArrayBuffer(1)`,
				expectedBody: "\x00",
			},
			{
				name:         "Uint8Array body",
				input:        `body: new Uint8Array(1)`,
				expectedBody: "\x00",
			},
			{
				name:         "Uint16Array body",
				input:        `body: new Uint16Array(1)`,
				expectedBody: "\x00\x00",
			},
			{
				name:         "DataView body",
				input:        `body: new DataView(new ArrayBuffer(1))`,
				expectedBody: "\x00",
			},
			{
				name:         "JSON body",
				input:        `body: JSON.stringify({message: "hello"})`,
				expectedBody: `{"message":"hello"}`,
				contentType:  "text/plain;charset=UTF-8",
			},
			{
				name:         "FormData body",
				input:        `body: new FormData({"field": "value"})`,
				expectedBody: "value",
				contentType:  "multipart/form-data",
			},
			{
				name:         "URLSearchParams body",
				input:        `body: new URLSearchParams({"query": "test"})`,
				expectedBody: "query=test",
				contentType:  "application/x-www-form-url",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			i, err := strconv.Atoi(r.URL.Query().Get("i"))
			require.NoError(t, err)
			tt := tests[i]
			if tt.contentType != "" {
				assert.Contains(t, r.Header.Get("Content-Type"), tt.contentType)
			}
			body, err := io.ReadAll(r.Body)
			if assert.NoError(t, err) {
				assert.Contains(t, string(body), tt.expectedBody)
			}
		}))
		defer server.Close()

		for i, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := vm.RunModule(ctx, `
				export default async (url) => {
					const response = await fetch(url, {
						method: "POST",
						`+tt.input+`
					});
					return response.status;
				}
				`, fmt.Sprintf("%s?i=%d", server.URL, i))
				require.NoError(t, err)
				assert.Equal(t, int64(200), modulestest.PromiseResult(result).ToInteger())
			})
		}
	})

	t.Run("response body methods", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Header.Get("Accept") {
			case "application/x-www-form-urlencoded":
				w.Header().Set("Content-Type", "application/x-www-form-urlencoded")
				w.Write([]byte(`foo=bar&name=11`))
			default:
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"message":"hello"}`))
			}
		}))
		defer server.Close()
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name: "text",
				input: `
				export default async (url) => {
					const response = await fetch(url);
					return await response.text();
				}`,
				expected: `{"message":"hello"}`,
			},
			{
				name: "json",
				input: `
				export default async (url) => {
					const response = await fetch(url);
					const data = await response.json();
					return data.message;
				}`,
				expected: "hello",
			},
			{
				name: "arrayBuffer",
				input: `
				export default async (url) => {
					const response = await fetch(url);
					const buffer = await response.arrayBuffer();
					return String.fromCharCode.apply(String, new Uint8Array(buffer));
				}`,
				expected: `{"message":"hello"}`,
			},
			{
				name: "formData",
				input: `
				export default async (url) => {
					const response = await fetch(url, { headers: { accept: "application/x-www-form-urlencoded" } });
					return [...(await response.formData()).keys()].sort();
				}`,
				expected: "foo,name",
			},
			{
				name: "body used",
				input: `
				export default async (url) => {
					const response = await fetch(url);
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
				result, err := vm.RunModule(ctx, tt.input, server.URL)
				require.NoError(t, err)
				value := modulestest.PromiseResult(result)
				assert.Contains(t, value.String(), tt.expected)
			})
		}
	})

	t.Run("response headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("X-Custom", "response header")
			w.Write([]byte("ok"))
		}))
		defer server.Close()

		result, err := vm.RunModule(ctx, `
		export default async (url) => {
			const response = await fetch(url);
			return {
				contentType: response.headers.get("content-type"),
				custom: response.headers.get("x-custom"),
			};
		}
		`, server.URL)
		require.NoError(t, err)
		obj := modulestest.PromiseResult(result).ToObject(vm.Runtime())
		assert.Equal(t, "text/plain", obj.Get("contentType").String())
		assert.Equal(t, "response header", obj.Get("custom").String())
	})

	t.Run("error handling", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		result, err := vm.RunModule(ctx, `
		export default async (url) => {
			const response = await fetch(url);
			return {
				ok: response.ok,
				status: response.status,
			};
		}
		`, server.URL)
		require.NoError(t, err)
		obj := modulestest.PromiseResult(result).ToObject(vm.Runtime())
		assert.False(t, obj.Get("ok").ToBoolean())
		assert.Equal(t, int64(404), obj.Get("status").ToInteger())
	})

	t.Run("abort controller", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}))
		defer server.Close()

		result, err := vm.RunModule(ctx, `
		export default async (url) => {
			const controller = new AbortController();
			const promise = fetch(url, { signal: controller.signal });
			controller.abort();
			try {
				await promise;
				return "should not reach here";
			} catch (e) {
				return e.toString();
			}
		}
		`, server.URL)
		require.NoError(t, err)
		assert.Contains(t, modulestest.PromiseResult(result).String(), "aborted")
	})

	t.Run("type error", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
		export default () => fetch()
		`)
		require.NoError(t, err)
		_, err = promise.Result(result)
		assert.ErrorContains(t, err, `TypeError: fetch requires at least 1 argument`)
	})
}
