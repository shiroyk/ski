package signal

import (
	"context"
	"errors"
	"testing"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js/modulestest"
	_ "github.com/shiroyk/ski/modules/timers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignalCancel(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t)
	ctx := context.Background()

	// New returns a cancel function that aborts the signal from any goroutine.
	signal, cancel := New(vm.Runtime(), ctx)
	result, err := vm.RunModule(ctx, `
		export default (signal, cancel) => new Promise((resolve) => {
			signal.addEventListener('abort', () => {
				resolve({
					aborted: signal.aborted,
					name: signal.reason.name,
					message: signal.reason.message,
				});
			});
			setTimeout(() => cancel(new Error('cause')), 1);
		})
	`, signal, func(sobek.FunctionCall, *sobek.Runtime) sobek.Value {
		cancel(errors.New("cause"))
		return sobek.Undefined()
	})
	require.NoError(t, err)
	obj := modulestest.PromiseResult(result).ToObject(vm.Runtime())
	assert.True(t, obj.Get("aborted").ToBoolean())
	assert.Equal(t, "Error", obj.Get("name").String())
	assert.Equal(t, "cause", obj.Get("message").String())

	// A canceled context without a cause aborts with the AbortError DOMException.
	aborted, cancelAbort := New(vm.Runtime(), ctx)
	result, err = vm.RunModule(ctx, `
		export default (signal, cancel) => new Promise((resolve) => {
			signal.addEventListener('abort', () => resolve({
				name: signal.reason.name,
				code: signal.reason.code,
			}));
			setTimeout(() => cancel(), 1);
		})
	`, aborted, func(sobek.FunctionCall, *sobek.Runtime) sobek.Value {
		cancelAbort(nil)
		return sobek.Undefined()
	})
	require.NoError(t, err)
	obj = modulestest.PromiseResult(result).ToObject(vm.Runtime())
	assert.Equal(t, "AbortError", obj.Get("name").String())
	assert.Equal(t, int64(20), obj.Get("code").ToInteger())
}

func TestSignalTimeout(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t)
	result, err := vm.RunModule(context.Background(), `
		export default () => new Promise((resolve) => {
			const signal = AbortSignal.timeout(5);
			signal.addEventListener('abort', () => resolve({
				name: signal.reason.name,
				code: signal.reason.code,
			}));
			setTimeout(() => resolve({ name: 'not-aborted' }), 100);
		})
	`)
	require.NoError(t, err)
	obj := modulestest.PromiseResult(result).ToObject(vm.Runtime())
	assert.Equal(t, "TimeoutError", obj.Get("name").String())
	assert.Equal(t, int64(23), obj.Get("code").ToInteger())
}

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
		reason := obj.Get("reason")
		assert.Equal(t, "TimeoutError", reason.ToObject(vm.Runtime()).Get("name").String())
		assert.Equal(t, int64(23), reason.ToObject(vm.Runtime()).Get("code").ToInteger())
	})

	t.Run("default abort reason", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const controller = new AbortController();
			controller.abort();
			
			assert.true(controller.signal.aborted);
			assert.equal(controller.signal.reason.name, 'AbortError');
			assert.equal(controller.signal.reason.code, 20);
			assert.true(controller.signal.reason instanceof DOMException);
			assert.true(controller.signal.reason instanceof Error);
		`)
		assert.NoError(t, err)
	})

	t.Run("event target", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const controller = new AbortController();
			const signal = controller.signal;
			
			assert.true(signal instanceof EventTarget);
			assert.true(signal instanceof AbortSignal);
			assert.true(signal.constructor === AbortSignal);
		`)
		assert.NoError(t, err)
	})

	t.Run("abort event", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const controller = new AbortController();
			const signal = controller.signal;
			const order = [];
			
			signal.addEventListener('abort', (event) => {
				order.push('first');
				assert.true(event.target === signal);
				assert.true(event.currentTarget === signal);
				assert.equal(event.type, 'abort');
				assert.true(signal.aborted);
			});
			signal.addEventListener('abort', () => order.push('second'));
			
			assert.equal(order.length, 0);
			controller.abort();
			assert.equal(order, ['first', 'second']);
		`)
		assert.NoError(t, err)
	})

	t.Run("onabort", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const controller = new AbortController();
			const signal = controller.signal;
			let called = 0;
			
			assert.true(signal.onabort === null);
			
			const handler = () => called++;
			signal.onabort = handler;
			assert.true(signal.onabort === handler);
			
			controller.abort();
			assert.equal(called, 1);
			
			signal.onabort = null;
			assert.true(signal.onabort === null);
		`)
		assert.NoError(t, err)
	})

	t.Run("throwIfAborted", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const controller = new AbortController();
			controller.signal.throwIfAborted();
			
			const reason = new Error('custom');
			controller.abort(reason);
			
			let thrown;
			try {
				controller.signal.throwIfAborted();
			} catch (err) {
				thrown = err;
			}
			assert.true(thrown === reason);
		`)
		assert.NoError(t, err)
	})

	t.Run("addEventListener signal option", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			const controller = new AbortController();
			let count = 0;
			
			target.addEventListener('test', () => count++, { signal: controller.signal });
			
			target.dispatchEvent(new Event('test'));
			assert.equal(count, 1);
			
			controller.abort();
			target.dispatchEvent(new Event('test'));
			assert.equal(count, 1);
		`)
		assert.NoError(t, err)
	})

	t.Run("addEventListener aborted signal option", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			const controller = new AbortController();
			controller.abort();
			let count = 0;
			
			target.addEventListener('test', () => count++, { signal: controller.signal });
			target.dispatchEvent(new Event('test'));
			
			assert.equal(count, 0);
		`)
		assert.NoError(t, err)
	})

	t.Run("addEventListener invalid signal option", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			let threw = false;
			
			try {
				target.addEventListener('test', () => {}, { signal: {} });
			} catch (err) {
				threw = err instanceof TypeError && err.code === 'ERR_INVALID_ARG_TYPE';
			}
			
			assert.true(threw);
		`)
		assert.NoError(t, err)
	})
}
