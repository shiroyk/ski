package events

import (
	"context"
	"testing"

	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
)

func TestCustomEvent(t *testing.T) {
	t.Parallel()
	vm := modulestest.New()
	ctx := context.Background()

	t.Run("custom event", func(t *testing.T) {
		_, err := vm.RunModule(ctx, `
			const event = new CustomEvent('test', { detail: {v: 1}, bubbles: true });
			assert.equal(event.type, 'test');
			assert.true(event.bubbles);
			assert.equal(event.detail.v, 1);
			assert.true(event instanceof Event);
			assert.true(event instanceof CustomEvent);
			assert.true(event.constructor === CustomEvent);
			assert.true(Object.getPrototypeOf(CustomEvent.prototype) === Event.prototype);
			assert.true(new CustomEvent('t').detail === null);
			assert.true(new CustomEvent('t', { detail: undefined }).detail === null);
			
			const target = new EventTarget();
			let detail = null;
			target.addEventListener('test', (e) => { detail = e.detail; });
			target.dispatchEvent(event);
			assert.equal(detail.v, 1);
			
			const capture = (fn) => {
				try { fn(); return null; } catch (err) { return err.code; }
			};
			assert.equal(capture(() => new CustomEvent()), 'ERR_MISSING_ARGS');
			assert.equal(capture(() => new CustomEvent('t', 5)), 'ERR_INVALID_ARG_TYPE');
		`)
		assert.NoError(t, err)
	})
}
