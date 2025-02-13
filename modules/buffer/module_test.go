package buffer

import (
	"context"
	"testing"

	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModule(t *testing.T) {
	vm := modulestest.New(t)
	ctx := context.Background()

	t.Run("btoa", func(t *testing.T) {
		v, err := vm.RunString(ctx, `btoa("Hello, world")`)
		require.NoError(t, err)
		assert.Equal(t, `SGVsbG8sIHdvcmxk`, v.String())
	})

	t.Run("atob", func(t *testing.T) {
		v, err := vm.RunString(ctx, `atob("SGVsbG8sIHdvcmxk")`)
		require.NoError(t, err)
		assert.Equal(t, `Hello, world`, v.String())
	})
}
