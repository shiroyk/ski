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

func TestTraversal(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t, js.WithInitial(func(rt *sobek.Runtime) {
		gq, _ := new(Gq).Instantiate(rt)
		require.NoError(t, rt.Set("$", gq))
		require.NoError(t, rt.Set("selector", gq.ToObject(rt).Get("selector")))
	}))
	ctx := context.Background()

	t.Run("find", func(t *testing.T) {
		t.Run("with selector", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$('<div><span>1</span><p>2</p></div>').find('span').text()
			`)
			require.NoError(t, err)
			assert.Equal(t, "1", v.String())
		})

		t.Run("with compiled selector", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$('<div><span>1</span><p>2</p></div>').find(selector('span')).text()
			`)
			require.NoError(t, err)
			assert.Equal(t, "1", v.String())
		})
	})

	t.Run("children", func(t *testing.T) {
		t.Run("with selector", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$('<div><span>1</span><p>2</p></div>').children('span').text()
			`)
			require.NoError(t, err)
			assert.Equal(t, "1", v.String())
		})

		t.Run("with compiled selector", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$('<div><span>1</span><p>2</p></div>').children(selector('span')).text()
			`)
			require.NoError(t, err)
			assert.Equal(t, "1", v.String())
		})
	})

	t.Run("parent", func(t *testing.T) {
		t.Run("without filter", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$('<div class="parent"><span>1</span></div>').find('span').parent().attr('class')
			`)
			require.NoError(t, err)
			assert.Equal(t, "parent", v.String())
		})

		t.Run("with selector", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$('<div class="parent"><span>1</span></div>').find('span').parent('.parent').length
			`)
			require.NoError(t, err)
			assert.Equal(t, int64(1), v.ToInteger())
		})
	})

	t.Run("parents", func(t *testing.T) {
		t.Run("without filter", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$('<div><p><span>1</span></p></div>').find('span').parents().length
			`)
			require.NoError(t, err)
			assert.Equal(t, int64(2), v.ToInteger())
		})

		t.Run("with selector", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$('<div><p><span>1</span></p></div>').find('span').parents('p').length
			`)
			require.NoError(t, err)
			assert.Equal(t, int64(1), v.ToInteger())
		})
	})

	t.Run("next", func(t *testing.T) {
		t.Run("without filter", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$('<div><span>1</span><p>2</p></div>').find('span').next().text()
			`)
			require.NoError(t, err)
			assert.Equal(t, "2", v.String())
		})

		t.Run("with selector", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$('<div><span>1</span><p>2</p><span>3</span></div>').find('span').next('p').text()
			`)
			require.NoError(t, err)
			assert.Equal(t, "2", v.String())
		})
	})

	t.Run("prev", func(t *testing.T) {
		t.Run("without filter", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$('<div><span>1</span><p>2</p></div>').find('p').prev().text()
			`)
			require.NoError(t, err)
			assert.Equal(t, "1", v.String())
		})

		t.Run("with selector", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$('<div><span>1</span><p>2</p><span>3</span></div>').find('span:last-child').prev('p').text()
			`)
			require.NoError(t, err)
			assert.Equal(t, "2", v.String())
		})
	})

	t.Run("siblings", func(t *testing.T) {
		t.Run("without filter", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
			$('<div><span>1</span><p>2</p><span>3</span></div>').find('p').siblings().length
			`)
			require.NoError(t, err)
			assert.Equal(t, int64(2), v.ToInteger())
		})

		t.Run("with selector", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$('<div><span>1</span><p>2</p><span>3</span></div>').find('p').siblings('span').length
			`)
			require.NoError(t, err)
			assert.Equal(t, int64(2), v.ToInteger())
		})
	})

	t.Run("nextAll", func(t *testing.T) {
		t.Run("without filter", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$('<div><span>1</span><p>2</p><span>3</span></div>')
				.find('span:first-child').nextAll().length
			`)
			require.NoError(t, err)
			assert.Equal(t, int64(2), v.ToInteger())
		})

		t.Run("with selector", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$('<div><span>1</span><p>2</p><span>3</span></div>')
				.find('span:first-child').nextAll('span').length
			`)
			require.NoError(t, err)
			assert.Equal(t, int64(1), v.ToInteger())
		})
	})

	t.Run("prevAll", func(t *testing.T) {
		t.Run("without filter", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
			$('<div><span>1</span><p>2</p><span>3</span></div>')
			.find('span:last-child').prevAll().length
			`)
			require.NoError(t, err)
			assert.Equal(t, int64(2), v.ToInteger())
		})

		t.Run("with selector", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$('<div><span>1</span><p>2</p><span>3</span></div>')
				.find('span:last-child').prevAll('span').length
			`)
			require.NoError(t, err)
			assert.Equal(t, int64(1), v.ToInteger())
		})
	})

	t.Run("nextUntil", func(t *testing.T) {
		v, err := vm.RunString(ctx, `
			$('<div><span>1</span><p>2</p><b>3</b><span>4</span></div>')
			.find('span:first-child').nextUntil('span').length
		`)
		require.NoError(t, err)
		assert.Equal(t, int64(2), v.ToInteger())
	})

	t.Run("prevUntil", func(t *testing.T) {
		v, err := vm.RunString(ctx, `
			$('<div><span>1</span><b>2</b><p>3</p><span>4</span></div>')
			.find('span:last-child').prevUntil('span').length
		`)
		require.NoError(t, err)
		assert.Equal(t, int64(2), v.ToInteger())
	})

	t.Run("parentsUntil", func(t *testing.T) {
		v, err := vm.RunString(ctx, `
			$('<div class="stop"><div><div><p><span>1</span></p></div></div></div>')
			.find('span').parentsUntil('.stop').length
		`)
		require.NoError(t, err)
		assert.Equal(t, int64(3), v.ToInteger())
	})

	t.Run("closest", func(t *testing.T) {
		v, err := vm.RunString(ctx, `
			$('<div class="parent"><p><span>1</span></p></div>')
			.find('span').closest('.parent').length
		`)
		require.NoError(t, err)
		assert.Equal(t, int64(1), v.ToInteger())
	})

	t.Run("contents", func(t *testing.T) {
		v, err := vm.RunString(ctx, `
			$('<div>text<span>1</span></div>').contents().length
		`)
		require.NoError(t, err)
		assert.Equal(t, int64(2), v.ToInteger())
	})

	t.Run("error cases", func(t *testing.T) {
		tests := []struct {
			name   string
			script string
		}{
			{
				name:   "find without args",
				script: `$('<div></div>').find()`,
			},
			{
				name:   "children without args",
				script: `$('<div></div>').children()`,
			},
			{
				name:   "nextUntil without args",
				script: `$('<div></div>').nextUntil()`,
			},
			{
				name:   "prevUntil without args",
				script: `$('<div></div>').prevUntil()`,
			},
			{
				name:   "parentsUntil without args",
				script: `$('<div></div>').parentsUntil()`,
			},
			{
				name:   "closest without args",
				script: `$('<div></div>').closest()`,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := vm.RunString(ctx, tt.script)
				assert.Error(t, err)
			})
		}
	})
}
