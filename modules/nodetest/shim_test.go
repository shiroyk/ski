package nodetest

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/grafana/sobek"
)

// goShims maps the resolved module path suffix to the global that holds the go
// implemented shim, the file loader serves a CommonJS module exporting it.
var goShims = map[string]string{
	"/node_modules/process":            "__nodetest_process__",
	"/node_modules/process/index.js":   "__nodetest_process__",
	"/node_modules/util":               "__nodetest_util__",
	"/node_modules/util/index.js":      "__nodetest_util__",
	"/node_modules/buffer":             "__nodetest_buffer__",
	"/node_modules/buffer/index.js":    "__nodetest_buffer__",
	"/node_modules/vm":                 "__nodetest_vm__",
	"/node_modules/vm/index.js":        "__nodetest_vm__",
	"/node_modules/node:test":          "__nodetest_test__",
	"/node_modules/node:test/index.js": "__nodetest_test__",
}

// shimSource is the CommonJS module that the file loader serves for the go
// implemented shims, the global is created by initShims.
const shimSource = "module.exports = globalThis.%s;\n"

// assertPaths are the resolved paths of the node assert module, the
// implementation lives in js/modulestest and the vm sets the global.
var assertPaths = []string{
	"/node_modules/assert",
	"/node_modules/assert.js",
	"/node_modules/assert/index.js",
	"/node_modules/node:assert",
	"/node_modules/node:assert/index.js",
}

// commonPaths are the resolved paths of the upstream test/common helpers.
var commonPaths = []string{
	"/test/common",
	"/test/common.js",
	"/test/common/index.js",
}

// initShims publishes the go implemented shims on the runtime, it is called for
// every vm before the test file is loaded.
func initShims(rt *sobek.Runtime) {
	_ = rt.Set("__nodetest_process__", newProcessShim(rt))
	_ = rt.Set("__nodetest_util__", newUtilShim(rt))
	_ = rt.Set("__nodetest_buffer__", newBufferShim(rt))
	_ = rt.Set("__nodetest_vm__", newVMShim(rt))
	_ = rt.Set("__nodetest_test__", newNodeTestShim(rt))
}

// fileLoader serves the shims and falls back to the checked out node sources.
func fileLoader(specifier *url.URL, _ string) ([]byte, error) {
	path := specifier.Path
	switch {
	case matchPath(path, assertPaths):
		return []byte("module.exports = globalThis.assert;\n"), nil
	case matchPath(path, commonPaths):
		return []byte(commonShim), nil
	}
	for suffix, global := range goShims {
		if matchPath(path, []string{suffix}) {
			return fmt.Appendf(nil, shimSource, global), nil
		}
	}
	return os.ReadFile(path)
}

// matchPath reports whether the resolved module path is one of the paths, the
// loader may resolve it relative to the base url.
func matchPath(path string, paths []string) bool {
	for _, item := range paths {
		if strings.HasSuffix(path, item) || path == strings.TrimPrefix(item, "/") {
			return true
		}
	}
	return false
}
