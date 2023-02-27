package console

import (
	"context"
	"testing"

	"github.com/shiroyk/cloudcat/js/modulestest"
	"github.com/stretchr/testify/assert"
)

func TestConsole(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	vm := modulestest.New(t)

	_, err := vm.RunString(ctx, `
		console.log("hello %s", "cloudcat")
	`)
	assert.NoError(t, err)
}
