package http

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/shiroyk/ski/js/promise"
	_ "github.com/shiroyk/ski/modules/timers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHttpServer(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t, js.WithInitial(func(rt *sobek.Runtime) {
		value, _ := new(Server).Instantiate(rt)
		_ = rt.Set("serve", value)
	}))

	t.Run("basic", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		_, err := vm.RunModule(ctx, `
		const s = serve((req) => new Response("ok"));
		const res = await fetch(s.url);
		assert.equal(await res.text(), "ok");
		await s.shutdown();
		`)
		require.NoError(t, err)
	})

	t.Run("port only", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		_, err := vm.RunModule(ctx, `
		const s = serve(3000, (req) => new Response("ok"));
		const res = await fetch(s.url);
		assert.equal(await res.text(), "ok");
		await s.shutdown();
		`)
		require.NoError(t, err)
	})

	t.Run("custom hostname", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		_, err := vm.RunModule(ctx, `
		const s = serve({
			port: 3000,
			hostname: "localhost",
			handler: (req) => new Response("ok")
		});
		const res = await fetch(s.url);
		assert.equal(await res.text(), "ok");
		await s.shutdown();
		`)
		require.NoError(t, err)
	})

	t.Run("error handling", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		_, err := vm.RunModule(ctx, `
		const s = serve({
			handler: (req) => { throw new Error("test error"); },
			onError: (err) => {
				assert.contains("test error", err.message);
				return new Response(err.message, { status: 500 });
			}
		});
		const res = await fetch(s.url);
		assert.equal(res.status, 500);
		assert.contains("test error", await res.text());
		await s.shutdown();
		`)
		require.NoError(t, err)
	})

	t.Run("error on onError", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		_, err := vm.RunModule(ctx, `
		const s = serve({
			handler: (req) => { throw new Error("test error") },
			onError: (err) => { throw new Error("onError error") }
		});
		const res = await fetch(s.url);
		assert.equal(res.status, 500);
		assert.contains("Internal Server Error", await res.text());
		await s.shutdown();
		`)
		require.NoError(t, err)
	})

	t.Run("error not response type", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		_, err := vm.RunModule(ctx, `
		const s = serve({
			handler: (req) => "ciallo",
			onError: (err) => "ciallo"
		});
		const res = await fetch(s.url);
		assert.equal(res.status, 500);
		assert.contains("Internal Server Error", await res.text());
		await s.shutdown();
		`)
		require.NoError(t, err)
	})

	t.Run("not response type", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		_, err := vm.RunModule(ctx, `
		const s = serve({
			handler: (req) => "ciallo",
		});
		const res = await fetch(s.url);
		assert.equal(res.status, 500);
		assert.contains("Internal Server Error", await res.text());
		await s.shutdown();
		`)
		require.NoError(t, err)
	})

	t.Run("request handling", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		_, err := vm.RunModule(ctx, `
		const s = serve(async (req) => {
			assert.equal(req.method, "POST");
			assert.equal(req.url, "/");
			await new Promise((r) => { setTimeout(r, 100) });
			assert.equal(await req.text(), "test");
			await new Promise((r) => { setTimeout(r, 100) });
			return Response.json({ foo: "bar" });
		});
		const res = await fetch(s.url, {
			method: "POST",
			body: new Blob(["test"])
		});
		assert.equal(res.status, 200);
		assert.equal(res.headers.get("content-type"), "application/json");
		assert.equal(await res.json(), { foo: "bar" });
		await s.shutdown();
		`)
		require.NoError(t, err)
	})

	t.Run("server shutdown", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		_, err := vm.RunModule(ctx, `
		const s = serve({
			handler: (req) => new Response("ok")
		});
		assert.equal(s.listening, true);
		await s.shutdown();
		assert.equal(s.listening, false);
		`)
		require.NoError(t, err)
	})

	t.Run("abort signal", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second*2)
		defer cancel()
		_, err := vm.RunModule(ctx, `
		const controller = new AbortController();
		const s = serve({
			signal: controller.signal,
			handler: (req) => new Response("ok")
		});
		controller.abort();
		await s.finished;
		assert.equal(s.listening, false);
		`)
		require.NoError(t, err)
	})

	t.Run("onListen callback", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		_, err := vm.RunModule(ctx, `
		const s = serve({
			onListen: ({ hostname, port }) => {
				assert.equal(hostname, "127.0.0.1");
				assert.equal(port, 8000);
			},
			handler: (req) => new Response("ok")
		});
		await s.shutdown();
		`)
		require.NoError(t, err)
	})

	t.Run("multiple servers", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		_, err := vm.RunModule(ctx, `
		const s1 = serve(8001, (req) => Response.json({ msg: "ok" }));
		const s2 = serve(async (req) => {
			const res = await fetch(s1.url);
			const json = await res.json();
			assert.equal(json, { msg: "ok" });
			return Response.json(json);
		});
		const res = await fetch(s2.url);
		assert.equal(await res.json(), { msg: "ok" });
		await s1.shutdown();
		await s2.shutdown();
		`)
		require.NoError(t, err)
	})

	t.Run("request concurrent", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second*10)
		defer cancel()

		goroutines := 100

		do := func(i int, wg *sync.WaitGroup, url string) {
			ctx2, cancel2 := context.WithTimeout(ctx, time.Second*10)
			defer cancel2()
			defer wg.Done()

			req, err := http.NewRequestWithContext(ctx2, http.MethodGet, url, nil)
			assert.NoError(t, err, "goroutine %d", i)
			res, err := http.DefaultClient.Do(req)
			if assert.NoError(t, err, "goroutine %d", i) {
				defer res.Body.Close()
				data, err := io.ReadAll(res.Body)
				assert.NoError(t, err, "goroutine %d", i)
				assert.Equal(t, `{"msg":"ok"}`, string(data), "goroutine %d", i)
			}
		}

		_ = vm.Runtime().Set("concurrent", func(call sobek.FunctionCall) sobek.Value {
			url := call.Argument(0).String()
			return promise.New(vm.Runtime(), func(callback promise.Callback) {
				var wg sync.WaitGroup
				start := time.Now()
				for i := range goroutines {
					wg.Add(1)
					go do(i, &wg, url)
				}
				wg.Wait()
				callback(func() (any, error) {
					return time.Since(start).String(), nil
				})
			})
		})

		_, err := vm.RunModule(ctx, `
		const s = serve(() => Response.json({ msg: "ok" }));
		const take = await concurrent(s.url);
		console.log("`+strconv.Itoa(goroutines)+` request", take);
		s.close();
		`)
		require.NoError(t, err)
	})
}
