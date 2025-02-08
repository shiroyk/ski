package assert

import (
	"context"
	"testing"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssert(t *testing.T) {
	vm := js.NewVM(js.WithInitial(func(rt *sobek.Runtime) {
		v, _ := new(Assert).Instantiate(rt)
		_ = rt.Set("assert", v)
	}))
	ctx := context.Background()

	t.Run(`true`, func(t *testing.T) {
		_, err := vm.RunString(ctx, `assert(1 == 1)`)
		require.NoError(t, err)
		_, err = vm.RunString(ctx, `assert(1 == 2)`)
		assert.Error(t, err)
	})

	t.Run(`equal`, func(t *testing.T) {
		_, err := vm.RunString(ctx, `assert.equal(1, 1)`)
		require.NoError(t, err)
		_, err = vm.RunString(ctx, `assert.equal('1', 1)`)
		require.NoError(t, err)
		_, err = vm.RunString(ctx, `assert.equal(1, 2)`)
		assert.Error(t, err)
		_, err = vm.RunString(ctx, `assert.equal(1, '2')`)
		assert.Error(t, err)
	})
}
