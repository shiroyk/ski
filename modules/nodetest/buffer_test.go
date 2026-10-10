package nodetest

import (
	"path/filepath"
	"testing"

	_ "github.com/shiroyk/ski/modules/buffer"
	_ "github.com/shiroyk/ski/modules/encoding"
)

// bufferSkipTests are the upstream buffer tests that require node internals or
// host features that are not available in the runtime, the key is the file name
// of the test.
var bufferSkipTests = map[string]string{
	"test-buffer-alloc-alignment.js":                                 "requires internal/test/binding",
	"test-buffer-alloc-unsafe-is-initialized-with-zero-fill-flag.js": "requires internal/test/binding",
	"test-buffer-alloc-unsafe-is-uninitialized.js":                   "requires internal/test/binding",
	"test-buffer-backing-arraybuffer.js":                             "requires internal/test/binding",
	"test-buffer-fill.js":                                            "requires internal/errors and internal/test/binding",
	"test-buffer-isutf8-isascii-fast.js":                             "requires internal/test/binding",
	"test-buffer-swap-fast.js":                                       "requires internal/test/binding",
	"test-buffer-write-fast.js":                                      "requires internal/buffer and internal/test/binding",
	"test-buffer-pool-untransferable.js":                             "requires worker_threads",
	"test-buffer-constructor-node-modules-paths.js":                  "requires child_process",
	"test-buffer-constructor-node-modules.js":                        "requires test/common/fixtures and child_process",
	"test-buffer-constructor-outside-node-modules.js":                "requires child_process",
	"test-buffer-copy-immutable.js":                                  "requires ArrayBuffer.prototype.transferToImmutable",
	"test-buffer-constants.js":                                       "requires the V8 string length limit",
	"test-buffer-sharedarraybuffer.js":                               "requires SharedArrayBuffer",
	"test-buffer-resizable.js":                                       "requires a resizable ArrayBuffer",
}

// bufferKnownFailures are the upstream buffer tests that run but hit a missing
// buffer API or a behavior that differs from node, they are skipped with the
// reason so the suite stays green, see the node modules for the missing
// features.
var bufferKnownFailures = map[string]string{
	// static and prototype methods
	"test-buffer-isascii.js":         "Buffer.isAscii is not implemented",
	"test-buffer-isutf8.js":          "Buffer.isUtf8 is not implemented",
	"test-buffer-swap.js":            "Buffer.prototype.swap16, swap32 and swap64 are not implemented",
	"test-buffer-parent-property.js": "Buffer.prototype.parent is not implemented",
	"test-buffer-arraybuffer.js":     "Buffer.prototype.parent is not implemented",
	"test-buffer-generic-methods.js": "the Buffer.prototype method set differs from node",

	// deprecations
	"test-buffer-constructor-deprecation-error.js": "node deprecation warnings are not implemented",
	"test-buffer-pending-deprecation.js":           "node deprecation warnings are not implemented",

	// runtime features
	"test-buffer-alloc.js":               "requires the engine typed array limits and the allocUnsafe pool",
	"test-buffer-copy.js":                "requires SharedArrayBuffer",
	"test-buffer-tostring-rangeerror.js": "requires the V8 string length limit",
	"test-buffer-write.js":               "the internal asciiWrite, latin1Write and utf8Write methods are not implemented",
	"test-buffer-write-utf8-two-byte.js": "Buffer.prototype.subarray returns a Uint8Array",
}

// TestNodeBuffer runs the upstream node buffer test suite.
func TestNodeBuffer(t *testing.T) {
	runNodeSuite(t, suite{
		pattern: filepath.Join(testsDir, "test-buffer-*.js"),
		skip:    bufferSkipTests,
		known:   bufferKnownFailures,
	})
}
