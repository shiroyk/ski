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
	// numeric encodings
	"test-buffer-ascii.js":               "ascii encoding is not implemented",
	"test-buffer-slice.js":               "utf16le encoding is not implemented",
	"test-buffer-tostring-range.js":      "ascii encoding is not implemented",
	"test-buffer-write-utf8-two-byte.js": "ascii encoding is not implemented",
	"test-buffer-tostring.js":            "encoding names are case insensitive in node, UTF8 is not supported",

	// static methods
	"test-buffer-bigint64.js":                  "Buffer.allocUnsafe is not implemented",
	"test-buffer-copy.js":                      "Buffer.allocUnsafe is not implemented",
	"test-buffer-failed-alloc-typed-arrays.js": "Buffer.allocUnsafe and Buffer.allocUnsafeSlow are not implemented",
	"test-buffer-inspect.js":                   "Buffer.allocUnsafe and Buffer.allocUnsafeSlow are not implemented",
	"test-buffer-readdouble.js":                "Buffer.allocUnsafe is not implemented",
	"test-buffer-safe-unsafe.js":               "Buffer.allocUnsafe is not implemented",
	"test-buffer-slow.js":                      "Buffer.allocUnsafeSlow is not implemented",
	"test-buffer-writedouble.js":               "Buffer.allocUnsafe is not implemented",
	"test-buffer-writefloat.js":                "Buffer.allocUnsafe is not implemented",
	"test-buffer-zero-fill-cli.js":             "Buffer.allocUnsafe is not implemented",
	"test-buffer-isencoding.js":                "Buffer.isEncoding is not implemented",
	"test-buffer-isascii.js":                   "Buffer.isAscii is not implemented",
	"test-buffer-isutf8.js":                    "Buffer.isUtf8 is not implemented",
	"test-buffer-swap.js":                      "Buffer.prototype.swap16, swap32 and swap64 are not implemented",
	"test-buffer-parent-property.js":           "Buffer.prototype.parent is not implemented",
	"test-buffer-arraybuffer.js":               "Buffer.prototype.parent is not implemented",

	// node error objects carry a code, the ski errors do not
	"test-buffer-alloc.js":                  "node error codes and messages are not implemented",
	"test-buffer-bytelength.js":             "node error codes and messages are not implemented",
	"test-buffer-concat.js":                 "node error codes and messages are not implemented",
	"test-buffer-compare-offset.js":         "node error codes and messages are not implemented",
	"test-buffer-new.js":                    "node error codes and messages are not implemented",
	"test-buffer-no-negative-allocation.js": "node error codes and messages are not implemented",
	"test-buffer-read.js":                   "node error codes and messages are not implemented",
	"test-buffer-readfloat.js":              "node error codes and messages are not implemented",
	"test-buffer-readint.js":                "node error codes and messages are not implemented",
	"test-buffer-readuint.js":               "node error codes and messages are not implemented",
	"test-buffer-set-inspect-max-bytes.js":  "node error codes and messages are not implemented",
	"test-buffer-tostring-rangeerror.js":    "node error codes and messages are not implemented",
	"test-buffer-writeuint.js":              "node error codes and messages are not implemented",

	// arguments
	"test-buffer-compare.js":  "node accepts Uint8Array arguments, the ski Buffer does not",
	"test-buffer-equals.js":   "node accepts Uint8Array arguments, the ski Buffer does not",
	"test-buffer-includes.js": "node accepts string and Uint8Array arguments, the ski Buffer does not",
	"test-buffer-indexof.js":  "node accepts string and Uint8Array arguments, the ski Buffer does not",
	"test-buffer-writeint.js": "node accepts Uint8Array arguments, the ski Buffer does not",

	// conversions and generic methods
	"test-buffer-from.js":            "Buffer.from does not convert String objects",
	"test-buffer-tojson.js":          `Buffer.from does not support { type: "Buffer", data: [] } objects`,
	"test-buffer-badhex.js":          "Buffer.prototype.write hex parsing differs from node",
	"test-buffer-generic-methods.js": "Buffer generic methods on Uint8Array and ArrayBuffer differ from node",

	// deprecations
	"test-buffer-constructor-deprecation-error.js": "node deprecation warnings are not implemented",
	"test-buffer-pending-deprecation.js":           "node deprecation warnings are not implemented",

	// panics in the ski buffer module
	"test-buffer-fakes.js":           "fake Buffer objects panic on a type assertion in GetBuffer",
	"test-buffer-over-max-length.js": "panics with makeslice out of range instead of a RangeError",
	"test-buffer-write.js":           "panics with slice bounds out of range instead of a RangeError",
}

// TestNodeBuffer runs the upstream node buffer test suite.
func TestNodeBuffer(t *testing.T) {
	runNodeSuite(t, suite{
		pattern: filepath.Join(testsDir, "test-buffer-*.js"),
		skip:    bufferSkipTests,
		known:   bufferKnownFailures,
	})
}
