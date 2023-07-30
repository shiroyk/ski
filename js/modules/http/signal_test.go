package http

import (
	"context"
	"fmt"
	"testing"

	"github.com/shiroyk/cloudcat/js/modulestest"
	"github.com/stretchr/testify/assert"
)

func TestAbortSignal(t *testing.T) {
	ctx := context.Background()
	vm := modulestest.New(t)

	testCase := []string{
		`signal = new AbortSignal();
		 signal.abort();
         assert.equal(signal.reason, "context canceled");
         assert.true(signal.aborted);`,
	}

	for i, s := range testCase {
		t.Run(fmt.Sprintf("Script%v", i), func(t *testing.T) {
			_, err := vm.RunString(ctx, s)
			assert.NoError(t, err)
		})
	}
}
