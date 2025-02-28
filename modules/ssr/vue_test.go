package ssr

import (
	"testing"

	"github.com/shiroyk/ski/js/modulestest"
	_ "github.com/shiroyk/ski/modules/fetch"
	"github.com/stretchr/testify/require"
)

func TestVueSSR(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()
	vm := modulestest.New(t)
	_ = vm.Runtime().Set("process", map[string]any{
		"env": map[string]any{
			"NODE_ENV": "development",
		},
	})

	source := `
import { h, createSSRApp } from "https://unpkg.com/vue@3/dist/vue.runtime.esm-browser.js";
import { renderToString } from "https://unpkg.com/@vue/server-renderer@3/dist/server-renderer.esm-browser.js";

const app = createSSRApp({
	data: () => ({ count: 1 }),
	render() { return h('div', { onClick: () => this.count++ }, this.count) },
});

let html = await renderToString(app);
assert.regexp(html, '<div>1</div>');
`
	_, err := vm.RunModule(t.Context(), source)
	require.NoError(t, err)
}
