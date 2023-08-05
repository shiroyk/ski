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
		`const controller = new AbortController();
		 controller.abort();
         assert.equal(controller.reason, "context canceled");
         assert.true(controller.aborted);`,
		`const signal = AbortSignal.abort();
         assert.equal(signal.reason, "context canceled");
         assert.true(signal.aborted);`,
		`const signal = AbortSignal.timeout(100);
         assert.equal(signal.reason, "");
         assert.true(!signal.aborted);`,
	}

	for i, s := range testCase {
		t.Run(fmt.Sprintf("Script%v", i), func(t *testing.T) {
			_, err := vm.RunString(ctx, s)
			assert.NoError(t, err)
		})
	}
}
