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
)

const (
	wptBASE = "testdata/wpt"
)

var skipTests = map[string]bool{
	// not defined, not implemented
	"fetch/api/idlharness.any.js":                                true,
	"url/idlharness.any.js":                                      true,
	"fetch/content-length/api-and-duplicate-headers.any.js":      true,
	"fetch/api/abort/cache.https.any.js":                         true,
	"fetch/api/basic/request-upload.any.js":                      true,
	"fetch/api/basic/request-headers.any.js":                     true,
	"fetch/api/headers/header-values.any.js":                     true,
	"fetch/api/request/request-error.any.js":                     true,
	"fetch/api/request/request-cache-default-conditional.any.js": true,
	"fetch/api/response/response-stream-with-broken-then.any.js": true,

	// not support
	"fetch/api/basic/request-private-network-headers.tentative.any.js": true,
	// uft-8 bom
	"fetch/api/basic/text-utf8.any.js": true,

	// encode empty FormData https://github.com/web-platform-tests/wpt/pull/3950
	"fetch/api/request/request-consume-empty.any.js":   true,
	"fetch/api/response/response-consume-empty.any.js": true,

	// ???
	"fetch/api/body/mime-type.any.js": true,

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
self.GLOBAL = {
	isWorker: () => true,
	isShadowRealm: () => false,
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

	_, err = vm.RunString(ctx, string(all))
	if err != nil {
		if strings.Contains(err.Error(), "not implement") {
			t.Skip("not implement")
			return
		}
		assert.NoError(t, err)
		return
	}

	rt := vm.Runtime()

	tests := rt.Get("tests").ToObject(rt).Get("tests")
	if tests == nil {
		return
	}

	rt.ForOf(tests, func(v sobek.Value) (ok bool) {
		current := v.ToObject(rt)
		status := current.Get("status").ToInteger()
		name := current.Get("name").String()
		message := current.Get("message").String()
		if status != 0 {
			if strings.Contains(message, "unsupported protocol scheme") {
			} else if strings.Contains(message, "not implement") {
			} else {
				t.Errorf("%s: \n%s", name, message)
			}
		}
		return true
	})
}
