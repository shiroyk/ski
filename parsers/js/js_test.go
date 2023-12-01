package js

import (
	"flag"
	"os"
	"testing"

	"github.com/shiroyk/cloudcat"
	"github.com/shiroyk/cloudcat/js/loader"
	"github.com/shiroyk/cloudcat/plugin"
	"github.com/stretchr/testify/assert"
)

var (
	jsParser Parser
	ctx      *plugin.Context
)

func TestMain(m *testing.M) {
	flag.Parse()
	cloudcat.Provide(loader.NewModuleLoader())
	ctx = plugin.NewContext(plugin.ContextOptions{
		URL: "http://localhost/home",
	})
	code := m.Run()
	os.Exit(code)
}

func TestGetString(t *testing.T) {
	{
		str, err := jsParser.GetString(ctx, "a", `(async () => ctx.get('content') + 1)()`)
		assert.NoError(t, err)
		assert.Equal(t, "a1", str)
	}

	{
		str, err := jsParser.GetString(ctx, "", `(async () => ({"test":"1"}))()`)
		assert.NoError(t, err)
		assert.JSONEq(t, `{"test":"1"}`, str)
	}
}

func TestGetStrings(t *testing.T) {
	{
		str, err := jsParser.GetStrings(ctx, `["a1"]`,
			`new Promise((r, j) => {
					let s = JSON.parse(ctx.get('content'));
					s.push('a2');
					r(s)
				});`)
		assert.NoError(t, err)
		assert.Equal(t, []string{"a1", "a2"}, str)
	}

	{
		str, err := jsParser.GetStrings(ctx, "", `[{"foo":"1"}, {"bar":"1"}, 19]`)
		assert.NoError(t, err)
		assert.Equal(t, []string{`{"foo":"1"}`, `{"bar":"1"}`, "19"}, str)
	}
}

func TestGetElement(t *testing.T) {
	ele, err := jsParser.GetElement(ctx, ``, `ctx.set('size', 1 + 2);ctx.get('size');`)
	assert.NoError(t, err)
	assert.Equal(t, "3", ele)
}

func TestGetElements(t *testing.T) {
	t.Parallel()
	ele, err := jsParser.GetElements(ctx, ``, `[1, 2]`)
	assert.NoError(t, err)
	assert.Equal(t, []string{"1", "2"}, ele)
}
