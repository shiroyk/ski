package stream

import (
	"context"
	"strings"
	"testing"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newResponse(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	stream := NewReadableStream(rt, strings.NewReader(call.Argument(0).String()))
	ret := rt.NewObject()
	_ = ret.DefineAccessorProperty("body", rt.ToValue(func(functionCall sobek.FunctionCall) sobek.Value { return stream }), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	return ret
}

func TestReadableStream(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t, js.WithInitial(func(rt *sobek.Runtime) {
		_ = rt.Set("Response", newResponse)
	}))
	ctx := context.Background()

	t.Run("default reader", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default async () => {
				const stream = new Response("hello world").body;
				const reader = stream.getReader();
				const results = [];
				
				try {
					while (true) {
						const { done, value } = await reader.read();
						if (done) break;
						results.push(String.fromCharCode.apply(String, value));
					}
				} finally {
					reader.releaseLock();
				}

				return {
					text: results.join(''),
					locked: stream.locked,
				};
			}
		`)
		require.NoError(t, err)
		obj := modulestest.PromiseResult(result).ToObject(vm.Runtime())
		assert.Equal(t, "hello world", obj.Get("text").String())
		assert.False(t, obj.Get("locked").ToBoolean())
	})

	t.Run("BYOB reader", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default async () => {
				const stream = new Response("hello world").body;
				const reader = stream.getReader({ mode: 'byob' });
				const buffer = new ArrayBuffer(5);
				const results = [];
				
				try {
					while (true) {
						const { done, value } = await reader.read(buffer);
						if (done) break;
						results.push(String.fromCharCode.apply(String, value));
					}
				} finally {
					reader.releaseLock();
				}

				return {
					text: results.join(''),
					locked: stream.locked,
				};
			}
		`)
		require.NoError(t, err)
		obj := modulestest.PromiseResult(result).ToObject(vm.Runtime())
		assert.Equal(t, "hello world", obj.Get("text").String())
		assert.False(t, obj.Get("locked").ToBoolean())
	})

	t.Run("cancel", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default async () => {
				const stream = new Response("hello world").body;
				await stream.cancel();
				return {
					locked: stream.locked,
					getReader: () => stream.getReader(),
				};
			}
		`)
		require.NoError(t, err)
		obj := modulestest.PromiseResult(result).ToObject(vm.Runtime())
		assert.False(t, obj.Get("locked").ToBoolean())
		getReader, _ := sobek.AssertFunction(obj.Get("getReader"))
		_, err = getReader(sobek.Undefined())
		assert.ErrorContains(t, err, "stream is already closed") // Should throw error after cancel
	})

	t.Run("lock errors", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const stream = new Response("test").body;
				const reader = stream.getReader();
				return {
					getReader: () => stream.getReader(),
					getReader2: () => stream.getReader(),
				};
			}
		`)
		require.NoError(t, err)
		obj := result.ToObject(vm.Runtime())
		getReader, _ := sobek.AssertFunction(obj.Get("getReader"))
		_, err = getReader(sobek.Undefined())
		assert.Error(t, err)
		getReader2, _ := sobek.AssertFunction(obj.Get("getReader2"))
		_, err = getReader2(sobek.Undefined())
		assert.ErrorContains(t, err, "stream is already locked")
	})

	t.Run("BYOB reader errors", func(t *testing.T) {
		tests := []struct {
			name, input, msg string
		}{
			{
				name:  "missing buffer",
				input: "reader.read()",
				msg:   "requires a buffer argument",
			},
			{
				name:  "invalid buffer type",
				input: `reader.read("not a buffer")`,
				msg:   "argument must be an ArrayBuffer",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := vm.RunModule(ctx, `
					export default async () => {
						const stream = new Response("test").body;
						const reader = stream.getReader({ mode: 'byob' });
						try {
							await `+tt.input+`
						} catch (e) {
							return e.message;
						}
					}
				`)
				require.NoError(t, err)
				assert.Contains(t, modulestest.PromiseResult(result).String(), tt.msg)
			})
		}
	})
}
