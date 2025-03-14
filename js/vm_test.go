package js

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/grafana/sobek"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVM(t *testing.T) {
	t.Run("basic execution", func(t *testing.T) {
		vm := NewVM()
		rt := vm.Runtime()

		result, err := rt.RunString("1 + 2")
		require.NoError(t, err)
		assert.Equal(t, int64(3), result.ToInteger())

		err = rt.Set("add", func(call sobek.FunctionCall) sobek.Value {
			a := call.Argument(0).ToInteger()
			b := call.Argument(1).ToInteger()
			return rt.ToValue(a + b)
		})
		require.NoError(t, err)

		result, err = rt.RunString("add(2, 3)")
		require.NoError(t, err)
		assert.Equal(t, int64(5), result.ToInteger())
	})

	t.Run("module execution", func(t *testing.T) {
		vm := NewVM()

		module, err := Loader().CompileModule("test", `
			export default function(a, b) { 
				return a + b 
			}
		`)
		require.NoError(t, err)

		result, err := vm.RunModule(context.Background(), module, 2, 3)
		require.NoError(t, err)
		assert.Equal(t, int64(5), result.ToInteger())
	})

	t.Run("context cancel", func(t *testing.T) {
		vm := NewVM()
		ctx, cancel := context.WithCancel(context.Background())
		time.AfterFunc(100*time.Millisecond, func() { cancel() })

		start := time.Now()
		_, err := vm.RunString(ctx, "{while(true){}}")
		assert.Error(t, err)
		assert.Less(t, time.Since(start), time.Millisecond*110)
		assert.True(t, errors.Is(err, context.Canceled))
	})

	t.Run("context timeout", func(t *testing.T) {
		vm := NewVM()
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		start := time.Now()
		_, err := vm.RunString(ctx, "{while(true){}}")
		assert.Error(t, err)
		assert.Less(t, time.Since(start), time.Millisecond*110)
		assert.True(t, errors.Is(err, context.DeadlineExceeded))
	})

	t.Run("promise handling", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"message":"ok"}`))
		}))
		defer server.Close()

		vm := NewVM()
		rt := vm.Runtime()

		err := rt.Set("fetch", func(call sobek.FunctionCall) sobek.Value {
			enqueue := EnqueueJob(rt)
			promise, resolve, reject := rt.NewPromise()
			go func() {
				res, err := http.Get(call.Argument(0).String())
				enqueue(func() error {
					if err != nil {
						return reject(err)
					} else {
						defer res.Body.Close()
						var data map[string]string
						err = json.NewDecoder(res.Body).Decode(&data)
						require.NoError(t, err)
						data["status"] = res.Status
						return resolve(data)
					}
				})
			}()
			return rt.ToValue(promise)
		})
		require.NoError(t, err)

		result, err := vm.RunString(context.Background(), `
			(async function() {
				const res = await fetch("`+server.URL+`");
				return res;
			})()
		`)
		require.NoError(t, err)

		value, err := Unwrap(result)
		require.NoError(t, err)
		assert.Equal(t, map[string]string{
			"status":  "200 OK",
			"message": "ok",
		}, value)
	})

	t.Run("error handling", func(t *testing.T) {
		vm := NewVM()

		_, err := vm.RunString(context.Background(), "1 + )")
		assert.Error(t, err)

		_, err = vm.RunString(context.Background(), "undefined.method()")
		assert.Error(t, err)

		assert.NotPanics(t, func() {
			var p []string
			err = vm.Run(context.Background(), func() error { p[1] = ""; return nil })
			assert.Error(t, err)
		})
	})

	t.Run("eventloop", func(t *testing.T) {
		vm := NewVM()
		rt := vm.Runtime()

		var results []int
		err := rt.Set("setTimeout", func(call sobek.FunctionCall) sobek.Value {
			enqueue := self(rt).eventloop.EnqueueJob()
			callback, _ := sobek.AssertFunction(call.Argument(0))
			time.AfterFunc(time.Duration(call.Argument(1).ToInteger())*time.Millisecond, func() {
				enqueue(func() error { callback(sobek.Undefined()); return nil })
			})
			return sobek.Undefined()
		})
		require.NoError(t, err)
		err = rt.Set("addResult", func(call sobek.FunctionCall) sobek.Value {
			results = append(results, int(call.Argument(0).ToInteger()))
			return sobek.Undefined()
		})
		require.NoError(t, err)

		_, err = vm.RunString(context.Background(), `
				setTimeout(() => addResult(1), 0);
				Promise.resolve().then(() => addResult(2));
				addResult(3);
			`)
		require.NoError(t, err)

		assert.Equal(t, []int{3, 2, 1}, results)
	})
}
