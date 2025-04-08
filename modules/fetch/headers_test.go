package fetch

import (
	"context"
	"testing"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeaders(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t)
	ctx := context.Background()

	t.Run("constructor", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected Header
		}{
			{
				name:     "empty",
				input:    "new Headers()",
				expected: Header{},
			},
			{
				name:  "object",
				input: `new Headers({"Content-Type": "text/plain", "X-Custom": "value"})`,
				expected: Header{
					"content-type": {"text/plain"},
					"x-custom":     {"value"},
				},
			},
			{
				name:  "from headers",
				input: `new Headers(new Headers({"Content-Type": "text/plain"}))`,
				expected: Header{
					"content-type": {"text/plain"},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := vm.RunString(ctx, tt.input)
				require.NoError(t, err)
				headers := result.Export().(Header)
				assert.Equal(t, tt.expected, headers)
			})
		}
	})

	t.Run("append and get", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const headers = new Headers();
				headers.append("Accept", "text/html");
				headers.append("Accept", "application/xhtml+xml");
				return {
					single: headers.get("content-type"),
					multiple: headers.get("accept"),
					missing: headers.get("not-exists")
				};
			}
		`)
		require.NoError(t, err)
		obj := result.ToObject(vm.Runtime())
		assert.True(t, sobek.IsNull(obj.Get("single")))
		assert.Equal(t, "text/html, application/xhtml+xml", obj.Get("multiple").String())
		assert.True(t, sobek.IsNull(obj.Get("missing")))
	})

	t.Run("set", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const headers = new Headers();
				headers.append("Accept", "text/html");
				headers.append("Accept", "application/xhtml+xml");
				headers.set("Accept", "text/plain");
				return headers.get("accept");
			}
		`)
		require.NoError(t, err)
		assert.Equal(t, "text/plain", result.String())
	})

	t.Run("has and delete", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const headers = new Headers();
				headers.set("Content-Type", "text/plain");
				const hasBeforeDelete = headers.has("content-type");
				headers.delete("content-type");
				return {
					hasBeforeDelete,
					hasAfterDelete: headers.has("content-type")
				};
			}
		`)
		require.NoError(t, err)
		obj := result.ToObject(vm.Runtime())
		assert.True(t, obj.Get("hasBeforeDelete").ToBoolean())
		assert.False(t, obj.Get("hasAfterDelete").ToBoolean())
	})

	t.Run("forEach", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const headers = new Headers({
					"Content-Type": "text/plain",
					"X-Custom": "value"
				});
				const entries = [];
				headers.forEach((value, key) => {
					entries.push([key, value]);
				});
				return entries.sort();
			}
		`)
		require.NoError(t, err)
		arr := result.ToObject(vm.Runtime())
		assert.Equal(t, int64(2), arr.Get("length").ToInteger())
		assert.Equal(t, "content-type,text/plain", arr.Get("0").String())
		assert.Equal(t, "x-custom,value", arr.Get("1").String())
	})

	t.Run("iterators", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const headers = new Headers({
					"Content-Type": "text/plain",
					"X-Custom": "value"
				});
				return {
					keys: Array.from(headers.keys()).sort(),
					values: Array.from(headers.values()).sort(),
					entries: Array.from(headers.entries()).sort(),
					spread: [...headers].sort()
				};
			}
		`)
		require.NoError(t, err)
		obj := result.ToObject(vm.Runtime())
		assert.Equal(t, "content-type,x-custom", obj.Get("keys").String())
		assert.Equal(t, "text/plain,value", obj.Get("values").String())
		assert.Equal(t, "content-type,text/plain,x-custom,value", obj.Get("entries").String())
		assert.Equal(t, "content-type,text/plain,x-custom,value", obj.Get("spread").String())
	})

	t.Run("case insensitive", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const headers = new Headers();
				headers.set("Content-Type", "text/plain");
				return {
					normalCase: headers.get("Content-Type"),
					lowerCase: headers.get("content-type"),
					upperCase: headers.get("CONTENT-TYPE")
				};
			}
		`)
		require.NoError(t, err)
		obj := result.ToObject(vm.Runtime())
		assert.Equal(t, "text/plain", obj.Get("normalCase").String())
		assert.Equal(t, "text/plain", obj.Get("lowerCase").String())
		assert.Equal(t, "text/plain", obj.Get("upperCase").String())
	})
}
