package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shiroyk/cloudcat/core"
	"github.com/shiroyk/cloudcat/core/js/modulestest"
	"github.com/shiroyk/cloudcat/fetch"
	"github.com/stretchr/testify/assert"
)

func TestHttp(t *testing.T) {
	core.Provide(fetch.NewFetcher(fetch.Options{}))
	ctx := context.Background()
	vm := modulestest.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			if token := r.Header.Get("Authorization"); token != "1919810" {
				t.Errorf("unexpected token %s", token)
			}
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

			_, err = fmt.Fprint(w, string(body))
			assert.NoError(t, err)
		}
	}))

	defer t.Cleanup(func() {
		ts.Close()
	})

	_, _ = vm.RunString(ctx, fmt.Sprintf(`
		const http = require('cloudcat/http');
		const url = "%s";`, ts.URL))

	testCase := []string{
		`fa = new Uint8Array([226, 153, 130, 239, 184, 142]).buffer`,
		`mp = new FormData();
		 mp.set('file', fa);
		 mp.set('name', 'foo');
		 assert.equal(http.post(url, mp).string(), "♂︎");`,
		`form = new URLSearchParams({'key': 'holy', 'value': 'fa'});
		 assert.equal(http.post(url, form).string(), "key=holy&value=fa");`,
		`assert.equal(http.get(url).string(), "");`,
		`assert.equal(http.head(url).headers.get("X-Total-Count"), "114514");`,
		`assert.equal(http.post(url).string(), "");`,
		`assert.equal(new Uint8Array(http.post(url, '1').bytes())[0], 49);`,
		`assert.equal(http.post(url, {'dark': 'o'}).json()['dark'], "o");`,
		`assert.equal(http.post(url, "post").string(), "post");`,
		`assert.equal(http.request('PUT', url, "put", {"Authorization": "1919810"}).string(), "put");`,
		`assert.equal(http.request('PATCH', url, fa).string(), "♂︎");`,
		`assert.equal(http.request('PATCH', url, new Uint8Array([97]).buffer).string(), "a");`,
		`assert.equal(http.request('PATCH', url, "fa", null).string(), "fa");`,
		`try {
			http.request('PATCH', url, 114514, null);
		 } catch (e) {
			assert.true(e.toString().includes("unsupported request body"));
		 }`,
	}

	for i, s := range testCase {
		t.Run(fmt.Sprintf("Script%v", i), func(t *testing.T) {
			_, err := vm.RunString(ctx, s)
			assert.NoError(t, err)
		})
	}
}
