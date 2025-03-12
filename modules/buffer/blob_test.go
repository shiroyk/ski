package buffer

import (
	"context"
	"io"
	"testing"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlob(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t)
	ctx := context.Background()

	t.Run("constructor", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			wantErr  bool
			expected struct {
				size  int64
				type_ string
				data  string
			}
		}{
			{
				name:  "default",
				input: "new Blob()",
			},
			{
				name:  "empty array",
				input: "new Blob([])",
				expected: struct {
					size  int64
					type_ string
					data  string
				}{
					size:  0,
					type_: "",
					data:  "",
				},
			},
			{
				name:  "string content",
				input: `new Blob(["hello world"], { type: "text/plain" })`,
				expected: struct {
					size  int64
					type_ string
					data  string
				}{
					size:  11,
					type_: "text/plain",
					data:  "hello world",
				},
			},
			{
				name:  "array buffer content",
				input: `new Blob([new ArrayBuffer(5)], { type: "application/octet-stream" })`,
				expected: struct {
					size  int64
					type_ string
					data  string
				}{
					size:  5,
					type_: "application/octet-stream",
					data:  "\x00\x00\x00\x00\x00",
				},
			},
			{
				name:  "Uint8Array content",
				input: `new Blob([new Uint8Array(1)], { type: "application/octet-stream" })`,
				expected: struct {
					size  int64
					type_ string
					data  string
				}{
					size:  1,
					type_: "application/octet-stream",
					data:  "\x00",
				},
			},
			{
				name:  "Uint16Array content",
				input: `new Blob([new Uint16Array(1)], { type: "application/octet-stream" })`,
				expected: struct {
					size  int64
					type_ string
					data  string
				}{
					size:  2,
					type_: "application/octet-stream",
					data:  "\x00\x00",
				},
			},
			{
				name:  "BigUint64Array content",
				input: `new Blob([new BigUint64Array(1)], { type: "application/octet-stream" })`,
				expected: struct {
					size  int64
					type_ string
					data  string
				}{
					size:  8,
					type_: "application/octet-stream",
					data:  "\x00\x00\x00\x00\x00\x00\x00\x00",
				},
			},
			{
				name: "multiple parts",
				input: `new Blob([
					"hello",
					new Blob([" ", "world"]),
					new ArrayBuffer(1)
				], { type: "text/plain" })`,
				expected: struct {
					size  int64
					type_ string
					data  string
				}{
					size:  12,
					type_: "text/plain",
					data:  "hello world\x00",
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
				blob := result.Export().(*blob)
				data, err := io.ReadAll(blob.data)
				require.NoError(t, err)
				assert.Equal(t, tt.expected.size, blob.size)
				assert.Equal(t, tt.expected.type_, blob.type_)
				assert.Equal(t, tt.expected.data, string(data))
			})
		}
	})

	t.Run("methods", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default async () => {
				const blob = new Blob(["hello world"], { type: "text/plain" });
				const slice = blob.slice(6);
				const sliceWithType = blob.slice(0, 5, "text/html");
				return {
					size: blob.size,
					type: blob.type,
					text: await blob.text(),
					sliceText: await slice.text(),
					sliceType: slice.type,
					sliceWithTypeText: await sliceWithType.text(),
					sliceWithType: sliceWithType.type,
					arrayBuffer: await blob.arrayBuffer(),
					bytes: await blob.bytes(),
				};
			}
		`)
		require.NoError(t, err)
		obj := modulestest.PromiseResult(result).ToObject(vm.Runtime())
		assert.Equal(t, int64(11), obj.Get("size").ToInteger())
		assert.Equal(t, "text/plain", obj.Get("type").String())
		assert.Equal(t, "hello world", obj.Get("text").String())
		assert.Equal(t, "world", obj.Get("sliceText").String())
		assert.Equal(t, "text/plain", obj.Get("sliceType").String())
		assert.Equal(t, "hello", obj.Get("sliceWithTypeText").String())
		assert.Equal(t, "text/html", obj.Get("sliceWithType").String())
		assert.Equal(t, "hello world", string(obj.Get("arrayBuffer").Export().(sobek.ArrayBuffer).Bytes()))
		assert.Equal(t, "hello world", string(obj.Get("bytes").Export().([]byte)))
	})

	t.Run("slice parameters", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "negative start",
				input:    "blob.slice(-5)",
				expected: "world",
			},
			{
				name:     "negative end",
				input:    "blob.slice(0, -6)",
				expected: "hello",
			},
			{
				name:     "start > end",
				input:    "blob.slice(8, 3)",
				expected: "",
			},
			{
				name:     "out of bounds",
				input:    "blob.slice(0, 100)",
				expected: "hello world",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := vm.RunModule(ctx, `
					export default async () => {
						const blob = new Blob(["hello world"]);
						const slice = `+tt.input+`;
						return slice.text();
					}
				`)
				require.NoError(t, err)
				obj := modulestest.PromiseResult(result).ToObject(vm.Runtime())
				assert.Equal(t, tt.expected, obj.String())
			})
		}
	})
}
