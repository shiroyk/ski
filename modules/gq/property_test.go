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

func TestProperty(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t, js.WithInitial(func(rt *sobek.Runtime) {
		gq, _ := new(Gq).Instantiate(rt)
		require.NoError(t, rt.Set("$", gq))
		require.NoError(t, rt.Set("selector", gq.ToObject(rt).Get("selector")))
	}))
	ctx := context.Background()

	t.Run("attr", func(t *testing.T) {
		t.Run("get attr", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$('<div id="test" data-value="123"></div>').attr('id')
			`)
			require.NoError(t, err)
			assert.Equal(t, "test", v.String())
		})

		t.Run("get non-existent attr", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$('<div></div>').attr('id')
			`)
			require.NoError(t, err)
			assert.Nil(t, v.Export())
		})
	})

	t.Run("removeAttr", func(t *testing.T) {
		v, err := vm.RunString(ctx, `
		{
			const sel = $('<div id="test"></div>');
			sel.removeAttr('id');
			sel.attr('id');
		}`)
		require.NoError(t, err)
		assert.Nil(t, v.Export())
	})

	t.Run("val", func(t *testing.T) {
		v, err := vm.RunString(ctx, `
			$('<input value="test">').val()
		`)
		require.NoError(t, err)
		assert.Equal(t, "test", v.String())
	})

	t.Run("html", func(t *testing.T) {
		v, err := vm.RunString(ctx, `
			$('<div><span>test</span></div>').html()
		`)
		require.NoError(t, err)
		assert.Equal(t, "<span>test</span>", v.String())
	})

	t.Run("text", func(t *testing.T) {
		v, err := vm.RunString(ctx, `
			$('<div>Hello <span>World</span></div>').text()
		`)
		require.NoError(t, err)
		assert.Equal(t, "Hello World", v.String())
	})

	t.Run("href", func(t *testing.T) {
		t.Run("absolute url", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$('<a href="https://example.com"></a>').href()
			`)
			require.NoError(t, err)
			assert.Equal(t, "https://example.com", v.String())
		})

		t.Run("relative url with base", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$('<a href="/path"></a>').href('https://example.com')
			`)
			require.NoError(t, err)
			assert.Equal(t, "https://example.com/path", v.String())
		})

		t.Run("multiple hrefs", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$('<a href="/1"></a><a href="/2"></a>').href('https://example.com')
			`)
			require.NoError(t, err)
			arr := v.Export().([]string)
			assert.Equal(t, []string{"https://example.com/1", "https://example.com/2"}, arr)
		})
	})

	t.Run("class operations", func(t *testing.T) {
		t.Run("addClass", func(t *testing.T) {
			t.Run("single class", func(t *testing.T) {
				v, err := vm.RunString(ctx, `
				{
					const sel = $('<div></div>')
					sel.addClass('test')
					sel.attr('class')
				}`)
				require.NoError(t, err)
				assert.Equal(t, "test", v.String())
			})

			t.Run("multiple classes", func(t *testing.T) {
				v, err := vm.RunString(ctx, `
				{
					const sel = $('<div></div>')
					sel.addClass(['test1', 'test2'])
					sel.attr('class')
				}`)
				require.NoError(t, err)
				assert.Equal(t, "test1 test2", v.String())
			})

			t.Run("function", func(t *testing.T) {
				v, err := vm.RunString(ctx, `
				{
					const sel = $('<div></div><div></div>')
					sel.addClass((i) => 'test' + i)
					sel.map((i, el) => el.attr('class')).join(',')
				}`)
				require.NoError(t, err)
				assert.Equal(t, "test0,test1", v.String())
			})
		})

		t.Run("hasClass", func(t *testing.T) {
			v, err := vm.RunString(ctx, `
				$('<div class="test"></div>').hasClass('test')
			`)
			require.NoError(t, err)
			assert.True(t, v.ToBoolean())
		})

		t.Run("removeClass", func(t *testing.T) {
			t.Run("single class", func(t *testing.T) {
				v, err := vm.RunString(ctx, `
				{
					const sel = $('<div class="test1 test2"></div>')
					sel.removeClass('test1')
					sel.attr('class')
				}`)
				require.NoError(t, err)
				assert.Equal(t, "test2", v.String())
			})

			t.Run("multiple classes", func(t *testing.T) {
				v, err := vm.RunString(ctx, `
				{
					const sel = $('<div class="test1 test2 test3"></div>')
					sel.removeClass(['test1', 'test2'])
					sel.attr('class')
				}`)
				require.NoError(t, err)
				assert.Equal(t, "test3", v.String())
			})

			t.Run("function", func(t *testing.T) {
				v, err := vm.RunString(ctx, `
				{
					const sel = $('<div class="test0"></div><div class="test1"></div>')
					sel.removeClass((i) => 'test' + i)
					sel.map((i, el) => el.attr('class')).join(',')
				}`)
				require.NoError(t, err)
				assert.Equal(t, ",", v.String())
			})
		})

		t.Run("toggleClass", func(t *testing.T) {
			t.Run("toggle on", func(t *testing.T) {
				v, err := vm.RunString(ctx, `
				{
					const sel = $('<div></div>')
					sel.toggleClass('test', true)
					sel.hasClass('test')
				}`)
				require.NoError(t, err)
				assert.True(t, v.ToBoolean())
			})

			t.Run("toggle off", func(t *testing.T) {
				v, err := vm.RunString(ctx, `
				{
					const sel = $('<div class="test"></div>')
					sel.toggleClass('test', false)
					sel.hasClass('test')
				}`)
				require.NoError(t, err)
				assert.False(t, v.ToBoolean())
			})

			t.Run("function", func(t *testing.T) {
				v, err := vm.RunString(ctx, `
				{
					const sel = $('<div class="test"></div>')
					sel.toggleClass((i, className, state) => state ? 'foo' : 'test', true)
					sel.attr('class')
				}`)
				require.NoError(t, err)
				assert.Equal(t, "test  foo", v.String())
			})
		})
	})

	t.Run("error cases", func(t *testing.T) {
		tests := []struct {
			name   string
			script string
		}{
			{
				name:   "attr without args",
				script: `$('<div></div>').attr()`,
			},
			{
				name:   "removeAttr without args",
				script: `$('<div></div>').removeAttr()`,
			},
			{
				name:   "addClass without args",
				script: `$('<div></div>').addClass()`,
			},
			{
				name:   "removeClass without args",
				script: `$('<div></div>').removeClass()`,
			},
			{
				name:   "toggleClass without args",
				script: `$('<div></div>').toggleClass()`,
			},
			{
				name:   "hasClass without args",
				script: `$('<div></div>').hasClass()`,
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
