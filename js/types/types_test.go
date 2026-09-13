package types

import (
	"context"
	"iter"
	"slices"
	"testing"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIterator(t *testing.T) {
	vm := js.NewVM()
	err := vm.Runtime().Set("iter", func() sobek.Value {
		return Iterator(vm.Runtime(), func(yield func(any) bool) {
			for _, r := range "foo" {
				yield(string(r))
			}
		})
	})
	require.NoError(t, err)
	v, err := vm.RunString(context.Background(), `[...iter()]`)
	require.NoError(t, err)
	assert.EqualValues(t, []any{"f", "o", "o"}, v.Export())
}

func TestNew(t *testing.T) {
	vm := js.NewVM()

	t.Run("new", func(t *testing.T) {
		err := vm.Runtime().Set("foo", func(call sobek.ConstructorCall) *sobek.Object {
			require.NoError(t, call.This.Set("toString", func() string { return "foo" }))
			return nil
		})
		require.NoError(t, err)
		err = vm.Runtime().Set("test", func() {
			object := New(vm.Runtime(), "foo")
			assert.Equal(t, "foo", object.String())
		})
		require.NoError(t, err)
		_, err = vm.RunString(context.Background(), "test()")
		require.NoError(t, err)
	})

	t.Run("not defined", func(t *testing.T) {
		err := vm.Runtime().Set("test", func() {
			New(vm.Runtime(), "bar")
		})
		require.NoError(t, err)
		_, err = vm.RunString(context.Background(), "test()")
		assert.ErrorContains(t, err, "bar is not defined")
	})
}

// exported returns the Go values of the JavaScript values.
func exported(values []sobek.Value) []any {
	result := make([]any, len(values))
	for i, value := range values {
		result[i] = value.Export()
	}
	return result
}

// pairs collects the key-value pairs of the sequence into a map.
func pairs(seq iter.Seq2[string, sobek.Value]) map[string]any {
	result := make(map[string]any)
	for key, value := range seq {
		result[key] = value.Export()
	}
	return result
}

func TestIterate(t *testing.T) {
	vm := js.NewVM()

	t.Run("array", func(t *testing.T) {
		value, err := vm.RunString(context.Background(), `[1, "foo", true]`)
		require.NoError(t, err)
		seq, length := Iterate(value)
		assert.Equal(t, 3, length)
		assert.EqualValues(t, []any{int64(1), "foo", true}, exported(slices.Collect(seq)))
	})

	t.Run("empty array", func(t *testing.T) {
		value, err := vm.RunString(context.Background(), `[]`)
		require.NoError(t, err)
		seq, length := Iterate(value)
		assert.Equal(t, 0, length)
		assert.Empty(t, slices.Collect(seq))
	})

	t.Run("sparse array", func(t *testing.T) {
		value, err := vm.RunString(context.Background(), `[1, , 3]`)
		require.NoError(t, err)
		seq, length := Iterate(value)
		assert.Equal(t, 2, length)
		assert.EqualValues(t, []any{int64(1), int64(3)}, exported(slices.Collect(seq)))
	})

	t.Run("typed array", func(t *testing.T) {
		value, err := vm.RunString(context.Background(), `new Uint8Array([1, 2, 3])`)
		require.NoError(t, err)
		seq, length := Iterate(value)
		assert.Equal(t, 3, length)
		assert.EqualValues(t, []any{int64(1), int64(2), int64(3)}, exported(slices.Collect(seq)))
	})

	t.Run("go slice", func(t *testing.T) {
		seq, length := Iterate(vm.Runtime().ToValue([]string{"foo", "bar"}))
		assert.Equal(t, 2, length)
		assert.EqualValues(t, []any{"foo", "bar"}, exported(slices.Collect(seq)))
	})

	t.Run("single value", func(t *testing.T) {
		cases := []struct {
			name     string
			script   string
			expected []any
		}{
			{"number", `1`, []any{int64(1)}},
			{"string", `"foo"`, []any{"foo"}},
			{"boolean", `true`, []any{true}},
			{"object", `({a: 1})`, []any{map[string]any{"a": int64(1)}}},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				value, err := vm.RunString(context.Background(), c.script)
				require.NoError(t, err)
				seq, length := Iterate(value)
				assert.Equal(t, 1, length)
				assert.EqualValues(t, c.expected, exported(slices.Collect(seq)))
			})
		}
	})

	t.Run("nil", func(t *testing.T) {
		for _, value := range []sobek.Value{nil, sobek.Null(), sobek.Undefined()} {
			seq, length := Iterate(value)
			assert.Equal(t, 0, length)
			assert.Empty(t, exported(slices.Collect(seq)))
		}
	})

	t.Run("break early", func(t *testing.T) {
		value, err := vm.RunString(context.Background(), `[1, 2, 3]`)
		require.NoError(t, err)
		seq, _ := Iterate(value)
		count := 0
		for range seq {
			count++
			break
		}
		assert.Equal(t, 1, count)
	})
}

