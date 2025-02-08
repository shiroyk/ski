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

func TestFilter(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t, js.WithInitial(func(rt *sobek.Runtime) {
		gq, _ := new(Gq).Instantiate(rt)
		require.NoError(t, rt.Set("$", gq))
		require.NoError(t, rt.Set("selector", gq.ToObject(rt).Get("selector")))
	}))
	ctx := context.Background()

	t.Run("eq", func(t *testing.T) {
		v, err := vm.RunString(ctx, `
			$('<div>1</div><div>2</div><div>3</div>').eq(1).html()
		`)
		require.NoError(t, err)
		assert.Equal(t, "2", v.String())
	})

	t.Run("filter with selector", func(t *testing.T) {
		v, err := vm.RunString(ctx, `
			$('<div class="a">1</div><div class="b">2</div><div class="a">3</div>').filter('.a').length
		`)
		require.NoError(t, err)
		assert.Equal(t, int64(2), v.ToInteger())
	})

	t.Run("filter with function", func(t *testing.T) {
		v, err := vm.RunString(ctx, `
			$('<div>1</div><div>2</div><div>3</div>').filter((i, el) => i % 2 === 0).length
		`)
		require.NoError(t, err)
		assert.Equal(t, int64(2), v.ToInteger())
	})

	t.Run("first", func(t *testing.T) {
		v, err := vm.RunString(ctx, `
			$('<div>1</div><div>2</div><div>3</div>').first().html()
		`)
		require.NoError(t, err)
		assert.Equal(t, "1", v.String())
	})

	t.Run("last", func(t *testing.T) {
		v, err := vm.RunString(ctx, `
			$('<div>1</div><div>2</div><div>3</div>').last().html()
		`)
		require.NoError(t, err)
		assert.Equal(t, "3", v.String())
	})

	t.Run("has", func(t *testing.T) {
		v, err := vm.RunString(ctx, `
			$('<div><span>1</span></div><div>2</div>').has('span').length
		`)
		require.NoError(t, err)
		assert.Equal(t, int64(1), v.ToInteger())
	})

	t.Run("is", func(t *testing.T) {
		v, err := vm.RunString(ctx, `
			$('<div class="test">1</div>').is('.test')
		`)
		require.NoError(t, err)
		assert.Equal(t, true, v.ToBoolean())
	})

	t.Run("even", func(t *testing.T) {
		v, err := vm.RunString(ctx, `
			$('<div>1</div><div>2</div><div>3</div>').even().length
		`)
		require.NoError(t, err)
		assert.Equal(t, int64(2), v.ToInteger())
	})

	t.Run("not", func(t *testing.T) {
		v, err := vm.RunString(ctx, `
			$('<div class="a">1</div><div>2</div><div>3</div>').not(".a").length
		`)
		require.NoError(t, err)
		assert.Equal(t, int64(2), v.ToInteger())
	})

	t.Run("add", func(t *testing.T) {
		v, err := vm.RunString(ctx, `
			$('<div class="a">1</div><p>2<p>').add("p").length
		`)
		require.NoError(t, err)
		assert.Equal(t, int64(3), v.ToInteger())
	})

	t.Run("odd", func(t *testing.T) {
		v, err := vm.RunString(ctx, `
			$('<div>1</div><div>2</div><div>3</div>').odd().length
		`)
		require.NoError(t, err)
		assert.Equal(t, int64(1), v.ToInteger())
	})

	t.Run("slice", func(t *testing.T) {
		v, err := vm.RunString(ctx, `
			$('<div>1</div><div>2</div><div>3</div><div>4</div>').slice(1, 3).length
		`)
		require.NoError(t, err)
		assert.Equal(t, int64(2), v.ToInteger())
	})

	t.Run("map", func(t *testing.T) {
		v, err := vm.RunString(ctx, `
			$('<div>1</div><div>2</div><div>3</div>').map((i, el) => el.html()).join(',')
		`)
		require.NoError(t, err)
		assert.Equal(t, "1,2,3", v.String())
	})

	t.Run("each", func(t *testing.T) {
		_, err := vm.RunString(ctx, `
			$('<div>0</div><div>1</div><div>2</div>').each((i, el) => assert.true(el.text() == i));
		`)
		assert.NoError(t, err)
	})

	t.Run("error cases", func(t *testing.T) {
		tests := []struct {
			name   string
			script string
		}{
			{
				name:   "filter without args",
				script: `$('<div>1</div>').filter()`,
			},
			{
				name:   "has without args",
				script: `$('<div>1</div>').has()`,
			},
			{
				name:   "is without args",
				script: `$('<div>1</div>').is()`,
			},
			{
				name:   "slice without args",
				script: `$('<div>1</div>').slice()`,
			},
			{
				name:   "map without args",
				script: `$('<div>1</div>').map()`,
			},
			{
				name:   "map with non-function",
				script: `$('<div>1</div>').map(1)`,
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
