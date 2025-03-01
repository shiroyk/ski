package dom

import (
	"context"
	"testing"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js/modulestest"
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

	t.Run("signal option", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			const controller = new AbortController();
			let count = 0;
			
			target.addEventListener('test', () => count++, { signal: controller.signal });
			
			const event = new Event('test');
			target.dispatchEvent(event);
			assert.equal(count, 1);
			
			controller.abort();
			setTimeout(() => {
				target.dispatchEvent(event);
				assert.equal(count, 1);
			}, 0);
		`)
		assert.NoError(t, err)
	})

	t.Run("event currentTarget", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			
			target.addEventListener('test', (event) => {
				assert.equal(event.currentTarget, target);
			});
			
			const event = new Event('test');
			target.dispatchEvent(event);
		`)
		assert.NoError(t, err)
	})

	t.Run("invalid listener", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const target = new EventTarget();
			
			target.addEventListener('test', 'not a function');
			target.addEventListener('test', null);
			target.addEventListener('test', undefined);
			target.addEventListener('test', {});
			
			const event = new Event('test');
			assert.true(target.dispatchEvent(event));
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

	t.Run("events order", func(t *testing.T) {
		setParent := func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
			toEventTarget(rt, call.Argument(0)).setParent(toEventTarget(rt, call.Argument(1)))
			return sobek.Undefined()
		}
		_, err := vm.RunModule(ctx, `
			export default (setParent) => {
				const e1 = new EventTarget();
				const e2 = new EventTarget();
				const e3 = new EventTarget();
				setParent(e2, e1);
				setParent(e3, e2);
				const order = [];
				
				e1.addEventListener('test', () => order.push('e1 capture'), true);
				e1.addEventListener('test', () => order.push('e1 bubble'), false);
				e2.addEventListener('test', () => order.push('e2 capture'), true);
				e2.addEventListener('test', () => order.push('e2 bubble'));
				e3.addEventListener('test', () => order.push('e3 target'), false);
				
				const event = new Event('test', { bubbles: true });
				e3.dispatchEvent(event);
				
				assert.equal(order, ["e1 capture", "e2 capture", "e3 target", "e2 bubble", "e1 bubble"]);
			}
		`, setParent)
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
}
