//go:build !race

package http

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// The test cannot run under race detector for some reason.
func TestHttpSignal(t *testing.T) {
	vm := createVM(t)
	_, err := vm.RunString(context.Background(), `
		const signal = new AbortSignal();
		fetch(url, { signal: signal, body: "sleep1000" }).catch(e => {});
		signal.abort();
		assert.equal(signal.reason, "context canceled");
		assert.true(signal.aborted);`)
	assert.NoError(t, err)
}