func TestIterate2(t *testing.T) {
	vm := js.NewVM()

	t.Run("object", func(t *testing.T) {
		value, err := vm.RunString(context.Background(), `({a: 1, b: "foo"})`)
		require.NoError(t, err)
		seq, length := Iterate2(value)
		assert.Equal(t, 2, length)
		assert.EqualValues(t, map[string]any{"a": int64(1), "b": "foo"}, pairs(seq))
	})

	t.Run("empty object", func(t *testing.T) {
		value, err := vm.RunString(context.Background(), `({})`)
		require.NoError(t, err)
		seq, length := Iterate2(value)
		assert.Equal(t, 0, length)
		assert.Empty(t, pairs(seq))
	})

	t.Run("go map", func(t *testing.T) {
		seq, length := Iterate2(vm.Runtime().ToValue(map[string]int{"a": 1}))
		assert.Equal(t, 1, length)
		assert.EqualValues(t, map[string]any{"a": int64(1)}, pairs(seq))
	})

	t.Run("non map", func(t *testing.T) {
		for _, script := range []string{`[1, 2]`, `1`, `"foo"`, `() => {}`} {
			value, err := vm.RunString(context.Background(), script)
			require.NoError(t, err)
			seq, length := Iterate2(value)
			assert.Equal(t, 0, length)
			assert.Empty(t, pairs(seq))
		}
	})

	t.Run("nil", func(t *testing.T) {
		for _, value := range []sobek.Value{nil, sobek.Null(), sobek.Undefined()} {
			seq, length := Iterate2(value)
			assert.Equal(t, 0, length)
			assert.Empty(t, pairs(seq))
		}
	})

	t.Run("break early", func(t *testing.T) {
		value, err := vm.RunString(context.Background(), `({a: 1, b: 2})`)
		require.NoError(t, err)
		seq, _ := Iterate2(value)
		count := 0
		for range seq {
			count++
			break
		}
		assert.Equal(t, 1, count)
	})
}

func TestCheck(t *testing.T) {
	vm := js.NewVM()

	t.Run("IsTypedArray", func(t *testing.T) {
		for _, typ := range typedArrayTypes {
			v, err := vm.RunString(context.Background(), `new `+typ+`(1);`)
			require.NoError(t, err)
			assert.True(t, IsTypedArray(vm.Runtime(), v))
		}

		for _, typ := range []string{"Array", "ArrayBuffer"} {
			v, err := vm.RunString(context.Background(), `new `+typ+`(1);`)
			require.NoError(t, err)
			assert.False(t, IsTypedArray(vm.Runtime(), v))
		}
	})

	t.Run("IsFunc", func(t *testing.T) {
		cases := []struct {
			script string
			result bool
		}{
			{`null`, false},
			{`1`, false},
			{`() => {}`, true},
			{`function foo() {}; foo`, true},
		}

		for _, c := range cases {
			value, err := vm.RunString(context.Background(), c.script)
			require.NoError(t, err)
			assert.Equal(t, c.result, IsFunc(value))
		}
	})

	t.Run("IsNil", func(t *testing.T) {
		cases := []struct {
			script string
			result bool
		}{
			{`null`, true},
			{`undefined`, true},
			{`1`, false},
		}

		for _, c := range cases {
			value, err := vm.RunString(context.Background(), c.script)
			require.NoError(t, err)
			assert.Equal(t, c.result, IsNil(value))
		}
	})

	t.Run("IsPromise", func(t *testing.T) {
		cases := []struct {
			script string
			result bool
		}{
			{`Promise.resolve(1)`, true},
			{`{}`, false},
			{`null`, false},
		}

		for _, c := range cases {
			value, err := vm.RunString(context.Background(), c.script)
			require.NoError(t, err)
			assert.Equal(t, c.result, IsPromise(value))
		}
	})
}
