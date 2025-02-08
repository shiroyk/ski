package encoding

import (
	"context"
	"testing"

	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextDecoder(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t)
	ctx := context.Background()

	t.Run("constructor", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			wantErr  bool
			expected struct {
				encoding  string
				fatal     bool
				ignoreBOM bool
			}
		}{
			{
				name:  "default options",
				input: "new TextDecoder()",
				expected: struct {
					encoding  string
					fatal     bool
					ignoreBOM bool
				}{
					encoding:  "utf-8",
					fatal:     false,
					ignoreBOM: false,
				},
			},
			{
				name:  "with encoding",
				input: `new TextDecoder("gbk")`,
				expected: struct {
					encoding  string
					fatal     bool
					ignoreBOM bool
				}{
					encoding:  "gbk",
					fatal:     false,
					ignoreBOM: false,
				},
			},
			{
				name:  "with options",
				input: `new TextDecoder("utf-8", { fatal: true, ignoreBOM: true })`,
				expected: struct {
					encoding  string
					fatal     bool
					ignoreBOM bool
				}{
					encoding:  "utf-8",
					fatal:     true,
					ignoreBOM: true,
				},
			},
			{
				name:    "unsupported encoding",
				input:   `new TextDecoder("invalid")`,
				wantErr: true,
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
				assert.Equal(t, tt.expected.encoding, obj.Get("encoding").String())
				assert.Equal(t, tt.expected.fatal, obj.Get("fatal").ToBoolean())
				assert.Equal(t, tt.expected.ignoreBOM, obj.Get("ignoreBOM").ToBoolean())
			})
		}
	})

	t.Run("decode", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
			wantErr  bool
		}{
			{
				name: "utf-8 text",
				input: `
				export default () => {
					const decoder = new TextDecoder();
					const bytes = new Uint8Array([104, 101, 108, 108, 111]);
					return decoder.decode(bytes);
				}
				`,
				expected: "hello",
			},
			{
				name: "utf-8 with BOM",
				input: `
				export default () => {
					const decoder = new TextDecoder();
					const bytes = new Uint8Array([0xEF, 0xBB, 0xBF, 104, 101, 108, 108, 111]);
					return decoder.decode(bytes);
				}
				`,
				expected: "hello",
			},
			{
				name: "utf-8 with BOM ignored",
				input: `
				export default () => {
					const decoder = new TextDecoder("utf-8", { ignoreBOM: true });
					const bytes = new Uint8Array([0xEF, 0xBB, 0xBF, 104, 101, 108, 108, 111]);
					return decoder.decode(bytes);
				}
				`,
				expected: "\ufeffhello",
			},
			{
				name: "empty input",
				input: `
				export default () => {
					const decoder = new TextDecoder();
					return decoder.decode();
				}
				`,
				expected: "",
			},
			{
				name: "gbk text",
				input: `
				export default () => {
					const decoder = new TextDecoder("gbk");
					const bytes = new Uint8Array([0xC4, 0xE3, 0xBA, 0xC3]); // 你好
					return decoder.decode(bytes);
				}
				`,
				expected: "你好",
			},
			{
				name: "invalid utf-8 with fatal",
				input: `
				export default () => {
					const decoder = new TextDecoder("utf-8", { fatal: true });
					const bytes = new Uint8Array([0xFF, 0xFF]);
					return decoder.decode(bytes);
				}
				`,
				wantErr: true,
			},
			{
				name: "invalid utf-8 without fatal",
				input: `
				export default () => {
					const decoder = new TextDecoder("utf-8", { fatal: false });
					const bytes = new Uint8Array([0xFF, 0xFF]);
					return decoder.decode(bytes);
				}
				`,
				expected: "\xff\xff",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := vm.RunModule(ctx, tt.input)
				if tt.wantErr {
					assert.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result.String())
			})
		}
	})

	t.Run("decode array buffer", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
		export default () => {
			const decoder = new TextDecoder();
			const buffer = new ArrayBuffer(5);
			const view = new Uint8Array(buffer);
			view.set([104, 101, 108, 108, 111]);
			return decoder.decode(buffer);
		}
		`)
		require.NoError(t, err)
		assert.Equal(t, "hello", result.String())
	})

	t.Run("invalid input type", func(t *testing.T) {
		_, err := vm.RunString(ctx, `
			const decoder = new TextDecoder();
			decoder.decode("not a buffer");
		`)
		assert.Error(t, err)
	})
}
