package ext

import (
	"context"
	"testing"

	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type valuesContext struct {
	context.Context
	values map[any]any
}

func (v *valuesContext) Value(key any) any {
	return v.values[key]
}

func (v *valuesContext) SetValue(key, value any) {
	v.values[key] = value
}

func TestExt(t *testing.T) {
	t.Parallel()
	t.Run("context values", func(t *testing.T) {
		vm := modulestest.New(t)

		ctx := &valuesContext{context.Background(), map[any]any{"test": "value"}}

		_, err := vm.RunModule(ctx, `
import { context } from "ski/ext";
assert.equal(context.test, "value");
context.foo = "bar";
assert.equal(context.foo, "bar");
`)
		require.NoError(t, err)
		assert.Equal(t, "bar", ctx.Value("foo"))
	})
}
