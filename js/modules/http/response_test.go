package http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/assert"
)

func TestResponse(t *testing.T) {
	vm := modulestest.New(t, initial)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/json":
			w.Header().Set("Content-Type", "application/json")
			_, err := fmt.Fprint(w, `{ "foo": "bar", "test": true }`)
			assert.NoError(t, err)
		case "/array":
			w.Header().Set("Content-Type", "application/json")
			_, err := fmt.Fprint(w, `[{ "foo": "bar", "test": true }]`)
			assert.NoError(t, err)
		case "/text":
			w.Header().Set("Content-Type", "text/plain")
			_, err := fmt.Fprint(w, `foo`)
			assert.NoError(t, err)
		}
	}))

	_ = vm.Runtime().Set("url", ts.URL)

	testCase := []string{
		`const res = http.get(url+'/array');
		 assert.true(res.ok);`,
		`const res = http.get(url+'/json');
		 assert.equal(res.json(), { "foo": "bar", "test": true });
		 assert.true(res.bodyUsed);
		 assert.true(res.ok);
		 assert.equal(res.status, 200);
		 assert.equal(res.statusText, "200 OK");
		 assert.equal(res.headers["Content-Type"], "application/json");`,
		`const res = http.get(url+'/array');
		 assert.equal(res.json(), [{ "foo": "bar", "test": true }]);
		 assert.true(res.bodyUsed);
		 assert.true(res.ok);
		 assert.equal(res.status, 200);
		 assert.equal(res.statusText, "200 OK");
		 assert.equal(res.headers["Content-Type"], "application/json");`,
		`const res = http.get(url+'/text');
		 assert.true(!res.bodyUsed);
		 assert.true(res.ok);
		 assert.equal(res.statusText, "200 OK");
		 assert.equal(res.text(), "foo");
		 assert.true(res.bodyUsed);
		 try {
			res.text();
		 } catch (e) {
		 	assert.true(e && e.toString().includes("body stream already read"));
		 }`,
	}

	for i, s := range testCase {
		t.Run(fmt.Sprintf("Script%v", i), func(t *testing.T) {
			_, err := vm.Runtime().RunString(fmt.Sprintf(`{%s}`, s))
			assert.NoError(t, err)
		})
	}
}

func TestAsyncResponse(t *testing.T) {
	vm := modulestest.New(t, initial)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/chunked":
			w.Header().Set("Content-Type", "text/plain")
			for i := 0; i < 6; i++ {
				_, err := fmt.Fprint(w, strconv.Itoa(i), "\r\n")
				assert.NoError(t, err)
				w.(http.Flusher).Flush()
				time.Sleep(time.Millisecond * 50)
			}
		case "/json":
			w.Header().Set("Content-Type", "application/json")
			_, err := fmt.Fprint(w, `{ "foo": "bar", "test": true }`)
			assert.NoError(t, err)
		case "/array":
			w.Header().Set("Content-Type", "application/json")
			_, err := fmt.Fprint(w, `[{ "foo": "bar", "test": true }]`)
			assert.NoError(t, err)
		case "/text":
			w.Header().Set("Content-Type", "text/plain")
			_, err := fmt.Fprint(w, `foo`)
			assert.NoError(t, err)
		}
	}))

	_ = vm.Runtime().Set("url", ts.URL)

	testCase := []string{
		`(async () => {
			const res = await fetch(url+'/chunked');
			const reader = res.body.getReader({ mode: "byob" });
			const chunks = [];
			let size = 0;
			const read = async () => {
				const { value, done } = await reader.read(new Uint8Array(4));
				if (done) return chunks.join("");
				const chunk = String.fromCharCode.apply(String, value);
				chunks.push(chunk);
				size++;
				return read();
			};
			assert.equal(await read(), "0\r\n1\r\n2\r\n3\r\n4\r\n5\r\n");
			assert.true(res.bodyUsed);
			assert.true(res.ok);
			assert.equal(res.status, 200);
			assert.equal(res.statusText, "200 OK");
			assert.equal(res.headers["Content-Type"], "text/plain");
		})()`,
		`(async () => {
			const res = await fetch(url+'/chunked');
			const reader = res.body.getReader();
			const chunks = [];
			let size = 0;
			while (true) {
				const { value, done } = await reader.read();
				if (done) break;
				const chunk = String.fromCharCode.apply(String, value);
				chunks.push(chunk);
				size++;
			}
			assert.equal(chunks.join(""), "0\r\n1\r\n2\r\n3\r\n4\r\n5\r\n");
			assert.true(res.bodyUsed);
		})()`,
		`(async () => {
			const res = await fetch(url+'/chunked');
			assert.equal(await res.arrayBuffer(), new Uint8Array([48, 13, 10, 49, 13, 10, 50, 13, 10, 51, 13, 10, 52, 13, 10, 53, 13, 10]));
			assert.true(res.bodyUsed);
		})()`,
		`(async () => {
			const res = await fetch(url+'/json');
			assert.equal(await res.json(), { "foo": "bar", "test": true });
			assert.true(res.bodyUsed);
		 })()`,
		`(async () => {
			const res = await fetch(url+'/array');
			assert.equal(await res.json(), [{ "foo": "bar", "test": true }]);
			assert.true(res.bodyUsed);
		 })()`,
		`(async () => {
			const res = await fetch(url+'/text');
			assert.equal(await res.text(), "foo");
			assert.true(res.bodyUsed);
		 })()`,
		`(async () => {
			const res = await fetch(url+'/text');
			assert.equal(await res.arrayBuffer(), new Uint8Array([102, 111, 111]));
			assert.true(res.bodyUsed);
		 })()`,
	}

	for i, s := range testCase {
		t.Run(fmt.Sprintf("Script%v", i), func(t *testing.T) {
			_, err := vm.RunString(context.Background(), s)
			assert.NoError(t, err)
		})
	}
}

type testBody struct {
	closed bool
}

func (*testBody) Read(p []byte) (n int, err error) {
	return 0, nil
}

func (b *testBody) Close() error {
	b.closed = true
	return nil
}

func TestAutoClose(t *testing.T) {
	vm := modulestest.New(t)
	body := new(testBody)
	res := NewResponse(vm.Runtime(), &http.Response{Body: body, StatusCode: http.StatusOK})
	ctx := context.WithValue(context.Background(), "res", res)
	assert.False(t, body.closed)
	v, err := vm.RunModule(ctx, `export default (ctx) => ctx.get('res').ok`)
	if assert.NoError(t, err) {
		assert.True(t, v.ToBoolean())
		assert.True(t, body.closed)
	}
}
