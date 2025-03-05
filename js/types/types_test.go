package types

import (
	"context"
	"testing"

	"github.com/shiroyk/ski/js"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsTypedArray(t *testing.T) {
	vm := js.NewVM()

	t.Run("typed array", func(t *testing.T) {
		for _, typ := range typedArrayTypes {
			v, err := vm.RunString(context.Background(), `new `+typ+`(1);`)
			require.NoError(t, err)
			assert.True(t, IsTypedArray(vm.Runtime(), v))
		}
	})

	t.Run("not typed array", func(t *testing.T) {
		for _, typ := range []string{"Array", "ArrayBuffer"} {
			v, err := vm.RunString(context.Background(), `new `+typ+`(1);`)
			require.NoError(t, err)
			assert.False(t, IsTypedArray(vm.Runtime(), v))
		}
	})
}
