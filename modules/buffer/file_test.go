package buffer

import (
	"context"
	"testing"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFile(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t)
	ctx := context.Background()

	t.Run("constructor", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			wantErr  bool
			expected struct {
				name         string
				size         int64
				type_        string
				lastModified int64
			}
		}{
			{
				name:    "missing arguments",
				input:   "new File()",
				wantErr: true,
			},
			{
				name:    "missing filename",
				input:   "new File(['hello world'])",
				wantErr: true,
			},
			{
				name:  "string content",
				input: `new File(["hello world"], "test.txt", { type: "text/plain" })`,
				expected: struct {
					name         string
					size         int64
					type_        string
					lastModified int64
				}{
					name:  "test.txt",
					size:  11,
					type_: "text/plain",
				},
			},
			{
				name:  "array buffer content",
				input: `new File([new ArrayBuffer(5)], "test.bin", { type: "application/octet-stream" })`,
				expected: struct {
					name         string
					size         int64
					type_        string
					lastModified int64
				}{
					name:  "test.bin",
					size:  5,
					type_: "application/octet-stream",
				},
			},
			{
				name: "with lastModified",
				input: `new File(["hello"], "test.txt", { 
					type: "text/plain",
					lastModified: 1234567890000
				})`,
				expected: struct {
					name         string
					size         int64
					type_        string
					lastModified int64
				}{
					name:         "test.txt",
					size:         5,
					type_:        "text/plain",
					lastModified: 1234567890000,
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := vm.RunString(ctx, tt.input)
				if tt.wantErr {
					assert.Error(t, err)
					return
				}
				require.NoError(t, err)
				obj := result.ToObject(vm.Runtime())
				blob := toBlob(vm.Runtime(), obj)
				assert.Equal(t, tt.expected.name, obj.Get("name").String())
				assert.Equal(t, tt.expected.size, blob.size)
				assert.Equal(t, tt.expected.type_, blob.type_)
				if tt.expected.lastModified > 0 {
					assert.Equal(t, tt.expected.lastModified, obj.Get("lastModified").ToInteger())
				} else {
					assert.Greater(t, obj.Get("lastModified").ToInteger(), int64(1600000000000))
				}
			})
		}
	})

	t.Run("blob methods", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
console.log(File instanceof Function);
			export default async () => {
				const file = new File(["hello world"], "test.txt", { type: "text/plain" });
				const slice = file.slice(0, 5);
				return {
					text: await file.text(),
					sliceText: await slice.text(),
					arrayBuffer: await file.arrayBuffer(),
					type: slice.type,
				};
			}
		`)
		require.NoError(t, err)
		obj := modulestest.PromiseResult(result).ToObject(vm.Runtime())
		assert.Equal(t, "hello world", obj.Get("text").String())
		assert.Equal(t, "hello", obj.Get("sliceText").String())
		assert.Equal(t, 11, len(obj.Get("arrayBuffer").Export().(sobek.ArrayBuffer).Bytes()))
		assert.Equal(t, "", obj.Get("type").String())
	})

	t.Run("webkitRelativePath", func(t *testing.T) {
		result, err := vm.RunString(ctx, `
			const file = new File([""], "file.txt");
			file.webkitRelativePath
		`)
		require.NoError(t, err)
		assert.Equal(t, "", result.String())
	})
}
