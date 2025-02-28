package ssr

import (
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"

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
		slog.Info("test server: http://" + server.Addr)
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
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()
	vm := modulestest.New(t)

	source := `
import { h, createSSRApp } from "https://unpkg.com/vue@3/dist/vue.runtime.esm-browser.js";
import { renderToString } from "https://unpkg.com/@vue/server-renderer@3/dist/server-renderer.esm-browser.js";

const app = createSSRApp({
	data: () => ({ count: 1 }),
	render() { return h('div', { onClick: () => this.count++ }, this.count) },
});

let html = await renderToString(app);
assert.regexp(html, '<div>1</div>');
`
	_, err := vm.RunModule(t.Context(), source)
	require.NoError(t, err)
}
