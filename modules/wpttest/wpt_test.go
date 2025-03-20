package wpttest

import (
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/stretchr/testify/assert"

	_ "github.com/shiroyk/ski/modules/encoding"
	_ "github.com/shiroyk/ski/modules/fetch"
	_ "github.com/shiroyk/ski/modules/signal"
	_ "github.com/shiroyk/ski/modules/timers"
)

const (
	wptBASE = "testdata/wpt"
)

var skipTests = map[string]bool{
	// not defined, not implemented
	"fetch/api/idlharness.any.js":                                true,
	"url/idlharness.any.js":                                      true,
	"url/historical.any.js":                                      true,
	"fetch/content-length/api-and-duplicate-headers.any.js":      true,
	"fetch/api/abort/cache.https.any.js":                         true,
	"fetch/api/basic/request-upload.any.js":                      true,
	"fetch/api/basic/request-headers.any.js":                     true,
	"fetch/api/headers/header-values.any.js":                     true,
	"fetch/api/request/request-error.any.js":                     true,
	"fetch/api/request/request-cache-default-conditional.any.js": true,
	"fetch/api/response/response-stream-with-broken-then.any.js": true,
	"fetch/http-cache/credentials.tentative.any.js":              true,

	// ???
	"fetch/api/body/mime-type.any.js": true,
	// not await
	"fetch/api/request/request-disturbed.any.js": true,

	// not support
	"fetch/api/basic/request-private-network-headers.tentative.any.js": true,
	// uft-8 bom
	"fetch/api/basic/text-utf8.any.js": true,
	// GET with body
	"fetch/api/request/request-init-002.any.js": true,

	// encode empty FormData https://github.com/web-platform-tests/wpt/pull/3950
	"fetch/api/request/request-consume-empty.any.js":   true,
	"fetch/api/response/response-consume-empty.any.js": true,

	// custom headers
	"fetch/api/request/request-headers.any.js": true,
	// priority
	"fetch/api/request/request-init-priority.any.js": true,
	// Headers immutable
	"fetch/api/response/response-static-error.any.js": true,

	// iterator
	"fetch/api/headers/headers-basic.any.js":  true,
	"fetch/api/headers/headers-record.any.js": true,
	// iterator, order
	"fetch/api/headers/header-setcookie.any.js": true,
	"fetch/api/headers/headers-combine.any.js":  true,
	// valid name, values
	"fetch/api/headers/headers-errors.any.js": true,

	// TODO: host info
	"fetch/api/basic/response-url.sub.any.js":                      true,
	"fetch/api/redirect/redirect-mode.any.js":                      true,
	"fetch/api/cors/cors-redirect-credentials.any.js":              true,
	"fetch/api/basic/integrity.sub.any.js":                         true,
	"fetch/api/cors/cors-preflight-star.any.js":                    true,
	"fetch/cross-origin-resource-policy/fetch.any.js":              true,
	"fetch/api/basic/mode-same-origin.any.js":                      true,
	"fetch/api/cors/cors-cookies-redirect.any.js":                  true,
	"fetch/api/cors/cors-filtering.sub.any.js":                     true,
	"fetch/cross-origin-resource-policy/fetch.https.any.js":        true,
	"fetch/orb/tentative/known-mime-type.sub.any.js":               true,
	"fetch/api/basic/mode-no-cors.sub.any.js":                      true,
	"fetch/api/basic/referrer.any.js":                              true,
	"fetch/api/basic/scheme-others.sub.any.js":                     true,
	"fetch/api/cors/cors-preflight.any.js":                         true,
	"fetch/api/cors/cors-basic.any.js":                             true,
	"fetch/api/cors/cors-cookies.any.js":                           true,
	"fetch/api/cors/cors-expose-star.sub.any.js":                   true,
	"fetch/api/cors/cors-multiple-origins.sub.any.js":              true,
	"fetch/api/credentials/authentication-redirection.any.js":      true,
	"fetch/api/redirect/redirect-to-dataurl.any.js":                true,
	"fetch/cross-origin-resource-policy/syntax.any.js":             true,
	"fetch/metadata/fetch-preflight.https.sub.any.js":              true,
	"fetch/metadata/fetch.https.sub.any.js":                        true,
	"fetch/metadata/trailing-dot.https.sub.any.js":                 true,
	"fetch/orb/tentative/content-range.sub.any.js":                 true,
	"fetch/orb/tentative/nosniff.sub.any.js":                       true,
	"fetch/orb/tentative/status.sub.any.js":                        true,
	"fetch/orb/tentative/unknown-mime-type.sub.any.js":             true,
	"fetch/api/cors/cors-preflight-cache.any.js":                   true,
	"fetch/api/redirect/redirect-back-to-original-origin.any.js":   true,
	"fetch/cross-origin-resource-policy/scheme-restriction.any.js": true,
	"fetch/range/general.any.js":                                   true,

	// TODO: test timeout
	"fetch/api/request/request-bad-port.any.js": true,
	"fetch/api/basic/request-upload.h2.any.js":  true,

	// TODO: events
	"fetch/api/abort/general.any.js": true,

	// TODO: fix body used
	"fetch/api/abort/request.any.js": true,

	// TODO: fix stream
	"fetch/api/response/response-consume-stream.any.js":     true,
	"fetch/api/response/response-stream-disturbed-5.any.js": true,

	// TODO: fix no cors
	"fetch/api/headers/headers-no-cors.any.js": true,

	// TODO: fix URL strip
	"url/url-setters-stripping.any.js": true,

	// TODO: valid characters
	"url/urlencoded-parser.any.js":           true,
	"url/urlsearchparams-constructor.any.js": true,
	"url/url-statics-parse.any.js":           true,
	"url/url-statics-canparse.any.js":        true,
	"url/urlsearchparams-sort.any.js":        true,
	"url/urlsearchparams-stringifier.any.js": true,

	// TODO: iterator
	"url/urlsearchparams-delete.any.js":  true,
	"url/urlsearchparams-foreach.any.js": true,

	// TODO: valid type characters
	"FileAPI/blob/Blob-constructor.any.js": true,
	"FileAPI/file/File-constructor.any.js": true,
}

