// Package nodetest runs the upstream Node.js test suites against the ski
// modules, the checkout is done by testdata/checkout.sh like the wpt tests do.
package nodetest

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nodeBASE is checked out by testdata/checkout.sh and ignored by git.
const nodeBASE = "testdata/node"

// testsDir is the directory of the upstream suites to run.
const testsDir = "test/parallel"

// suite describes an upstream node test suite.
type suite struct {
	// pattern is the file glob of the suite relative to nodeBASE.
	pattern string
	// skip are the tests that cannot run, with the reason, e.g. a test that
	// requires node internals.
	skip map[string]string
	// known are the tests that run but hit a feature that is not implemented,
	// the reason is reported instead of the failure.
	known map[string]string
}

// runNodeSuite runs every test file of the suite.
func runNodeSuite(t *testing.T, s suite) {
	if testing.Short() {
		t.Skip("skipping node test suites in short mode")
	}
	if _, err := os.Stat(nodeBASE); err != nil {
		t.Skipf("If you want to run the node tests, see testdata/checkout.sh for the latest working commit id. (%v)", err)
	}

	files, err := filepath.Glob(filepath.Join(nodeBASE, s.pattern))
	require.NoError(t, err)
	require.NotEmpty(t, files)

	for _, file := range files {
		name := filepath.Base(file)
		t.Run(strings.TrimSuffix(name, ".js"), func(t *testing.T) {
			if reason, ok := s.skip[name]; ok {
				t.Skip(reason)
			}
			if reason, ok := s.known[name]; ok {
				t.Skipf("known failure: %s", reason)
			}
			t.Parallel()
			runNodeTest(t, file)
		})
	}
}

// runNodeTest executes the test file as a CommonJS module, the test fails when
// it throws or when the common.mustCall() checks do not match.
func runNodeTest(t *testing.T, path string) {
	file, err := filepath.Abs(path)
	require.NoError(t, err)

	// every VM owns its loader, no global state is shared between the tests
	vm := modulestest.New(js.WithInitial(initShims))
	vm.SetFileLoader(fileLoader)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	// the process global is required by most test files
	_, err = vm.RunString(ctx, `require("process");`)
	require.NoError(t, err)

	// the loader resolves the absolute path of the test file from a file url,
	// a plain absolute path is resolved relative to the loader base
	specifier := (&url.URL{Scheme: "file", Path: file}).String()
	_, err = vm.RunString(ctx, fmt.Sprintf("require(%q);", specifier))
	if err != nil {
		if vm.Runtime().Get("__nodetest_skipped__") != nil {
			t.Skip(vm.Runtime().Get("__nodetest_skipped__").String())
		}
		t.Errorf("%v", err)
		return
	}

	_, err = vm.RunString(ctx, `globalThis.__nodetest_runExitChecks__?.();`)
	if err != nil {
		assert.NoError(t, err)
	}
}
