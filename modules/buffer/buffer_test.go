package buffer

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuffer(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t)
	ctx := context.Background()

	t.Run("constructor", func(t *testing.T) {
		tests := []struct {
			name   string
			input  string
			length int
			data   string
		}{
			{
				name:   "size",
				input:  "new Buffer(2);",
				length: 2,
				data:   "\x00\x00",
			},
			{
				name:   "string",
				input:  `new Buffer("hello");`,
				length: 5,
				data:   "hello",
			},
			{
				name:   "string base64",
				input:  `new Buffer("Y2lhbGxv");`,
				length: 8,
				data:   "Y2lhbGxv",
			},
			{
				name:   "string hex",
				input:  `new Buffer("6162", "hex");`,
				length: 2,
				data:   "ab",
			},
			{
				name:   "buffer",
				input:  `new Buffer(new Buffer(2));`,
				length: 2,
				data:   "\x00\x00",
			},
			{
				name:   "ArrayBuffer",
				input:  `new Buffer(new ArrayBuffer(2));`,
				length: 2,
				data:   "\x00\x00",
			},
			{
				name:   "Uint8Array",
				input:  `new Buffer(new Uint8Array(1));`,
				length: 1,
				data:   "\x00",
			},
			{
				name:   "Uint32Array",
				input:  `new Buffer(new Uint32Array(1));`,
				length: 1,
				data:   "\x00",
			},
			{
				name:   "Array",
				input:  `new Buffer([61, 62]);`,
				length: 2,
				data:   "=>",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := vm.RunString(ctx, tt.input)
				require.NoError(t, err)
				data := result.Export().([]byte)
				assert.Equal(t, tt.length, len(data))
				assert.Equal(t, tt.data, string(data))
			})
		}
	})

	t.Run("static methods", func(t *testing.T) {
		t.Run("from", func(t *testing.T) {
			tests := []struct {
				name   string
				input  string
				length int
				data   string
			}{
				{
					name:   "string",
					input:  `Buffer.from("hello")`,
					length: 5,
					data:   "hello",
				},
				{
					name:   "string hex",
					input:  `Buffer.from("68656c6c6f", "hex")`,
					length: 5,
					data:   "hello",
				},
				{
					name:   "string base64",
					input:  `Buffer.from("aGVsbG8=", "base64")`,
					length: 5,
					data:   "hello",
				},
				{
					name:   "buffer",
					input:  `Buffer.from(Buffer.from("test"))`,
					length: 4,
					data:   "test",
				},
				{
					name:   "array",
					input:  `Buffer.from([104, 101, 108, 108, 111])`,
					length: 5,
					data:   "hello",
				},
				{
					name: "ArrayBuffer",
					input: `
							Buffer.from(new Uint8Array([104, 101, 108, 108, 111]).buffer)
					`,
					length: 5,
					data:   "hello",
				},
				{
					name:   "Uint8Array",
					input:  `Buffer.from(new Uint8Array([104, 101, 108, 108, 111]))`,
					length: 5,
					data:   "hello",
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result, err := vm.RunString(ctx, tt.input)
					require.NoError(t, err)
					data := result.Export().([]byte)
					assert.Equal(t, tt.length, len(data))
					assert.Equal(t, tt.data, string(data))
				})
			}
		})

		t.Run("alloc", func(t *testing.T) {
			tests := []struct {
				name   string
				input  string
				length int
				data   string
			}{
				{
					name:   "zero filled",
					input:  `Buffer.alloc(5)`,
					length: 5,
					data:   "\x00\x00\x00\x00\x00",
				},
				{
					name:   "fill with number",
					input:  `Buffer.alloc(3, 1)`,
					length: 3,
					data:   "\x01\x01\x01",
				},
				{
					name:   "fill with string",
					input:  `Buffer.alloc(6, "ab")`,
					length: 6,
					data:   "ababab",
				},
				{
					name:   "fill with encoding",
					input:  `Buffer.alloc(2, "6162", "hex")`,
					length: 2,
					data:   "ab",
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result, err := vm.RunString(ctx, tt.input)
					require.NoError(t, err)
					data := result.Export().([]byte)
					assert.Equal(t, tt.length, len(data))
					assert.Equal(t, tt.data, string(data))
				})
			}
		})

		t.Run("byteLength", func(t *testing.T) {
			tests := []struct {
				name     string
				input    string
				expected int64
			}{
				{
					name:     "string utf8",
					input:    `Buffer.byteLength("hello")`,
					expected: 5,
				},
				{
					name:     "string hex",
					input:    `Buffer.byteLength("68656c6c6f", "hex")`,
					expected: 5,
				},
				{
					name:     "string base64",
					input:    `Buffer.byteLength("aGVsbG8=", "base64")`,
					expected: 5,
				},
				{
					name:     "buffer",
					input:    `Buffer.byteLength(Buffer.from("test"))`,
					expected: 4,
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result, err := vm.RunString(ctx, tt.input)
					require.NoError(t, err)
					assert.Equal(t, tt.expected, result.ToInteger())
				})
			}
		})

		t.Run("compare", func(t *testing.T) {
			tests := []struct {
				name     string
				input    string
				expected int64
			}{
				{
					name: "equal",
					input: `
						Buffer.compare(Buffer.from("hello"), Buffer.from("hello"));
					`,
					expected: 0,
				},
				{
					name: "less than",
					input: `
						Buffer.compare(Buffer.from("hello"), Buffer.from("world"));
					`,
					expected: -1,
				},
				{
					name: "greater than",
					input: `
						Buffer.compare(Buffer.from("world"), Buffer.from("hello"));
					`,
					expected: 1,
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result, err := vm.RunString(ctx, tt.input)
					require.NoError(t, err)
					assert.Equal(t, tt.expected, result.ToInteger())
				})
			}
		})

		t.Run("concat", func(t *testing.T) {
			tests := []struct {
				name   string
				input  string
				length int
				data   string
			}{
				{
					name: "basic concat",
					input: `
						Buffer.concat([Buffer.from("hello"), Buffer.from("world")])
					`,
					length: 10,
					data:   "helloworld",
				},
				{
					name: "with total length",
					input: `
						Buffer.concat([Buffer.from("hello"), Buffer.from("world")], 8)
					`,
					length: 8,
					data:   "hellowor",
				},
				{
					name:   "empty array",
					input:  `Buffer.concat([])`,
					length: 0,
					data:   "",
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result, err := vm.RunString(ctx, tt.input)
					require.NoError(t, err)
					data := result.Export().([]byte)
					assert.Equal(t, tt.length, len(data))
					assert.Equal(t, tt.data, string(data))
				})
			}
		})

		t.Run("isBuffer", func(t *testing.T) {
			tests := []struct {
				name     string
				input    string
				expected bool
			}{
				{
					name:     "buffer",
					input:    `Buffer.isBuffer(Buffer.from("test"))`,
					expected: true,
				},
				{
					name:     "string",
					input:    `Buffer.isBuffer("test")`,
					expected: false,
				},
				{
					name:     "array",
					input:    `Buffer.isBuffer([])`,
					expected: false,
				},
				{
					name:     "typed array",
					input:    `Buffer.isBuffer(new Uint8Array())`,
					expected: false,
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result, err := vm.RunString(ctx, tt.input)
					require.NoError(t, err)
					assert.Equal(t, tt.expected, result.ToBoolean())
				})
			}
		})
	})

	t.Run("read methods", func(t *testing.T) {
		t.Run("read integers", func(t *testing.T) {
			tests := []struct {
				name     string
				script   string
				expected int64
			}{
				{
					name:     "readInt8 positive",
					script:   `Buffer.from([0x12]).readInt8(0)`,
					expected: 0x12,
				},
				{
					name:     "readInt8 negative",
					script:   `Buffer.from([0xFF]).readInt8(0)`,
					expected: -1,
				},
				{
					name:     "readUInt8",
					script:   `Buffer.from([0xFF]).readUInt8(0)`,
					expected: 255,
				},
				{
					name:     "readInt16BE",
					script:   `Buffer.from([0x12, 0x34]).readInt16BE(0)`,
					expected: 0x1234,
				},
				{
					name:     "readInt16LE",
					script:   `Buffer.from([0x34, 0x12]).readInt16LE(0)`,
					expected: 0x1234,
				},
				{
					name:     "readUInt16BE",
					script:   `Buffer.from([0xFF, 0xFF]).readUInt16BE(0)`,
					expected: 65535,
				},
				{
					name:     "readUInt16LE",
					script:   `Buffer.from([0xFF, 0xFF]).readUInt16LE(0)`,
					expected: 65535,
				},
				{
					name:     "readInt32BE",
					script:   `Buffer.from([0x12, 0x34, 0x56, 0x78]).readInt32BE(0)`,
					expected: 0x12345678,
				},
				{
					name:     "readInt32LE",
					script:   `Buffer.from([0x78, 0x56, 0x34, 0x12]).readInt32LE(0)`,
					expected: 0x12345678,
				},
				{
					name:     "readUInt32BE",
					script:   `Buffer.from([0xFF, 0xFF, 0xFF, 0xFF]).readUInt32BE(0)`,
					expected: 4294967295,
				},
				{
					name:     "readUInt32LE",
					script:   `Buffer.from([0xFF, 0xFF, 0xFF, 0xFF]).readUInt32LE(0)`,
					expected: 4294967295,
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result, err := vm.RunString(ctx, tt.script)
					require.NoError(t, err)
					assert.Equal(t, tt.expected, result.ToInteger())
				})
			}
		})

		t.Run("read big integers", func(t *testing.T) {
			tests := []struct {
				name     string
				script   string
				expected string
			}{
				{
					name:     "readBigInt64BE positive",
					script:   `Buffer.from([0x00, 0x00, 0x00, 0x00, 0x12, 0x34, 0x56, 0x78]).readBigInt64BE(0)`,
					expected: "305419896",
				},
				{
					name:     "readBigInt64BE negative",
					script:   `Buffer.from([0xFF, 0xFF, 0xFF, 0xFF, 0xED, 0xCB, 0xA9, 0x88]).readBigInt64BE(0)`,
					expected: "-305419896",
				},
				{
					name:     "readBigInt64BE max",
					script:   `Buffer.from([0x7F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF]).readBigInt64BE(0)`,
					expected: "9223372036854775807",
				},
				{
					name:     "readBigInt64BE min",
					script:   `Buffer.from([0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00]).readBigInt64BE(0)`,
					expected: "-9223372036854775808",
				},
				{
					name:     "readBigInt64LE positive",
					script:   `Buffer.from([0x78, 0x56, 0x34, 0x12, 0x00, 0x00, 0x00, 0x00]).readBigInt64LE(0)`,
					expected: "305419896",
				},
				{
					name:     "readBigInt64LE negative",
					script:   `Buffer.from([0x88, 0xA9, 0xCB, 0xED, 0xFF, 0xFF, 0xFF, 0xFF]).readBigInt64LE(0)`,
					expected: "-305419896",
				},
				{
					name:     "readBigUInt64BE zero",
					script:   `Buffer.from([0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00]).readBigUInt64BE(0)`,
					expected: "0",
				},
				{
					name:     "readBigUInt64BE max",
					script:   `Buffer.from([0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF]).readBigUInt64BE(0)`,
					expected: "18446744073709551615",
				},
				{
					name:     "readBigUInt64LE zero",
					script:   `Buffer.from([0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00]).readBigUInt64LE(0)`,
					expected: "0",
				},
				{
					name:     "readBigUInt64LE max",
					script:   `Buffer.from([0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF]).readBigUInt64LE(0)`,
					expected: "18446744073709551615",
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result, err := vm.RunString(ctx, tt.script)
					require.NoError(t, err)
					assert.Equal(t, tt.expected, result.String())
				})
			}
		})

		t.Run("read variable length integers", func(t *testing.T) {
			tests := []struct {
				name     string
				script   string
				method   string
				expected int64
			}{
				{
					name:     "readIntBE 1 byte",
					script:   `Buffer.from([0x12]).readIntBE(0, 1)`,
					expected: 0x12,
				},
				{
					name:     "readIntBE 2 bytes",
					script:   `Buffer.from([0x12, 0x34]).readIntBE(0, 2)`,
					expected: 0x1234,
				},
				{
					name:     "readIntBE 3 bytes",
					script:   `Buffer.from([0x12, 0x34, 0x56]).readIntBE(0, 3)`,
					expected: 0x123456,
				},
				{
					name:     "readIntLE 1 byte",
					script:   `Buffer.from([0x12]).readIntLE(0, 1)`,
					expected: 0x12,
				},
				{
					name:     "readIntLE 2 bytes",
					script:   `Buffer.from([0x34, 0x12]).readIntLE(0, 2)`,
					expected: 0x1234,
				},
				{
					name:     "readIntLE 3 bytes",
					script:   `Buffer.from([0x56, 0x34, 0x12]).readIntLE(0, 3)`,
					expected: 0x123456,
				},
				{
					name:     "readUIntBE 1 byte",
					script:   `Buffer.from([0xFF]).readUIntBE(0, 1)`,
					expected: 255,
				},
				{
					name:     "readUIntBE 2 bytes",
					script:   `Buffer.from([0xFF, 0xFF]).readUIntBE(0, 2)`,
					expected: 65535,
				},
				{
					name:     "readUIntLE 1 byte",
					script:   `Buffer.from([0xFF]).readUIntLE(0, 1)`,
					expected: 255,
				},
				{
					name:     "readUIntLE 2 bytes",
					script:   `Buffer.from([0xFF, 0xFF]).readUIntLE(0, 2)`,
					expected: 65535,
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result, err := vm.RunString(ctx, tt.script)
					require.NoError(t, err)
					assert.Equal(t, tt.expected, result.ToInteger())
				})
			}
		})

		t.Run("read floating point", func(t *testing.T) {
			tests := []struct {
				name     string
				script   string
				expected float64
				delta    float64
			}{
				{
					name: "readDoubleBE",
					script: `
						Buffer.from("400921fb54442eea", "hex").readDoubleBE(0)`,
					expected: 3.14159265359,
					delta:    0.0000000001,
				},
				{
					name: "readDoubleLE",
					script: `
						Buffer.from("ea2e4454fb210940", "hex").readDoubleLE(0)`,
					expected: 3.14159265359,
					delta:    0.0000000001,
				},
				{
					name: "readFloatBE",
					script: `
						Buffer.from("4048f5c3", "hex").readFloatBE(0)`,
					expected: 3.14,
					delta:    0.0001,
				},
				{
					name: "readFloatLE",
					script: `
						Buffer.from("c3f54840", "hex").readFloatLE(0);
					`,
					expected: 3.14,
					delta:    0.0001,
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result, err := vm.RunString(ctx, tt.script)
					require.NoError(t, err)
					assert.InDelta(t, tt.expected, result.ToFloat(), tt.delta)
				})
			}
		})

		t.Run("read errors", func(t *testing.T) {
			tests := []struct {
				name   string
				script string
			}{
				{
					name: "negative offset",
					script: `
						Buffer.alloc(1).readInt8(-1);
					`,
				},
				{
					name: "offset out of bounds",
					script: `
						Buffer.alloc(1).readInt8(2);
					`,
				},
				{
					name: "not enough bytes",
					script: `
						Buffer.alloc(1).readInt16BE(0);
					`,
				},
				{
					name: "invalid byteLength",
					script: `
						Buffer.alloc(4).readIntBE(0, 7);
					`,
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					_, err := vm.RunString(ctx, tt.script)
					assert.Error(t, err)
				})
			}
		})
	})

	t.Run("write methods", func(t *testing.T) {
		t.Run("write integers", func(t *testing.T) {
			tests := []struct {
				name     string
				method   string
				value    string
				expected []byte
			}{
				{
					name:     "writeInt8 positive",
					method:   "writeInt8(0x12, 0)",
					expected: []byte{0x12},
				},
				{
					name:     "writeInt8 negative",
					method:   "writeInt8(-1, 0)",
					expected: []byte{0xFF},
				},
				{
					name:     "writeUInt8",
					method:   "writeUInt8(255, 0)",
					expected: []byte{0xFF},
				},
				{
					name:     "writeInt16BE",
					method:   "writeInt16BE(0x1234, 0)",
					expected: []byte{0x12, 0x34},
				},
				{
					name:     "writeInt16LE",
					method:   "writeInt16LE(0x1234, 0)",
					expected: []byte{0x34, 0x12},
				},
				{
					name:     "writeUInt16BE",
					method:   "writeUInt16BE(0xFFFF, 0)",
					expected: []byte{0xFF, 0xFF},
				},
				{
					name:     "writeUInt16LE",
					method:   "writeUInt16LE(0xFFFF, 0)",
					expected: []byte{0xFF, 0xFF},
				},
				{
					name:     "writeInt32BE",
					method:   "writeInt32BE(0x12345678, 0)",
					expected: []byte{0x12, 0x34, 0x56, 0x78},
				},
				{
					name:     "writeInt32LE",
					method:   "writeInt32LE(0x12345678, 0)",
					expected: []byte{0x78, 0x56, 0x34, 0x12},
				},
				{
					name:     "writeUInt32BE",
					method:   "writeUInt32BE(0xFFFFFFFF, 0)",
					expected: []byte{0xFF, 0xFF, 0xFF, 0xFF},
				},
				{
					name:     "writeUInt32LE",
					method:   "writeUInt32LE(0xFFFFFFFF, 0)",
					expected: []byte{0xFF, 0xFF, 0xFF, 0xFF},
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					script := fmt.Sprintf(`
						var buf = Buffer.alloc(%d);
						buf.%s;
						buf
					`, len(tt.expected), tt.method)
					result, err := vm.RunString(ctx, script)
					require.NoError(t, err)
					data := result.Export().([]byte)
					assert.Equal(t, tt.expected, data)
				})
			}
		})

		t.Run("write big integers", func(t *testing.T) {
			tests := []struct {
				name     string
				method   string
				value    string
				expected []byte
			}{
				{
					name:     "writeBigInt64BE positive",
					method:   "writeBigInt64BE(BigInt('305419896'), 0)",
					expected: []byte{0x00, 0x00, 0x00, 0x00, 0x12, 0x34, 0x56, 0x78},
				},
				{
					name:     "writeBigInt64BE negative",
					method:   "writeBigInt64BE(BigInt('-305419896'), 0)",
					expected: []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xED, 0xCB, 0xA9, 0x88},
				},
				{
					name:     "writeBigInt64BE max",
					method:   "writeBigInt64BE(BigInt('9223372036854775807'), 0)",
					expected: []byte{0x7F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
				},
				{
					name:     "writeBigInt64LE positive",
					method:   "writeBigInt64LE(BigInt('305419896'), 0)",
					expected: []byte{0x78, 0x56, 0x34, 0x12, 0x00, 0x00, 0x00, 0x00},
				},
				{
					name:     "writeBigUInt64BE",
					method:   "writeBigUInt64BE(BigInt('18446744073709551615'), 0)",
					expected: []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
				},
				{
					name:     "writeBigUInt64LE",
					method:   "writeBigUInt64LE(BigInt('18446744073709551615'), 0)",
					expected: []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					script := fmt.Sprintf(`
						var buf = Buffer.alloc(8);
						buf.%s;
						buf
					`, tt.method)
					result, err := vm.RunString(ctx, script)
					require.NoError(t, err)
					data := result.Export().([]byte)
					assert.Equal(t, tt.expected, data)
				})
			}
		})

		t.Run("write variable length integers", func(t *testing.T) {
			tests := []struct {
				name     string
				method   string
				value    string
				expected []byte
			}{
				{
					name:     "writeIntBE 1 byte",
					method:   "writeIntBE(0x12, 0, 1)",
					expected: []byte{0x12},
				},
				{
					name:     "writeIntBE 2 bytes",
					method:   "writeIntBE(0x1234, 0, 2)",
					expected: []byte{0x12, 0x34},
				},
				{
					name:     "writeIntBE 3 bytes",
					method:   "writeIntBE(0x123456, 0, 3)",
					expected: []byte{0x12, 0x34, 0x56},
				},
				{
					name:     "writeIntLE 1 byte",
					method:   "writeIntLE(0x12, 0, 1)",
					expected: []byte{0x12},
				},
				{
					name:     "writeIntLE 2 bytes",
					method:   "writeIntLE(0x1234, 0, 2)",
					expected: []byte{0x34, 0x12},
				},
				{
					name:     "writeIntLE 3 bytes",
					method:   "writeIntLE(0x123456, 0, 3)",
					expected: []byte{0x56, 0x34, 0x12},
				},
				{
					name:     "writeUIntBE 1 byte",
					method:   "writeUIntBE(0xFF, 0, 1)",
					expected: []byte{0xFF},
				},
				{
					name:     "writeUIntBE 2 bytes",
					method:   "writeUIntBE(0xFFFF, 0, 2)",
					expected: []byte{0xFF, 0xFF},
				},
				{
					name:     "writeUIntLE 1 byte",
					method:   "writeUIntLE(0xFF, 0, 1)",
					expected: []byte{0xFF},
				},
				{
					name:     "writeUIntLE 2 bytes",
					method:   "writeUIntLE(0xFFFF, 0, 2)",
					expected: []byte{0xFF, 0xFF},
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					script := fmt.Sprintf(`
						var buf = Buffer.alloc(%d);
						buf.%s;
						buf
					`, len(tt.expected), tt.method)
					result, err := vm.RunString(ctx, script)
					require.NoError(t, err)
					data := result.Export().([]byte)
					assert.Equal(t, tt.expected, data)
				})
			}
		})

		t.Run("write floating point", func(t *testing.T) {
			tests := []struct {
				name   string
				method string
				value  float64
				read   string
				delta  float64
			}{
				{
					name:   "writeFloatBE",
					method: "writeFloatBE",
					value:  3.14,
					read:   "readFloatBE",
					delta:  0.0001,
				},
				{
					name:   "writeFloatBE Inf",
					method: "writeFloatBE",
					value:  math.Inf(0),
					read:   "readFloatBE",
				},
				{
					name:   "writeFloatBE Nan",
					method: "writeFloatBE",
					value:  math.NaN(),
					read:   "readFloatBE",
					delta:  0.0001,
				},
				{
					name:   "writeFloatLE",
					method: "writeFloatLE",
					value:  3.14,
					read:   "readFloatLE",
					delta:  0.0001,
				},
				{
					name:   "writeFloatLE Inf",
					method: "writeFloatLE",
					value:  math.Inf(0),
					read:   "readFloatLE",
				},
				{
					name:   "writeFloatLE NaN",
					method: "writeFloatLE",
					value:  math.NaN(),
					read:   "readFloatLE",
				},
				{
					name:   "writeDoubleBE",
					method: "writeDoubleBE",
					value:  3.14159265359,
					read:   "readDoubleBE",
					delta:  0.0000000001,
				},
				{
					name:   "writeDoubleBE Inf",
					method: "writeDoubleBE",
					value:  math.Inf(0),
					read:   "readDoubleBE",
				},
				{
					name:   "writeDoubleBE NaN",
					method: "writeDoubleBE",
					value:  math.NaN(),
					read:   "readDoubleBE",
				},
				{
					name:   "writeDoubleLE",
					method: "writeDoubleLE",
					value:  3.14159265359,
					read:   "readDoubleLE",
					delta:  0.0000000001,
				},
				{
					name:   "writeDoubleLE Inf",
					method: "writeDoubleLE",
					value:  math.Inf(0),
					read:   "readDoubleLE",
				},
				{
					name:   "writeDoubleLE NaN",
					method: "writeDoubleLE",
					value:  math.NaN(),
					read:   "readDoubleLE",
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					size := 4
					if strings.Contains(tt.method, "Double") {
						size = 8
					}
					value := strconv.FormatFloat(tt.value, 'g', -1, 64)
					switch value {
					case "-Inf":
						value = "-Infinity"
					case "+Inf":
						value = "Infinity"
					}
					script := fmt.Sprintf(`
						var buf = Buffer.alloc(%d);
						buf.%s(%v, 0);
						buf.%s(0);
					`, size, tt.method, value, tt.read)
					result, err := vm.RunString(ctx, script)
					require.NoError(t, err)
					assert.InDelta(t, tt.value, result.ToFloat(), tt.delta)
				})
			}
		})

		t.Run("write value range checks", func(t *testing.T) {
			tests := []struct {
				name   string
				script string
			}{
				{
					name: "writeInt8 too large",
					script: `
						Buffer.alloc(1).writeInt8(128, 0);
					`,
				},
				{
					name: "writeInt8 too small",
					script: `
						Buffer.alloc(1).writeInt8(-129, 0);
					`,
				},
				{
					name: "writeUInt8 negative",
					script: `
						Buffer.alloc(1).writeUInt8(-1, 0);
					`,
				},
				{
					name: "writeUInt8 too large",
					script: `
						Buffer.alloc(1).writeUInt8(256, 0);
					`,
				},
				{
					name: "writeInt16BE too large",
					script: `
						Buffer.alloc(2).writeInt16BE(32768, 0);
					`,
				},
				{
					name: "writeInt16LE too small",
					script: `
						Buffer.alloc(2).writeInt16LE(-32769, 0);
					`,
				},
				{
					name: "writeUInt16BE negative",
					script: `
						Buffer.alloc(2).writeUInt16BE(-1, 0);
					`,
				},
				{
					name: "writeUInt16LE too large",
					script: `
						Buffer.alloc(2).writeUInt16LE(65536, 0);
					`,
				},
				{
					name: "writeInt32BE too large",
					script: `
						Buffer.alloc(4).writeInt32BE(2147483648, 0);
					`,
				},
				{
					name: "writeInt32LE too small",
					script: `
						Buffer.alloc(4).writeInt32LE(-2147483649, 0);
					`,
				},
				{
					name: "writeUInt32BE negative",
					script: `
						Buffer.alloc(4).writeUInt32BE(-1, 0);
					`,
				},
				{
					name: "writeUInt32LE too large",
					script: `
						Buffer.alloc(4).writeUInt32LE(4294967296, 0);
					`,
				},
				{
					name: "writeBigInt64BE too large",
					script: `
						Buffer.alloc(8).writeBigInt64BE(BigInt('9223372036854775808'), 0);
					`,
				},
				{
					name: "writeBigInt64LE too small",
					script: `
						Buffer.alloc(8).writeBigInt64LE(BigInt('-9223372036854775809'), 0);
					`,
				},
				{
					name: "writeBigUInt64BE negative",
					script: `
						Buffer.alloc(8).writeBigUInt64BE(BigInt(-1), 0);
					`,
				},
				{
					name: "writeBigUInt64LE too large",
					script: `
						Buffer.alloc(8).writeBigUInt64LE(BigInt('18446744073709551616'), 0);
					`,
				},
				{
					name: "writeIntBE 1 byte too large",
					script: `
						Buffer.alloc(1).writeIntBE(128, 0, 1);
					`,
				},
				{
					name: "writeIntLE 2 bytes too small",
					script: `
						Buffer.alloc(2).writeIntLE(-32769, 0, 2);
					`,
				},
				{
					name: "writeUIntBE 3 bytes too large",
					script: `
						Buffer.alloc(3).writeUIntBE(16777216, 0, 3);
					`,
				},
				{
					name: "writeUIntLE negative value",
					script: `
						Buffer.alloc(4).writeUIntLE(-1, 0, 4);
					`,
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					_, err := vm.RunString(ctx, tt.script)
					assert.Error(t, err, "Expected error for %s", tt.name)
					assert.Contains(t, err.Error(), "value")
				})
			}
		})

		t.Run("write errors", func(t *testing.T) {
			tests := []struct {
				name   string
				script string
			}{
				{
					name: "negative offset",
					script: `
						Buffer.alloc(1).writeInt8(0, -1);
					`,
				},
				{
					name: "offset out of bounds",
					script: `
						Buffer.alloc(1).writeInt8(0, 2);
					`,
				},
				{
					name: "buffer too small",
					script: `
						Buffer.alloc(1).writeInt16BE(0, 0);
					`,
				},
				{
					name: "invalid byteLength",
					script: `
						Buffer.alloc(4).writeIntBE(0, 0, 7);
					`,
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					_, err := vm.RunString(ctx, tt.script)
					assert.Error(t, err)
				})
			}
		})
	})

	t.Run("instance methods", func(t *testing.T) {
		t.Run("equals", func(t *testing.T) {
			tests := []struct {
				name     string
				buf1     string
				buf2     string
				expected bool
			}{
				{
					name:     "identical buffers",
					buf1:     `Buffer.from("hello")`,
					buf2:     `Buffer.from("hello")`,
					expected: true,
				},
				{
					name:     "different content",
					buf1:     `Buffer.from("hello")`,
					buf2:     `Buffer.from("world")`,
					expected: false,
				},
				{
					name:     "different length",
					buf1:     `Buffer.from("hello")`,
					buf2:     `Buffer.from("hi")`,
					expected: false,
				},
				{
					name:     "empty buffers",
					buf1:     `Buffer.alloc(0)`,
					buf2:     `Buffer.alloc(0)`,
					expected: true,
				},
				{
					name:     "same content different encoding",
					buf1:     `Buffer.from("hello")`,
					buf2:     `Buffer.from("68656c6c6f", "hex")`,
					expected: true,
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					script := fmt.Sprintf(`%s.equals(%s);
					`, tt.buf1, tt.buf2)
					result, err := vm.RunString(ctx, script)
					require.NoError(t, err)
					assert.Equal(t, tt.expected, result.ToBoolean())
				})
			}
		})

		t.Run("compare", func(t *testing.T) {
			tests := []struct {
				name     string
				buf1     string
				buf2     string
				expected int64
			}{
				{
					name:     "equal buffers",
					buf1:     `Buffer.from("hello")`,
					buf2:     `Buffer.from("hello")`,
					expected: 0,
				},
				{
					name:     "first buffer smaller",
					buf1:     `Buffer.from("hello")`,
					buf2:     `Buffer.from("world")`,
					expected: -1,
				},
				{
					name:     "first buffer larger",
					buf1:     `Buffer.from("world")`,
					buf2:     `Buffer.from("hello")`,
					expected: 1,
				},
				{
					name:     "different lengths",
					buf1:     `Buffer.from("hi")`,
					buf2:     `Buffer.from("hello")`,
					expected: 1,
				},
				{
					name:     "empty buffers",
					buf1:     `Buffer.alloc(0)`,
					buf2:     `Buffer.alloc(0)`,
					expected: 0,
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					script := fmt.Sprintf(`%s.compare(%s);
					`, tt.buf1, tt.buf2)
					result, err := vm.RunString(ctx, script)
					require.NoError(t, err)
					assert.Equal(t, tt.expected, result.ToInteger())
				})
			}
		})

		t.Run("copy", func(t *testing.T) {
			tests := []struct {
				name     string
				setup    string
				expected string
			}{
				{
					name: "basic copy",
					setup: `
						var dest = Buffer.alloc(5);
						Buffer.from("hello").copy(dest);
						dest.toString();
					`,
					expected: "hello",
				},
				{
					name: "partial copy with offset",
					setup: `
						var dest = Buffer.alloc(5);
						Buffer.from("hello").copy(dest, 2);
						dest.toString();
					`,
					expected: "\x00\x00hel",
				},
				{
					name: "copy with source offset",
					setup: `
						var dest = Buffer.alloc(3);
						Buffer.from("hello").copy(dest, 0, 1, 4);
						dest.toString();
					`,
					expected: "ell",
				},
				{
					name: "copy to smaller buffer",
					setup: `
						var dest = Buffer.alloc(3);
						Buffer.from("hello").copy(dest);
						dest.toString();
					`,
					expected: "hel",
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result, err := vm.RunString(ctx, tt.setup)
					require.NoError(t, err)
					assert.Equal(t, tt.expected, result.String())
				})
			}
		})

		t.Run("write", func(t *testing.T) {
			tests := []struct {
				name     string
				setup    string
				expected []byte
			}{
				{
					name: "write string",
					setup: `
						var buf = Buffer.alloc(5);
						buf.write("hello");
						buf
					`,
					expected: []byte("hello"),
				},
				{
					name: "write with offset",
					setup: `
						var buf = Buffer.alloc(5);
						buf.write("hi", 2);
						buf
					`,
					expected: []byte{0, 0, 'h', 'i', 0},
				},
				{
					name: "write with length",
					setup: `
						var buf = Buffer.alloc(5);
						buf.write("hello", 0, 3);
						buf
					`,
					expected: []byte{'h', 'e', 'l', 0, 0},
				},
				{
					name: "write with encoding",
					setup: `
						var buf = Buffer.alloc(3);
						buf.write("68656c", 0, 3, "hex");
						buf
					`,
					expected: []byte("hel"),
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result, err := vm.RunString(ctx, tt.setup)
					require.NoError(t, err)
					data := result.Export().([]byte)
					assert.Equal(t, tt.expected, data)
				})
			}
		})

		t.Run("fill", func(t *testing.T) {
			tests := []struct {
				name     string
				setup    string
				expected []byte
			}{
				{
					name: "fill with number",
					setup: `
						var buf = Buffer.alloc(3);
						buf.fill(65);
						buf
					`,
					expected: []byte{'A', 'A', 'A'},
				},
				{
					name: "fill with string",
					setup: `
						var buf = Buffer.alloc(6);
						buf.fill("ab");
						buf
					`,
					expected: []byte("ababab"),
				},
				{
					name: "fill with offset",
					setup: `
						var buf = Buffer.alloc(5);
						buf.fill("x", 2);
						buf
					`,
					expected: []byte{0, 0, 'x', 'x', 'x'},
				},
				{
					name: "fill with range",
					setup: `
						var buf = Buffer.alloc(5);
						buf.fill("x", 1, 4);
						buf
					`,
					expected: []byte{0, 'x', 'x', 'x', 0},
				},
				{
					name: "fill with encoding",
					setup: `
						var buf = Buffer.alloc(3);
						buf.fill("414243", 0, 3, "hex");
						buf
					`,
					expected: []byte("ABC"),
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result, err := vm.RunString(ctx, tt.setup)
					require.NoError(t, err)
					data := result.Export().([]byte)
					assert.Equal(t, tt.expected, data)
				})
			}
		})
	})
}
