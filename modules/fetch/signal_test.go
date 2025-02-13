package fetch

import (
	"context"
	"testing"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAbortSignal(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t)
	ctx := context.Background()

	t.Run("abort", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const controller = new AbortController();
				const signal = controller.signal;
				const results = {
					aborted: signal.aborted,
					reason: signal.reason
				};
				controller.abort("test reason");
				return {
					...results,
					abortedAfter: signal.aborted,
					reasonAfter: signal.reason
				};
			}
		`)
		require.NoError(t, err)
		obj := result.ToObject(vm.Runtime())
		assert.False(t, obj.Get("aborted").ToBoolean())
		assert.True(t, sobek.IsUndefined(obj.Get("reason")))
		assert.True(t, obj.Get("abortedAfter").ToBoolean())
		assert.Equal(t, "test reason", obj.Get("reasonAfter").String())
	})

	t.Run("aborted signal", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const signal = AbortSignal.abort("immediate abort");
				return {
					aborted: signal.aborted,
					reason: signal.reason
				};
			}
		`)
		require.NoError(t, err)
		obj := result.ToObject(vm.Runtime())
		assert.True(t, obj.Get("aborted").ToBoolean())
		assert.Equal(t, "immediate abort", obj.Get("reason").String())
	})

	t.Run("timeout signal", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
			export default () => {
				const signal = AbortSignal.timeout(0);
				return {
					aborted: signal.aborted,
					reason: signal.reason
				};
			}
		`)
		require.NoError(t, err)
		obj := result.ToObject(vm.Runtime())
		assert.True(t, obj.Get("aborted").ToBoolean())
		assert.Equal(t, "context deadline exceeded", obj.Get("reason").String())
	})
}
