package timers

import (
	"context"
	"testing"
	"time"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimers(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t, js.WithInitial(func(rt *sobek.Runtime) {
		_, _ = new(Timers).Instantiate(rt)
	}))
	ctx := context.Background()

	t.Run("setTimeout", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
		export default () => {
			return new Promise((resolve) => {
				let start = Date.now();
				setTimeout(() => {
					resolve(Date.now() - start);
				}, 100);
			});
		}
		`)
		require.NoError(t, err)
		elapsed := modulestest.PromiseResult(result).ToInteger()
		assert.GreaterOrEqual(t, elapsed, int64(100))
	})

	t.Run("setTimeout with arguments", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
		export default () => {
			return new Promise((resolve) => {
				setTimeout((a, b) => {
					resolve(a + b);
				}, 0, 1, 2);
			});
		}
		`)
		require.NoError(t, err)
		assert.Equal(t, int64(3), modulestest.PromiseResult(result).ToInteger())
	})

	t.Run("clearTimeout", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
		export default () => {
			return new Promise((resolve) => {
				let called = false;
				const id = setTimeout(() => {
					called = true;
				}, 100);
				clearTimeout(id);
				setTimeout(() => {
					resolve(called);
				}, 200);
			});
		}
		`)
		require.NoError(t, err)
		assert.False(t, modulestest.PromiseResult(result).ToBoolean())
	})

	t.Run("setInterval", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
		export default async () => {
			return await new Promise((resolve) => {
				let count = 0;
				const id = setInterval(() => {
					count++;
					if (count === 3) {
						clearInterval(id);
						resolve(count);
					}
				}, 100);
			});
		}
		`)
		require.NoError(t, err)
		assert.Equal(t, int64(3), modulestest.PromiseResult(result).ToInteger())
	})

	t.Run("setInterval with arguments", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
		export default () => {
			return new Promise((resolve) => {
				let sum = 0;
				const id = setInterval((a, b) => {
					sum += (a + b);
					if (sum >= 9) {
						clearInterval(id);
						resolve(sum);
					}
				}, 100, 1, 2);
			});
		}
		`)
		require.NoError(t, err)
		assert.Equal(t, int64(9), modulestest.PromiseResult(result).ToInteger())
	})

	t.Run("clearInterval", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
		export default () => {
			return new Promise((resolve) => {
				let count = 0;
				const id = setInterval(() => {
					count++;
				}, 100);
				
				setTimeout(() => {
					clearInterval(id);
					resolve(count);
				}, 250);
			});
		}
		`)
		require.NoError(t, err)
		assert.LessOrEqual(t, modulestest.PromiseResult(result).ToInteger(), int64(3))
		assert.Equal(t, 0, len(rtTimers(vm.Runtime()).timer))
	})

	t.Run("error handling", func(t *testing.T) {
		_, err := vm.RunString(ctx, "setTimeout('not a function')")
		assert.Error(t, err)
		_, err = vm.RunString(ctx, "setInterval('not a function')")
		assert.Error(t, err)
	})

	t.Run("multiple stops", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
		export default () => {
			return new Promise((resolve) => {
				let count = 0;
				const id = setInterval(() => {
					count++;
				}, 50);
				
				clearInterval(id);
				clearInterval(id);
				clearInterval(id);
				
				setTimeout(() => { resolve(count) }, 200);
			});
		}
		`)
		require.NoError(t, err)
		assert.Equal(t, int64(0), modulestest.PromiseResult(result).ToInteger())
		assert.Equal(t, 0, len(rtTimers(vm.Runtime()).timer))
	})

	t.Run("concurrent stops", func(t *testing.T) {
		result, err := vm.RunModule(ctx, `
		export default async () => {
			return await new Promise((resolve) => {
				const results = new Set();
				const ids = [];
				
				for(let i = 0; i < 10; i++) {
					ids.push(setInterval(() => { results.add(i) }, 50));
				}

				setTimeout(() => {
					ids.forEach(id => clearInterval(id));
					setTimeout(() => { resolve(results.size) }, 100);
				}, 75);
			});
		}
		`)
		require.NoError(t, err)
		count := modulestest.PromiseResult(result).ToInteger()
		assert.GreaterOrEqual(t, count, int64(5))
		assert.LessOrEqual(t, count, int64(15))
		assert.Equal(t, 0, len(rtTimers(vm.Runtime()).timer))
	})

	t.Run("interrupt", func(t *testing.T) {
		ctx2, cancel := context.WithTimeout(ctx, time.Millisecond*100)
		defer cancel()
		_, err := vm.RunModule(ctx2, `
		export default async () => {
			return await new Promise((resolve) => {
				setTimeout(resolve, 1000);
			})
		}
		`)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Equal(t, 0, len(rtTimers(vm.Runtime()).timer))
	})
}