var ignoreErrors = []string{
	"unsupported protocol scheme",
	"not implement",
	"duplex",
	"isReloadNavigation",
	"getSetCookie",
	"structuredClone",
	"MessageChannel",
}

func TestWPT(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	if _, err := os.Stat(wptBASE); err != nil {
		t.Skipf("If you want to run wpt tests, see testdata/checkout.sh for the latest working commit id. (%v)", err)
	}

	ctx := new(testCtx)

	t.Run("WPT", func(t *testing.T) {
		ctx.runWPTTest(t, "fetch")
		ctx.runWPTTest(t, "url")
		ctx.runWPTTest(t, "FileAPI/blob")
		ctx.runWPTTest(t, "FileAPI/file")
	})
}

type testCtx struct {
	cache sync.Map
}

func (c *testCtx) newVM() (js.VM, error) {
	harness := filepath.Join(wptBASE, "resources/testharness.js")
	p, ok := c.cache.Load(harness)
	if !ok {
		data, err := os.ReadFile(harness)
		if err != nil {
			return nil, err
		}
		src := string(data)
		src = strings.Replace(src, "var tests = new Tests();", "var tests = new Tests();self.tests = tests;", 1)
		program, err := sobek.Compile(harness, src, false)
		if err != nil {
			return nil, err
		}
		c.cache.Store(harness, program)
		p = program
	}

	vm := js.NewVM()
	_, err := vm.RunString(context.Background(), `var self = this;
Date.prototype.toGMTString = () => "Sat, 01 Jan 2000 00:00:00 GMT";
self.GLOBAL = {
	isWorker: () => true,
	isShadowRealm: () => true,
	isWindow: () => false,
};
location = {
  "ancestorOrigins": {},
  "href": "https://example.com/",
  "origin": "https://example.com",
  "protocol": "https:",
  "host": "example.com",
  "hostname": "example.com",
  "port": "80",
  "pathname": "/",
  "search": "",
  "hash": ""
};
`)
	if err != nil {
		return nil, err
	}
	_, err = vm.RunProgram(context.Background(), p.(*sobek.Program))
	if err != nil {
		return nil, err
	}
	return vm, nil
}

func (c *testCtx) runWPTTest(t *testing.T, dir string) {
	err := filepath.WalkDir(filepath.Join(wptBASE, dir), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, "any.js") {
			return nil
		}
		name := strings.TrimPrefix(path, "testdata/wpt/")
		t.Run(name, func(t *testing.T) {
			if skipTests[name] {
				t.Skip(path)
				return
			}
			c.testScript(t, path)
		})
		return nil
	})
	assert.NoError(t, err)
}

func (c *testCtx) testScript(t *testing.T, path string) {
	t.Parallel()
	file, err := os.Open(path)
	if !assert.NoError(t, err) {
		return
	}

	reader := bufio.NewReader(file)
	meta := make(map[string][]string)

	for {
		line, err := reader.ReadString('\n')
		if !assert.NoError(t, err) {
			return
		}
		if s, ok := strings.CutPrefix(line, "// META: "); ok {
			key, value, _ := strings.Cut(s, "=")
			meta[key] = append(meta[key], strings.TrimSuffix(value, "\n"))
		} else {
			break
		}
	}

	if g, ok := meta["global"]; ok && len(g) > 0 && !strings.Contains(g[0], "worker") {
		t.Log("skipping no-worker test")
		t.SkipNow()
		return
	}

	_, err = file.Seek(0, io.SeekStart)
	if !assert.NoError(t, err) {
		return
	}
	all, err := io.ReadAll(file)
	if !assert.NoError(t, err) {
		return
	}

	vm, err := c.newVM()
	if !assert.NoError(t, err) {
		return
	}

	for _, v := range meta["script"] {
		var script string
		if strings.HasPrefix(v, "/") {
			script = filepath.Join(wptBASE, v)
		} else {
			script = filepath.Join(filepath.Dir(path), v)
		}
		p, ok := c.cache.Load(script)
		if !ok {
			data, err := os.ReadFile(script)
			if !assert.NoError(t, err) {
				return
			}
			program, err := sobek.Compile(v, string(data), false)
			if !assert.NoError(t, err) {
				return
			}
			c.cache.Store(script, program)
			p = program
		}
		_, err := vm.RunProgram(t.Context(), p.(*sobek.Program))
		if !assert.NoError(t, err) {
			return
		}
	}

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	result, ok := sobek.AssertFunction(vm.Runtime().Get("add_result_callback"))
	if !ok {
		t.Logf("add_result_callback is not function")
		t.FailNow()
	}

	_, err = result(sobek.Undefined(), vm.Runtime().ToValue(func(call sobek.FunctionCall) sobek.Value {
		test := call.Argument(0).ToObject(vm.Runtime())
		status := test.Get("status").ToInteger()
		name := test.Get("name").String()
		message := test.Get("message").String()
		if status != 0 {
			for _, s := range ignoreErrors {
				if strings.Contains(message, s) {
					return nil
				}
			}
			t.Errorf("%s: \n%s", name, message)
		}
		return nil
	}))
	assert.NoError(t, err)

	_, err = vm.RunString(ctx, string(all))
	if err != nil {
		for _, s := range ignoreErrors {
			if strings.Contains(err.Error(), s) {
				t.Skip(s)
				return
			}
		}
		assert.NoError(t, err)
		return
	}
}
