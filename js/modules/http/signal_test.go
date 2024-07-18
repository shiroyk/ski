package http

import (
	"fmt"
	"testing"

	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
)

func TestAbortSignal(t *testing.T) {
	vm := modulestest.New(t)

	testCases := []string{
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

	for i, s := range testCases {
		t.Run(fmt.Sprintf("Script%v", i), func(t *testing.T) {
			_, err := vm.Runtime().RunString(fmt.Sprintf(`{%s}`, s))
			assert.NoError(t, err)
		})
	}
}
