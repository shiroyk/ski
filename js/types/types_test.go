package types

import (
	"context"
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
