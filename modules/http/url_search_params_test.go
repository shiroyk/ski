package http

import (
	"context"
	"testing"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURLSearchParams(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t)
	ctx := context.Background()

	t.Run("constructor", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "empty",
				input:    "new URLSearchParams()",
				expected: "",
			},
			{
				name:     "string",
				input:    `new URLSearchParams("foo=1&bar=2")`,
				expected: "foo=1&bar=2",
			},
			{
				name:     "string with ?",
				input:    `new URLSearchParams("?foo=1&bar=2")`,
				expected: "foo=1&bar=2",
			},
			{
				name:     "object",
				input:    `new URLSearchParams({foo: "1", bar: ["2", "3"]})`,
				expected: "foo=1&bar=2&bar=3",
			},
			{
				name:     "URLSearchParams",
				input:    `new URLSearchParams(new URLSearchParams("foo=1&bar=2"))`,
				expected: "foo=1&bar=2",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := vm.RunString(ctx, tt.input+".toString()")
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result.String())
			})
		}
	})

	t.Run("append and get", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const params = new URLSearchParams();
				params.append("a", "1");
				params.append("b", "2");
				params.append("b", "3");
				return {
					a: params.get("a"),
					b: params.get("b"),
					none: params.get("none"),
					all: params.getAll("b"),
					toString: params.toString()
				};
			}
		`)
		require.NoError(t, err)
		obj := result.ToObject(vm.Runtime())
		assert.Equal(t, "1", obj.Get("a").String())
		assert.Equal(t, "2", obj.Get("b").String())
		assert.True(t, sobek.IsNull(obj.Get("none")))
		assert.Equal(t, "2,3", obj.Get("all").String())
		assert.Equal(t, "a=1&b=2&b=3", obj.Get("toString").String())
	})

	t.Run("set and delete", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const params = new URLSearchParams("a=1&b=2&b=3");
				params.set("b", "4");
				params.delete("a");
				return params.toString();
			}
		`)
		require.NoError(t, err)
		assert.Equal(t, "b=4", result.String())
	})

	t.Run("has", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const params = new URLSearchParams("a=1");
				return {
					exists: params.has("a"),
					notExists: params.has("b")
				};
			}
		`)
		require.NoError(t, err)
		obj := result.ToObject(vm.Runtime())
		assert.True(t, obj.Get("exists").ToBoolean())
		assert.False(t, obj.Get("notExists").ToBoolean())
	})

	t.Run("sort", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const params = new URLSearchParams("c=3&a=1&b=2");
				params.sort();
				return params.toString();
			}
		`)
		require.NoError(t, err)
		assert.Equal(t, "a=1&b=2&c=3", result.String())
	})

	t.Run("forEach", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const params = new URLSearchParams("a=1&b=2&b=3");
				const entries = [];
				params.forEach((value, key) => {
					entries.push([key, value]);
				});
				return entries;
			}
		`)
		require.NoError(t, err)
		arr := result.ToObject(vm.Runtime())
		assert.Equal(t, "a,1", arr.Get("0").String())
		assert.Equal(t, "b,2", arr.Get("1").String())
		assert.Equal(t, "b,3", arr.Get("2").String())
	})

	t.Run("iterators", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const params = new URLSearchParams("a=1&b=2");
				return {
					keys: Array.from(params.keys()),
					values: Array.from(params.values()),
					entries: Array.from(params.entries()),
					spread: [...params]
				};
			}
		`)
		require.NoError(t, err)
		obj := result.ToObject(vm.Runtime())
		assert.Equal(t, "a,b", obj.Get("keys").String())
		assert.Equal(t, "1,2", obj.Get("values").String())
		assert.Equal(t, "a,1,b,2", obj.Get("entries").String())
		assert.Equal(t, "a,1,b,2", obj.Get("spread").String())
	})

	t.Run("special characters", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const params = new URLSearchParams();
				params.append("special", "!@#$%^&*()");
				return params.toString();
			}
		`)
		require.NoError(t, err)
		assert.Equal(t, "special=%21%40%23%24%25%5E%26%2A%28%29", result.String())
	})
}
