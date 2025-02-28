package ssr

import (
	"testing"

	"github.com/shiroyk/ski/js/modulestest"
	_ "github.com/shiroyk/ski/modules/encoding"
	"github.com/stretchr/testify/require"
)

func TestReactSSR(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()
	vm := modulestest.New(t)
	source := `
import React from "https://esm.sh/react@18";
import { renderToString } from "https://esm.sh/react-dom@18/server";

const app = React.createElement('div', null
	, React.createElement('h1', null, "React SSR Example"  )
	, React.createElement('p', null, "Current time: "  , new Date().toLocaleTimeString())
);

let html = renderToString(app);
assert.regexp(html, 'React SSR Example');
`
	_, err := vm.RunModule(t.Context(), source)
	require.NoError(t, err)
}
