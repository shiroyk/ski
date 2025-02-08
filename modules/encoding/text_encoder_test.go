package encoding

import (
	"context"
	"testing"

	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextEncoder(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t)
	ctx := context.Background()

	t.Run("constructor", func(t *testing.T) {
		result, err := vm.RunString(ctx, "new TextEncoder()")
		require.NoError(t, err)
		obj := result.ToObject(vm.Runtime())
		assert.Equal(t, "utf-8", obj.Get("encoding").String())
	})

	t.Run("encode", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected []byte
		}{
			{
				name: "ascii text",
				input: `
				export default () => {
					const encoder = new TextEncoder();
					const bytes = encoder.encode("hello");
					return Array.from(new Uint8Array(bytes));
				}
				`,
				expected: []byte("hello"),
			},
			{
				name: "unicode text",
				input: `
				export default () => {
					const encoder = new TextEncoder();
					const bytes = encoder.encode("你好");
					return Array.from(new Uint8Array(bytes));
				}
				`,
				expected: []byte("你好"),
			},
			{
				name: "empty string",
				input: `
				export default () => {
					const encoder = new TextEncoder();
					const bytes = encoder.encode("");
					return Array.from(new Uint8Array(bytes));
				}
				`,
				expected: []byte(""),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := vm.RunModule(ctx, tt.input)
				require.NoError(t, err)
				array := result.Export().([]any)
				bytes := make([]byte, len(array))
				for i, v := range array {
					bytes[i] = byte(v.(int64))
				}
				assert.Equal(t, tt.expected, bytes)
			})
		}
	})

	t.Run("encodeInto", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
		export default () => {
			const encoder = new TextEncoder();
			const text = "hello";
			const u8 = new Uint8Array(10);
			const result = encoder.encodeInto(text, u8);
			return {
				bytes: u8,
				read: result.read,
				written: result.written,
			};
		}
		`)
		require.NoError(t, err)
		obj := result.ToObject(vm.Runtime())
		bytes := obj.Get("bytes").Export().([]byte)
		assert.Equal(t, []byte{104, 101, 108, 108, 111, 0, 0, 0, 0, 0}, bytes)
		assert.Equal(t, int64(5), obj.Get("read").ToInteger())
		assert.Equal(t, int64(5), obj.Get("written").ToInteger())
	})

	t.Run("encodeInto errors", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			msg   string
		}{
			{
				name:  "missing arguments",
				input: "encoder.encodeInto()",
				msg:   "requires 2 arguments",
			},
			{
				name:  "invalid Uint8Array",
				input: `encoder.encodeInto("text", "not a Uint8Array")`,
				msg:   "must be a Uint8Array",
			},
		}
		_, err := vm.RunString(ctx, `const encoder = new TextEncoder();`)
		require.NoError(t, err)

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err = vm.RunString(ctx, tt.input)
				assert.ErrorContains(t, err, tt.msg)
			})
		}
	})
}
