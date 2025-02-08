package http

import (
	"context"
	"testing"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormData(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t)
	ctx := context.Background()

	t.Run("constructor", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected formData
		}{
			{
				name:     "empty",
				input:    "new FormData()",
				expected: formData{data: map[string][]sobek.Value{}},
			},
			{
				name:  "object",
				input: `new FormData({foo: "1"})`,
				expected: formData{keys: []string{"foo"}, data: map[string][]sobek.Value{
					"foo": {vm.Runtime().ToValue("1")},
				}},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := vm.RunString(ctx, tt.input)
				require.NoError(t, err)
				assert.Equal(t, tt.expected, *result.Export().(*formData))
			})
		}
	})

	t.Run("append and get", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const form = new FormData();
				form.append("text", "hello");
				form.append("file", new ArrayBuffer(5), "test.txt");
				form.append("multi", "1");
				form.append("multi", "2");
				return {
					text: form.get("text"),
					file: form.get("file"),
					fileName: form.get("file").name,
					multi: form.get("multi"),
					none: form.get("none"),
					all: form.getAll("multi")
				};
			}
		`)
		require.NoError(t, err)
		obj := result.ToObject(vm.Runtime())
		assert.Equal(t, "hello", obj.Get("text").String())
		assert.Equal(t, "test.txt", obj.Get("fileName").String())
		assert.Equal(t, "1", obj.Get("multi").String())
		assert.True(t, sobek.IsUndefined(obj.Get("none")))
		assert.Equal(t, "1,2", obj.Get("all").String())
	})

	t.Run("set and delete", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const form = new FormData();
				form.append("a", "1");
				form.append("b", "2");
				form.append("b", "3");
				form.set("b", "4");
				form.delete("a");
				return {
					hasA: form.has("a"),
					hasB: form.has("b"),
					b: form.get("b"),
					allB: form.getAll("b")
				};
			}
		`)
		require.NoError(t, err)
		obj := result.ToObject(vm.Runtime())
		assert.False(t, obj.Get("hasA").ToBoolean())
		assert.True(t, obj.Get("hasB").ToBoolean())
		assert.Equal(t, "4", obj.Get("b").String())
		assert.Equal(t, "4", obj.Get("allB").String())
	})

	t.Run("forEach", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const form = new FormData();
				form.append("a", "1");
				form.append("b", "2");
				form.append("b", "3");
				const entries = [];
				form.forEach((value, key) => {
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
				const form = new FormData();
				form.append("a", "1");
				form.append("b", "2");
				return {
					keys: Array.from(form.keys()),
					values: Array.from(form.values()),
					entries: Array.from(form.entries()),
					spread: [...form]
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

	t.Run("binary data", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const form = new FormData();
				const buffer = new ArrayBuffer(5);
				form.append("file1", buffer);
				form.append("file2", buffer, "test.bin");
				return {
					hasFile1: form.has("file1"),
					file1Name: form.get("file1").name,
					hasFile2: form.has("file2"),
					file2Name: form.get("file2").name
				};
			}
		`)
		require.NoError(t, err)
		obj := result.ToObject(vm.Runtime())
		assert.True(t, obj.Get("hasFile1").ToBoolean())
		assert.Equal(t, "blob", obj.Get("file1Name").String())
		assert.True(t, obj.Get("hasFile2").ToBoolean())
		assert.Equal(t, "test.bin", obj.Get("file2Name").String())
	})
}
