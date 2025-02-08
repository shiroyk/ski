package main

import (
	"context"
	"testing"

	"github.com/shiroyk/ski"
	"github.com/shiroyk/ski/js"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsage(t *testing.T) {
	module, err := js.CompileModule(`module`, `
        import { default as $ } from "ski/gq";

        export default function () {
            return $('<div><span>ciallo</ span></ div>').find('span').text();
        }
    `)
	if err != nil {
		panic(err)
	}

	result, err := ski.RunModule(context.Background(), module)
	require.NoError(t, err)
	assert.Equal(t, "ciallo", result.Export())
}
