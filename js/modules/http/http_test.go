package http

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
)

func TestHttp(t *testing.T) {
	vm := createVM(t)
	testCase := []string{
		`assert.equal(http.get(url).text(), "");`,
		`assert.equal(http.post(url, { body: new FormData({'file': fa, 'name': 'foo'}) }).text(), "♂︎");`,
		`assert.equal(http.post(url, { body: new URLSearchParams({'key': 'holy', 'value': 'fa'}) }).text(), "key=holy&value=fa");`,
		`assert.equal(http.head(url).headers["X-Total-Count"], "114514");`,
		`assert.equal(http.post(url).text(), "");`,
		`assert.equal(new Uint8Array(http.post(url, { body: '1' }).arrayBuffer())[0], 49);`,
		`assert.equal(http.post(url, { body: {'dark': 'o'} }).json()['dark'], "o");`,
		`assert.equal(http.post(url, { body: "post" }).text(), "post");`,
		`fetch(url, { method: 'put', body: 'put', headers: {"Authorization": "1919810"} })
		 .then(res => res.text())
		 .then(body => assert.equal(body, "put"));`,
		`fetch(url, { method: 'patch', body: fa })
		 .then(res => res.text())
		 .then(body => assert.equal(body, "♂︎"));`,
		`fetch(url, { method: 'PATCH', body: new Uint8Array([97]) })
		 .then(res => res.text())
		 .then(body => assert.equal(body, "a"));`,
		`fetch(url, { method: 'custom' })
		 .then(res => res.text())
		 .then(body => assert.equal(body,  "CUSTOM"));`,
		`fetch(url, { proxy: proxyURL })
		 .then(res => res.text())
		 .then(body => assert.equal(body, "proxy ok"))`,
		`try {
			fetch(url, { method: 'put', body: 114514 })
		 } catch (e) {
			assert.true(e.toString().includes("unsupported request body"), e.toString());
		 }`,
		`const controller = new AbortController();
		 fetch(url, { signal: controller.signal, body: "sleep1000" }).catch(e => {});
		 controller.abort();
		 assert.equal(controller.reason, "context canceled");
		 assert.true(controller.aborted);`,
		`(async () => {
			try {
				await fetch(url, { signal: AbortSignal.timeout(500), body: "sleep1000" });
			} catch (e) {
				assert.true(e.toString().includes("context deadline exceeded"), e);
			}
		 })()`,
	}

	for i, s := range testCase {
		t.Run(fmt.Sprintf("Script%v", i), func(t *testing.T) {
			_, err := vm.Runtime().RunString(s)
			assert.NoError(t, err)
		})
	}
}

var initial = js.WithInitial(func(rt *sobek.Runtime) {
	client := http.Client{Transport: &http.Transport{Proxy: ski.ProxyFromRequest}}
	instance, _ := (&Http{&client}).Instantiate(rt)
	_ = rt.Set("http", instance)
	f, _ := (&Fetch{&client}).Instantiate(rt)
	_ = rt.Set("fetch", f)
})

func createVM(t *testing.T) modulestest.VM {
	vm := modulestest.New(t, initial)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			if token := r.Header.Get("Authorization"); token != "1919810" {
				t.Errorf("unexpected token %s", token)
			}
		}
		if r.Method == "CUSTOM" {
			_, err := fmt.Fprint(w, "CUSTOM")
			assert.NoError(t, err)
		}
		w.Header().Set("Content-Type", "text/plain; charset=iso-8859-9")
		w.Header().Set("X-Total-Count", "114514")

		isMp := strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data")

		if isMp {
			file, _, err := r.FormFile("file")
			assert.NoError(t, err)

			body, err := io.ReadAll(file)
			assert.NoError(t, err)

			_, err = fmt.Fprint(w, string(body))
			assert.NoError(t, err)
		} else {
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err)

			if strings.HasPrefix(string(body), "sleep") {
				duration, err := strconv.Atoi(string(body)[5:])
				assert.NoError(t, err)
				time.Sleep(time.Duration(duration))
			}

			_, err = fmt.Fprint(w, string(body))
			assert.NoError(t, err)
		}
	}))
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprint(w, "proxy ok")
		assert.NoError(t, err)
	}))

	t.Cleanup(func() {
		ts.Close()
		proxy.Close()
	})

	_, _ = vm.Runtime().RunString(fmt.Sprintf(`
		const url = "%s";
		const proxyURL = "%s";
		const fa = new Uint8Array([226, 153, 130, 239, 184, 142])`, ts.URL, proxy.URL))

	return vm
}
