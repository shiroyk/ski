package modulestest

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/promise"
	"github.com/stretchr/testify/require"
)

func TestHttpServer(t *testing.T) {
	t.Parallel()
	vm := New(t)
	_ = vm.Runtime().Set("createServer", HttpServer)
	_ = vm.Runtime().Set("post", func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
		url := call.Argument(0).String()
		reader := strings.NewReader(call.Argument(1).String())
		return promise.New(rt, func(callback promise.Callback) {
			ctx, cancel := context.WithTimeout(t.Context(), time.Second*5)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, "GET", url, reader)
			if err != nil {
				js.Throw(rt, err)
			}
			req.Header.Set("Accept", "application/json")

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				js.Throw(rt, err)
			}
			defer res.Body.Close()

			data, err := io.ReadAll(res.Body)
			if err != nil {
				js.Throw(rt, err)
			}
			callback(func() (any, error) {
				return map[string]any{
					"body":    string(data),
					"status":  res.StatusCode,
					"headers": res.Header,
				}, nil
			})
		})
	})

	source := `
const server = createServer((req, res) => {
	assert.equal(req.url, '/test?foo=bar');
	assert.equal(req.method, 'GET');
	assert.equal(req.path, '/test');
	assert.equal(req.protocol, 'HTTP/1.1');
	assert.equal(req.body.json(), {"foo": "bar"});
	assert.equal(req.getHeader("accept"), 'application/json');
	res.statusCode = 400;
	res.setHeader('Content-Type', 'application/json');
	res.end('{"hello": "world"}');
});

const res = await post(server.url+"/test?foo=bar", '{"foo": "bar"}');
assert.equal(res.status, 400);
assert.equal(res.body, '{"hello": "world"}');
assert.equal(res.headers.get('content-type'), 'application/json');
server.close();
`
	_, err := vm.RunModule(t.Context(), source)
	require.NoError(t, err)
}
