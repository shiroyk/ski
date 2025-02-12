package gq

import (
	"context"
	"testing"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGq(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t, js.WithInitial(func(rt *sobek.Runtime) {
		gq, _ := new(Gq).Instantiate(rt)
		require.NoError(t, rt.Set("$", gq))
		require.NoError(t, rt.Set("selector", gq.ToObject(rt).Get("selector")))
	}))
	ctx := context.Background()

	t.Run("basic usage", func(t *testing.T) {
		v, err := vm.RunString(ctx, `
			$('<div><span>ciallo</span></div>').find('span').text()
		`)
		require.NoError(t, err)
		assert.Equal(t, "ciallo", v.String())
	})

	t.Run("constructor", func(t *testing.T) {
		t.Run("empty", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$().length
			`)
			require.NoError(t, err)
			assert.Equal(t, int64(0), v.ToInteger())
		})

		t.Run("selection", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$($('<div>test</div>')).text()
			`)
			require.NoError(t, err)
			assert.Equal(t, "test", v.String())
		})

		t.Run("html string", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$('<div>test</div>').text()
			`)
			require.NoError(t, err)
			assert.Equal(t, "test", v.String())
		})

		t.Run("selector string", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$('span', '<div><span>test</span></div>').text()
			`)
			require.NoError(t, err)
			assert.Equal(t, "test", v.String())
		})

		t.Run("compiled selector", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$(selector('span'), '<div><span>test</span></div>').text()
			`)
			require.NoError(t, err)
			assert.Equal(t, "test", v.String())
		})
	})

	t.Run("length property", func(t *testing.T) {
		v, err := vm.RunString(ctx, `
			$('<div></div><div></div>').length
		`)
		require.NoError(t, err)
		assert.Equal(t, int64(2), v.ToInteger())
	})

	t.Run("iterator", func(t *testing.T) {
		v, err := vm.RunString(ctx, `
			let count = 0;
			for (const node of $('<div>1</div><div>2</div>')) {
				count++;
			}
			count;
		`)
		require.NoError(t, err)
		assert.Equal(t, int64(2), v.ToInteger())
	})

	t.Run("selector", func(t *testing.T) {
		t.Run("valid selector", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$('<div><span>test</span></div>').find(selector('span')).text()
			`)
			require.NoError(t, err)
			assert.Equal(t, "test", v.String())
		})

		t.Run("invalid selector", func(t *testing.T) {
			_, err := vm.RunString(ctx, `
				$('<div><span>test</span></div>').find(selector('[')).length
			`)
			assert.Error(t, err)
		})
	})
}
