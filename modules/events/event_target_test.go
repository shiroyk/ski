package events

import (
	"context"
	"testing"

	"github.com/shiroyk/ski/js/modulestest"
	_ "github.com/shiroyk/ski/modules/timers"
	"github.com/stretchr/testify/assert"
)

func TestEventTarget(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t)
	ctx := context.Background()

	t.Run("addEventListener", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			let called = false;
			
			target.addEventListener('test', (event) => {
				called = true;
				assert.equal(event.type, 'test');
			});
			
			const event = new Event('test');
			target.dispatchEvent(event);
			
			assert.true(called);
		`)
		assert.NoError(t, err)
	})

	t.Run("removeEventListener", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			let count = 0;
			
			const listener = () => count++;
			
			target.addEventListener('test', listener);
			target.removeEventListener('test', listener);
			
			const event = new Event('test');
			target.dispatchEvent(event);
			
			assert.equal(count, 0);
		`)
		assert.NoError(t, err)
	})

	t.Run("multiple listeners", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			let count = 0;
			
			target.addEventListener('test', () => count++);
			target.addEventListener('test', () => count++);
			
			const event = new Event('test');
			target.dispatchEvent(event);
			
			assert.equal(count, 2);
		`)
		assert.NoError(t, err)
	})

	t.Run("once option", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			let count = 0;
			
			target.addEventListener('test', () => count++, { once: true });
			
			const event = new Event('test');
			target.dispatchEvent(event);
			target.dispatchEvent(event);
			
			assert.equal(count, 1);
		`)
		assert.NoError(t, err)
	})

	t.Run("stopPropagation", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			let count = 0;
			
			target.addEventListener('test', (event) => {
				count++;
				event.stopPropagation();
			});
			target.addEventListener('test', () => count++);
			
			const event = new Event('test');
			target.dispatchEvent(event);
			
			assert.equal(count, 2);
		`)
		assert.NoError(t, err)
	})

	t.Run("stopImmediatePropagation", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			let count = 0;
			
			target.addEventListener('test', (event) => {
				count++;
				event.stopImmediatePropagation();
			});
			target.addEventListener('test', () => count++);
			
			const event = new Event('test');
			target.dispatchEvent(event);
			
			assert.equal(count, 1);
		`)
		assert.NoError(t, err)
	})

	t.Run("preventDefault", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			
			target.addEventListener('test', (event) => {
				event.preventDefault();
			});
			
			const event = new Event('test', {cancelable: true});
			const result = target.dispatchEvent(event);
			
			assert.true(!result);
			assert.true(event.defaultPrevented, event.NONE);
		`)
		assert.NoError(t, err)
	})

	t.Run("event currentTarget", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			
			target.addEventListener('test', (event) => {
				assert.true(event.currentTarget === target);
			});
			
			const event = new Event('test');
			target.dispatchEvent(event);
		`)
		assert.NoError(t, err)
	})

	t.Run("invalid listener", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			
			// Node.js: null and undefined listeners have no effect, plain objects
			// are accepted and only invoked when handleEvent is a function.
			target.addEventListener('test', null);
			target.addEventListener('test', undefined);
			target.addEventListener('test', {});
			
			assert.true(target.dispatchEvent(new Event('test')));
			
			let code = null;
			try {
				target.addEventListener('test', 'not a function');
			} catch (err) {
				code = err.code;
			}
			assert.equal(code, 'ERR_INVALID_ARG_TYPE');
			
			code = null;
			try {
				target.addEventListener('test');
			} catch (err) {
				code = err.code;
			}
			assert.equal(code, 'ERR_MISSING_ARGS');
		`)
		assert.NoError(t, err)
	})

	t.Run("node errors", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			const capture = (fn) => {
				try { fn(); return null; } catch (err) { return err.code; }
			};
			
			assert.equal(capture(() => target.dispatchEvent()), 'ERR_MISSING_ARGS');
			assert.equal(capture(() => target.dispatchEvent(undefined)), 'ERR_INVALID_ARG_TYPE');
			assert.equal(capture(() => target.dispatchEvent(null)), 'ERR_INVALID_ARG_TYPE');
			assert.equal(capture(() => target.dispatchEvent({type: 'test'})), 'ERR_INVALID_ARG_TYPE');
			assert.equal(capture(() => target.dispatchEvent('test')), 'ERR_INVALID_ARG_TYPE');
			assert.equal(capture(() => target.removeEventListener('test')), 'ERR_MISSING_ARGS');
			assert.equal(capture(() => new Event()), 'ERR_MISSING_ARGS');
			assert.equal(capture(() => new Event('test', 5)), 'ERR_INVALID_ARG_TYPE');
		`)
		assert.NoError(t, err)
	})

	t.Run("listener error handling", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			let secondCalled = false;
			
			target.addEventListener('test', () => {
				throw new Error('Test error');
			});
			target.addEventListener('test', () => {
				secondCalled = true;
			});
			
			const event = new Event('test');
			target.dispatchEvent(event);
			assert.true(secondCalled);
		`)
		assert.NoError(t, err)
	})

	t.Run("composed path", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			let path;
			
			target.addEventListener('test', (event) => {
				path = event.composedPath();
			});
			target.dispatchEvent(new Event('test'));
			
			assert.equal(path.length, 1);
			assert.true(path[0] === target);
		`)
		assert.NoError(t, err)
	})

	t.Run("passive option", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			
			target.addEventListener('test', (event) => {
				event.preventDefault();
			}, { passive: true });
			
			const event = new Event('test');
			const result = target.dispatchEvent(event);
			
			assert.true(result);
			assert.true(!event.defaultPrevented);
		`)
		assert.NoError(t, err)
	})

	t.Run("event identity", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			const seen = [];
			let first;
			
			target.addEventListener('test', (event) => {
				seen.push(event.currentTarget === target);
				event.marker = 1;
				first = event;
			});
			target.addEventListener('test', (event) => {
				seen.push(event.marker);
				seen.push(event === first);
			});
			
			target.dispatchEvent(new Event('test'));
			
			assert.equal(seen, [true, 1, true]);
		`)
		assert.NoError(t, err)
	})

	t.Run("at target captures", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			const order = [];
			
			target.addEventListener('test', () => order.push('capture'), true);
			target.addEventListener('test', () => order.push('bubble'), false);
			
			target.dispatchEvent(new Event('test'));
			
			assert.equal(order, ['capture', 'bubble']);
		`)
		assert.NoError(t, err)
	})

	t.Run("dispatch recursion", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			const event = new Event('test');
			let code = null;
			
			target.addEventListener('test', () => {
				try {
					target.dispatchEvent(event);
				} catch (err) {
					code = err.code;
				}
			});
			target.dispatchEvent(event);
			
			assert.equal(code, 'ERR_EVENT_RECURSION');
		`)
		assert.NoError(t, err)
	})

	t.Run("redispatch", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			let count = 0;
			
			target.addEventListener('test', (event) => {
				count++;
				event.stopImmediatePropagation();
			});
			
			const event = new Event('test');
			target.dispatchEvent(event);
			target.dispatchEvent(event);
			
			assert.equal(count, 2);
		`)
		assert.NoError(t, err)
	})

	t.Run("target persists", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			target.addEventListener('test', () => {});
			
			const event = new Event('test');
			target.dispatchEvent(event);
			
			assert.true(event.target === target);
			assert.true(event.currentTarget === null);
			assert.equal(event.eventPhase, Event.NONE);
			assert.equal(event.composedPath().length, 0);
		`)
		assert.NoError(t, err)
	})

	t.Run("duplicate listener", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			let count = 0;
			const listener = () => count++;
			
			target.addEventListener('test', listener);
			target.addEventListener('test', listener);
			target.addEventListener('test', listener, { once: true });
			
			target.dispatchEvent(new Event('test'));
			target.dispatchEvent(new Event('test'));
			
			assert.equal(count, 2);
		`)
		assert.NoError(t, err)
	})

	t.Run("this binding", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			let bound = null;
			
			target.addEventListener('test', function () { bound = this === target; });
			target.dispatchEvent(new Event('test'));
			
			assert.true(bound);
		`)
		assert.NoError(t, err)
	})

	t.Run("listener removed during dispatch", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			const hits = [];
			const second = () => hits.push('second');
			
			target.addEventListener('test', () => {
				hits.push('first');
				target.removeEventListener('test', second);
			});
			target.addEventListener('test', second);
			
			target.dispatchEvent(new Event('test'));
			
			assert.equal(hits, ['first']);
		`)
		assert.NoError(t, err)
	})

	t.Run("handleEvent listener", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			let handled = null;
			const listener = {
				handleEvent(event) {
					handled = this === listener && event.type === 'test';
				}
			};
			
			target.addEventListener('test', listener);
			
			target.dispatchEvent(new Event('test'));
			
			assert.true(handled);
		`)
		assert.NoError(t, err)
	})

	t.Run("isTrusted", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			let trusted = null;
			target.addEventListener('test', (event) => { trusted = event.isTrusted; });
			target.dispatchEvent(new Event('test'));
			assert.equal(trusted, false);
		`)
		assert.NoError(t, err)
	})

	t.Run("plain object", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const event = new Event('test');
			const target = new EventTarget();
			const custom = new CustomEvent('test');
			
			// The JavaScript objects must stay plain: no Go methods leaked as own
			// properties, extensible for expando properties and subclasses.
			assert.equal(Object.getOwnPropertyNames(event).length, 0);
			assert.equal(Object.getOwnPropertyNames(target).length, 0);
			assert.equal(Object.getOwnPropertyNames(custom).length, 0);
			assert.true(Object.isExtensible(event));
			assert.true(Object.isExtensible(target));
			event.expando = 1;
			target.expando = 2;
			assert.equal(event.expando, 1);
			assert.equal(target.expando, 2);
		`)
		assert.NoError(t, err)
	})

	t.Run("extends Event", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			class MyEvent extends Event {
				constructor(type, init) {
					super(type, init);
					this.extra = 1;
				}
				get detail() { return 'd'; }
			}
			
			const event = new MyEvent('test', { bubbles: true });
			assert.equal(event.type, 'test');
			assert.true(event.bubbles);
			assert.equal(event.extra, 1);
			assert.equal(event.detail, 'd');
			assert.true(event instanceof Event);
			assert.true(event instanceof MyEvent);
			assert.true(event.constructor === MyEvent);
			
			const target = new EventTarget();
			let same = false, current = false;
			target.addEventListener('test', (e) => {
				same = e === event;
				current = e.target === target;
				assert.equal(e.detail, 'd');
			});
			target.dispatchEvent(event);
			
			assert.true(same);
			assert.true(current);
			assert.true(event.currentTarget === null);
		`)
		assert.NoError(t, err)
	})

	t.Run("extends EventTarget", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			class MyTarget extends EventTarget {
				constructor() {
					super();
					this.hits = 0;
				}
			}
			
			const target = new MyTarget();
			let got = null;
			target.addEventListener('test', function (event) {
				got = this.hits + ':' + (event.currentTarget === target);
			});
			target.dispatchEvent(new Event('test'));
			
			assert.true(target instanceof EventTarget);
			assert.true(target instanceof MyTarget);
			assert.equal(got, '0:true');
		`)
		assert.NoError(t, err)
	})
}
