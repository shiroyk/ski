package dom

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/shiroyk/ski/js/promise"
	_ "github.com/shiroyk/ski/modules/fetch"
	"github.com/stretchr/testify/require"
)

func httpServer(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	handler, ok := sobek.AssertFunction(call.Argument(0))
	if !ok {
		panic(rt.NewTypeError("argument must be a function"))
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}

	server := &http.Server{
		Addr: l.Addr().String(),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			done := make(chan struct{})
			js.EnqueueJob(rt)(func() error {
				v, err := handler(sobek.Undefined(), rt.ToValue(func(body string) {
					io.WriteString(w, body)
					close(done)
				}), rt.ToValue(r.URL.Path))
				if err == nil {
					_, err = promise.Result(v)
				}
				if err == nil {
					return nil
				}
				w.WriteHeader(http.StatusInternalServerError)
				io.WriteString(w, err.Error())
				close(done)
				return nil
			})
			<-done
		}),
	}

	enqueue := js.EnqueueJob(rt)

	go func() {
		slog.Info("test server: " + server.Addr)
		_ = server.Serve(l)
		enqueue(func() error { return nil })
	}()

	js.Cleanup(rt, func() { server.Close() })

	ret := rt.NewObject()
	_ = ret.Set("url", server.Addr)
	_ = ret.Set("close", func() { server.Close() })

	return ret
}

func TestVueSSR(t *testing.T) {
	vm := modulestest.New(t)
	_ = vm.Runtime().Set("server", httpServer)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	source := `
import { h, createSSRApp } from "https://unpkg.com/vue@3/dist/vue.runtime.esm-browser.js";
import { renderToString } from "https://unpkg.com/@vue/server-renderer@3/dist/server-renderer.esm-browser.js";

const s = server(async (ok) => {
	const app = createSSRApp({
		data: () => ({ count: 1 }),
		render() { return h('div', { onClick: () => this.count++ }, this.count) },
	});

	const html = await renderToString(app);
	ok(` + "`" + `
		<!DOCTYPE html>
		<html>
		  <head>
			<title>Vue SSR Example</title>
			<script type="importmap">
			  {
				"imports": {
				  "vue": "https://unpkg.com/vue@3/dist/vue.esm-browser.js"
				}
			  }
			</script>
			<script type="module">
				import { h, createSSRApp } from 'vue';
				createSSRApp({
					data: () => ({ count: 1 }),
					render() { return h('div', { onClick: () => this.count++ }, this.count) },
				}).mount('#app');
			</script>
		  </head>
		  <body>
			<div id="app">${html}</div>
		  </body>
		</html>` + "`" + `);
	});

const body = await (await fetch("http://"+s.url)).text();
if (!/<div>1<\/div>/.exec(body)) {
	throw new Error('ssr render failed: ' + body);
}
s.close();
`
	_, err := vm.RunModule(ctx, source)
	require.NoError(t, err)
}
