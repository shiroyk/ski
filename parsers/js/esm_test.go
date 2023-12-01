package js

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var esmParser = NewESMParser()

func TestESMCache(t *testing.T) {
	_, err := esmParser.GetString(ctx, ``, `export default 1;`)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(esmParser.cache))
	_, err = esmParser.GetString(ctx, ``, `export default 1;`)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(esmParser.cache))
	esmParser.ClearCache()
	assert.Equal(t, 0, len(esmParser.cache))
}

func TestESMGetString(t *testing.T) {
	{
		str, err := esmParser.GetString(ctx, "a", `export default (ctx) => ctx.get('content') + 1`)
		assert.NoError(t, err)
		assert.Equal(t, "a1", str)
	}

	{
		str, err := esmParser.GetString(ctx, "", `export default () => ({"test":"1"})`)
		assert.NoError(t, err)
		assert.JSONEq(t, `{"test":"1"}`, str)
	}
}

func TestESMGetStrings(t *testing.T) {
	{
		str, err := esmParser.GetStrings(ctx, `["a1"]`,
			`export default function (ctx) {
				return new Promise((r, j) => {
					let s = JSON.parse(ctx.get('content'));
					s.push('a2');
					r(s)
				});
			}`)
		assert.NoError(t, err)
		assert.Equal(t, []string{"a1", "a2"}, str)
	}

	{
		str, err := esmParser.GetStrings(ctx, "", `export default [{"foo":"1"}, {"bar":"1"}, 19]`)
		assert.NoError(t, err)
		assert.Equal(t, []string{`{"foo":"1"}`, `{"bar":"1"}`, "19"}, str)
	}
}

func TestESMGetElement(t *testing.T) {
	ele, err := esmParser.GetElement(ctx, ``, `
	export default (ctx) => {
		ctx.set('esm_size', 1 + 2);
		return ctx.get('esm_size');
	}
	`)
	assert.NoError(t, err)
	assert.Equal(t, "3", ele)
}

func TestESMGetElements(t *testing.T) {
	ele, err := esmParser.GetElements(ctx, ``, `export default [1, 2];`)
	assert.NoError(t, err)
	assert.Equal(t, []string{"1", "2"}, ele)
}
