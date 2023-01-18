package js

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHttp(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			if token := r.Header.Get("Authorization"); token != "1919810" {
				t.Errorf("unexpected token %s", token)
			}
		}
		w.Header().Set("Content-Type", "text/plain; charset=iso-8859-9")
		w.Header().Set("X-Total-Count", "114514")

		if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
			file, _, err := r.FormFile("file")
			if err != nil {
				t.Error(err)
			}

			body, err := io.ReadAll(file)
			if err != nil {
				t.Error(err)
			}

			_, err = fmt.Fprint(w, string(body))
			if err != nil {
				t.Error(err)
			}
		} else {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
			}

			_, err = fmt.Fprint(w, string(body))
			if err != nil {
				t.Error(err)
			}
		}
	}))
	defer ts.Close()

	_, _ = testVM.RunString(fmt.Sprintf(`const url = "%s";`, ts.URL))

	testCase := []string{
		`fa = new Uint8Array([226, 153, 130, 239, 184, 142]).buffer`,
		`mp = new FormData();
		 mp.set('file', fa);
		 mp.set('name', 'foo');
		 assert.equal(go.http.post(url, mp).string(), "♂︎");`,
		`form = new URLSearchParams({'key': 'holy', 'value': 'fa'});
		 assert.equal(go.http.post(url, form).string(), "key=holy&value=fa");`,
		`assert.equal(go.http.get(url).string(), "");`,
		`assert.equal(go.http.head(url).headers.get("X-Total-Count"), "114514");`,
		`assert.equal(go.http.post(url).string(), "");`,
		`assert.equal(new Uint8Array(go.http.post(url, '1').bytes())[0], 49);`,
		`assert.equal(go.http.post(url, {'dark': 'o'}).json()['dark'], "o");`,
		`assert.equal(go.http.post(url, "post").string(), "post");`,
		`assert.equal(go.http.request('PUT', url, "put", {"Authorization": "1919810"}).string(), "put");`,
		`assert.equal(go.http.request('PATCH', url, fa).string(), "♂︎");`,
		`assert.equal(go.http.request('PATCH', url, new Uint8Array([97]).buffer).string(), "a");`,
		`assert.equal(go.http.request('PATCH', url, "fa", null).string(), "fa");`,
		`try {
			go.http.request('PATCH', url, 114514, null);
		 } catch (e) {
			assert(e.toString().includes("unsupported request body"));
		 }`}

	for i, s := range testCase {
		t.Run(fmt.Sprintf("Script%v", i), func(t *testing.T) {
			_, err := testVM.RunString(s)
			if err != nil {
				t.Errorf("Script: %s , %s", s, err)
			}
		})
	}
}

func TestFormData(t *testing.T) {
	testCase := []string{
		`mp = new FormData();`,
		`try {
			mp = new FormData(0);
		 } catch (e) {
			assert(e.toString().includes('unsupported type'))
		 }`,
		`mp = new FormData({
			'file': new Uint8Array([50]).buffer,
			'name': 'foo'
		 });
		 assert.equal(mp.get('name'), 'foo')`,
		`mp.append('file', new Uint8Array([51]).buffer);
		 assert.equal(mp.getAll('file').length, 2)`,
		`mp.append('name', 'bar');
		 assert.equal(mp.keys().length, 2);
		 assert.equal(mp.get('name'), 'foo');`,
		`assert.equal(mp.entries().length, 4)`,
		`mp.delete('name');
		 assert.equal(mp.getAll('name').length, 0)`,
		`assert(!mp.has('name'))`,
		`mp.set('name', 'foobar');
		 assert.equal(mp.values().length, 2)`,
	}

	for i, s := range testCase {
		t.Run(fmt.Sprintf("Script%v", i), func(t *testing.T) {
			_, err := testVM.RunString(s)
			if err != nil {
				t.Errorf("Script: %s , %s", s, err)
			}
		})
	}
}

func TestURLSearchParams(t *testing.T) {
	testCase := []string{
		`form = new URLSearchParams();form.sort();`,
		`try {
			form = new URLSearchParams(0);
		 } catch (e) {
			assert(e.toString().includes('unsupported type'))
		 }`,
		`form = new URLSearchParams({'name': 'foo'});
		 form.forEach((v, k) => assert(v.length == 1))
		 assert.equal(form.get('name'), 'foo')`,
		`form.append('name', 'bar');
		 assert.equal(form.getAll('name').length, 2)`,
		`assert.equal(form.toString(), 'name=foo&name=bar')`,
		`form.append('value', 'zoo');
		 assert(compareArray(form.keys(), ['name', 'value']))`,
		`assert.equal(form.entries().length, 5)`,
		`form.delete('name');
		 assert.equal(form.getAll('name').length, 0)`,
		`assert(!form.has('name'))`,
		`form.set('name', 'foobar');
		 assert.equal(form.values().length, 2)`}

	for i, s := range testCase {
		t.Run(fmt.Sprintf("Script%v", i), func(t *testing.T) {
			_, err := testVM.RunString(s)
			if err != nil {
				t.Errorf("Script: %s , %s", s, err)
			}
		})
	}
}
