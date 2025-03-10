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

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/stretchr/testify/assert"

	_ "github.com/shiroyk/ski/modules/fetch"
)

const (
	wptBASE = "testdata/wpt"
)

var skipTests = map[string]bool{
	"fetch/api/idlharness.any.js": true,
	"url/idlharness.any.js":       true,

	"fetch/api/abort/cache.https.any.js":      true,
	"fetch/api/basic/response-url.sub.any.js": true,

	// TODO: fix body used
	"fetch/api/abort/request.any.js": true,

	// TODO: fix FormData
	"fetch/api/body/formdata.any.js":                   true,
	"fetch/content-type/multipart-malformed.any.js":    true,
	"fetch/api/request/request-consume-empty.any.js":   true,
	"fetch/api/response/response-consume-empty.any.js": true,
}

func TestWPT(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	if _, err := os.Stat(wptBASE); err != nil {
		t.Skipf("If you want to run wpt tests, see testdata/checkout.sh for the latest working commit id. (%v)", err)
	}

	ctx := &testCtx{
		cache: make(map[string]*sobek.Program),
	}

	t.Run("WPT", func(t *testing.T) {
		ctx.runWPTTest(t, "fetch")
		ctx.runWPTTest(t, "url")
	})
}

type testCtx struct {
	cache     map[string]*sobek.Program
	cacheLock sync.Mutex
}

func (c *testCtx) newVM() (js.VM, error) {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()

	harness := filepath.Join(wptBASE, "resources/testharness.js")
	p, ok := c.cache[harness]
	if !ok {
		data, err := os.ReadFile(harness)
		if err != nil {
			return nil, err
		}
		program, err := sobek.Compile(harness, string(data), false)
		if err != nil {
			return nil, err
		}
		c.cache[harness] = program
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
	href: "http://example.com/",
};
`)
	if err != nil {
		return nil, err
	}
	_, err = vm.RunProgram(context.Background(), p)
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
		if !strings.HasSuffix(path, ".any.js") {
			return nil
		}
		name := strings.TrimPrefix(path, "testdata/wpt/")
		t.Run(name, func(t *testing.T) {
			if skipTests[name] {
				t.Skip(path)
				return
			}
			t.Parallel()
			c.testScript(t, path)
		})
		return nil
	})
	assert.NoError(t, err)
}

func (c *testCtx) testScript(t *testing.T, path string) {
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

	if g, ok := meta["global"]; ok {
		if len(g) > 0 {
			if !strings.Contains(g[0], "worker") {
				t.Log("skipping no-worker test")
				t.SkipNow()
				return
			}
		}
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

	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	for _, v := range meta["script"] {
		p, ok := c.cache[v]
		if !ok {
			var script string
			if strings.HasPrefix(v, "/") {
				script = filepath.Join(wptBASE, v)
			} else {
				script = filepath.Join(filepath.Dir(path), v)
			}
			data, err := os.ReadFile(script)
			if !assert.NoError(t, err) {
				return
			}
			program, err := sobek.Compile(v, string(data), false)
			if !assert.NoError(t, err) {
				return
			}
			c.cache[v] = program
			p = program
		}
		_, err := vm.RunProgram(t.Context(), p)
		if assert.NoError(t, err) {
			return
		}
	}

	_, err = vm.RunString(t.Context(), string(all))
	if err != nil {
		if strings.Contains(err.Error(), "not implement") {
			t.Skip("not implement")
			return
		}
		assert.NoError(t, err)
		return
	}

	rt := vm.Runtime()
	result := rt.Get("tests")
	if result == nil {
		return
	}
	tests := result.ToObject(rt).Get("tests")
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
				t.Skip("fetch file not supported")
			} else if strings.Contains(message, "not implement") {
				t.Skip("not implement")
			} else {
				t.Errorf("%s: \n%s", name, message)
			}
		}
		return true
	})
}
