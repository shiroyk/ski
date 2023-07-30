package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/shiroyk/cloudcat"
	"github.com/shiroyk/cloudcat/js/modulestest"
	"github.com/stretchr/testify/assert"
)

func TestHttp(t *testing.T) {
	cloudcat.Provide[cloudcat.Fetch](&http.Client{Transport: &http.Transport{Proxy: cloudcat.ProxyFromRequest}})
	ctx := context.Background()
	vm := modulestest.New(t)

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

	defer t.Cleanup(func() {
		ts.Close()
	})

	_, _ = vm.RunString(ctx, fmt.Sprintf(`
		const http = require('cloudcat/http');
		const fetch = require('cloudcat/fetch');
		const url = "%s";`, ts.URL))

	testCase := []string{
		`fa = new Uint8Array([226, 153, 130, 239, 184, 142]).buffer`,
		`mp = new FormData();
		 mp.set('file', fa);
		 mp.set('name', 'foo');
		 assert.equal(http.post(url, { body: mp }).string(), "♂︎");`,
		`form = new URLSearchParams({'key': 'holy', 'value': 'fa'});
		 assert.equal(http.post(url, { body: form }).string(), "key=holy&value=fa");`,
		`assert.equal(http.get(url).string(), "");`,
		`assert.equal(http.head(url).headers.get("X-Total-Count"), "114514");`,
		`assert.equal(http.post(url).string(), "");`,
		`assert.equal(new Uint8Array(http.post(url, { body: '1' }).bytes())[0], 49);`,
		`assert.equal(http.post(url, { body: {'dark': 'o'} }).json()['dark'], "o");`,
		`assert.equal(http.post(url, { body: "post" }).string(), "post");`,
		`assert.equal(fetch(url, { method: 'put', body: 'put', headers: {"Authorization": "1919810"} }).string(), "put");`,
		`assert.equal(fetch(url, { method: 'patch', body: fa }).string(), "♂︎");`,
		`assert.equal(fetch(url, { method: 'PATCH', body: new Uint8Array([97]).buffer }).string(), "a");`,
		`assert.equal(fetch(url, { method: 'custom' }).string(), "CUSTOM");`,
		`try {
			fetch(url, { method: 'put', body: 114514 });
		 } catch (e) {
			assert.true(e.toString().includes("unsupported request body"), e);
		 }`,
		`try {
			fetch(url, { proxy: 'http://127.0.0.1:22' });
		 } catch (e) {
			error = e.toString();
         	assert.true(error.includes("connect: connection refused") || error.includes("read: connection reset by peer"), error);
		 }`,
		`try {
			fetch(url, { signal: new AbortSignal(500), body: "sleep1000" });
		 } catch (e) {
         	assert.true(e.toString().includes("context deadline exceeded"), e);
		 }`,
	}

	for i, s := range testCase {
		t.Run(fmt.Sprintf("Script%v", i), func(t *testing.T) {
			_, err := vm.RunString(ctx, s)
			assert.NoError(t, err)
		})
	}
}
