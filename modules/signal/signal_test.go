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

func TestOnAbort(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t)
	ctx := context.Background()

	var calls []string

	// onAbort is a Go function the module uses to register callbacks on signals.
	onAbort := func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
		name := call.Argument(1).String()
		registered := OnAbort(rt, call.Argument(0), func() { calls = append(calls, name) })
		return rt.ToValue(registered)
	}

	result, err := vm.RunModule(ctx, `
		export default (onAbort) => {
			const out = {};

			const controller = new AbortController();
			out.registered = onAbort(controller.signal, 'first');
			controller.abort();
			out.afterAbort = onAbort(controller.signal, 'second');
			out.aborted = controller.signal.aborted;

			out.immediate = onAbort(AbortSignal.abort('immediate'), 'third');

			const target = new EventTarget();
			const other = new AbortController();
			let fired = 0;
			target.addEventListener('test', () => fired++, { signal: other.signal });
			target.dispatchEvent(new Event('test'));
			other.abort();
			target.dispatchEvent(new Event('test'));
			out.fired = fired;

			try {
				onAbort({}, 'nope');
				out.nonSignal = 'ok';
			} catch (err) {
				out.nonSignal = err.code;
			}
			return out;
		}
	`, onAbort)
	require.NoError(t, err)

	obj := result.ToObject(vm.Runtime())
	assert.True(t, obj.Get("registered").ToBoolean())
	assert.False(t, obj.Get("afterAbort").ToBoolean())
	assert.True(t, obj.Get("aborted").ToBoolean())
	assert.False(t, obj.Get("immediate").ToBoolean())
	assert.Equal(t, int64(1), obj.Get("fired").ToInteger())
	assert.Equal(t, "ERR_INVALID_ARG_TYPE", obj.Get("nonSignal").String())
	assert.Equal(t, []string{"first"}, calls)
}

func TestAbortSignalAny(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t)
	ctx := context.Background()

	_, err := vm.RunModule(ctx, `
		const message = (fn) => {
			try { fn(); return null; } catch (err) { return err.message; }
		};

		// An empty sequence returns a signal which never aborts.
		const empty = AbortSignal.any([]);
		assert.true(empty instanceof AbortSignal);
		assert.equal(empty.aborted, false);
		assert.true(empty.reason === undefined);

		// The result is a new signal, aborted with the reason of the source.
		const controller = new AbortController();
		const result = AbortSignal.any([controller.signal]);
		assert.true(result !== controller.signal);
		assert.equal(result.aborted, false);
		const reason = {detail: 1};
		controller.abort(reason);
		assert.true(result.aborted, 'aborted');
		assert.true(result.reason === reason);

		// An already aborted signal aborts the result with its reason.
		const aborted = new AbortController();
		aborted.abort('first');
		const other = new AbortController();
		const pre = AbortSignal.any([other.signal, aborted.signal]);
		assert.true(pre.aborted);
		assert.equal(pre.reason, 'first');
		other.abort('late');
		assert.equal(pre.reason, 'first');

		// The signal which aborts first wins, the result aborts only once.
		const a = new AbortController();
		const b = new AbortController();
		const first = AbortSignal.any([a.signal, b.signal]);
		let events = 0;
		first.addEventListener('abort', () => events++);
		b.abort('b');
		a.abort('a');
		assert.equal(first.reason, 'b');
		assert.equal(events, 1);

		// Any iterable works.
		assert.true(AbortSignal.any(new Set([aborted.signal])).aborted);
		function* generate() { yield aborted.signal; }
		assert.true(AbortSignal.any(generate()).aborted);

		// The result can be used as the signal option of addEventListener.
		const s1 = new AbortController();
		const s2 = new AbortController();
		const combined = AbortSignal.any([s1.signal, s2.signal]);
		const target = new EventTarget();
		let count = 0;
		target.addEventListener('test', () => count++, { signal: combined });
		target.dispatchEvent(new Event('test'));
		s2.abort();
		target.dispatchEvent(new Event('test'));
		assert.equal(count, 1);

		// The originating signal fires first, then the dependents in the order
		// they were added.
		const origin = new AbortController();
		const seq = [];
		seq.push(origin.signal);
		seq.push(AbortSignal.any([origin.signal]));
		seq.push(AbortSignal.any([origin.signal]));
		seq.push(AbortSignal.any([seq[0]]));
		seq.push(AbortSignal.any([seq[1]]));
		let fired = "";
		for (let i = 0; i < seq.length; i++) {
			seq[i].addEventListener('abort', () => { fired += i; });
		}
		origin.abort();
		assert.equal(fired, "01234");

		// Dependent signals are marked aborted before the abort events fire.
		const source = new AbortController();
		const level1 = AbortSignal.any([source.signal]);
		const level2 = AbortSignal.any([level1]);
		let marked = false;
		source.signal.addEventListener('abort', () => {
			marked = source.signal.aborted && level1.aborted && level2.aborted &&
				AbortSignal.any([level2]).aborted;
		});
		source.abort();
		assert.true(marked);

		// The whole sequence is validated before the signals are observed.
		assert.equal(message(() => AbortSignal.any(1)), 'object is not iterable');
		assert.equal(message(() => AbortSignal.any([1])), 'signals[0] is not of type AbortSignal.');
	`)
	assert.NoError(t, err)
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
			export default () => new Promise((resolve) => {
				const signal = AbortSignal.timeout(0);
				// The signal is not aborted synchronously.
				const aborted = signal.aborted;
				signal.addEventListener('abort', () => resolve({
					aborted,
					name: signal.reason.name,
					code: signal.reason.code,
				}));
				setTimeout(() => resolve({aborted, name: 'not aborted'}), 100);
			})
		`)
		require.NoError(t, err)
		obj := modulestest.PromiseResult(result).ToObject(vm.Runtime())
		assert.False(t, obj.Get("aborted").ToBoolean())
		assert.Equal(t, "TimeoutError", obj.Get("name").String())
		assert.Equal(t, int64(23), obj.Get("code").ToInteger())
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
				assert.true(event.isTrusted);
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
