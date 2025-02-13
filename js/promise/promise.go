package promise

import (
	"fmt"
	"log/slog"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
)

// New return a sobek.Promise object.
// The second argument is a long-running asynchronous task that will be executed in a child goroutine.
// The third optional argument is a callback that will be executed in the main goroutine.
// Additional arguments will be ignored.
// like this:
//
//	func main() {
//		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//			w.WriteHeader(http.StatusOK)
//			_, _ = w.Write([]byte(`{"foo":"bar"}`))
//		}))
//		defer server.Close()
//
//		vm := js.NewVM()
//		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
//		defer cancel()
//
//		fetch := func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
//			return rt.ToValue(promise.New(rt,
//				func() (*http.Response, error) { return http.Get(call.Argument(0).String()) },
//				func(res *http.Response, err error) (any, error) {
//					if err != nil {
//						return nil, err
//					}
//					defer res.Body.Close()
//					data, err := io.ReadAll(res.Body)
//					if err != nil {
//						return nil, err
//					}
//					return string(data), nil
//				}))
//		}
//		_ = vm.Runtime().Set("fetch", fetch)
//
//		start := time.Now()
//
//		result, err := vm.RunString(ctx, fmt.Sprintf(`fetch("%s")`, server.URL))
//		if err != nil {
//			panic(err)
//		}
//		value, err := js.Unwrap(result)
//		if err != nil {
//			panic(err)
//		}
//
//		fmt.Println(value)
//		fmt.Println(time.Since(start))
//	}
func New[T any](rt *sobek.Runtime, async func() (T, error), then ...func(T, error) (any, error)) *sobek.Promise {
	enqueue := js.EnqueueJob(rt)
	promise, resolve, reject := rt.NewPromise()

	thenFun := func(r T, e error) (any, error) { return r, e }
	if len(then) > 0 {
		thenFun = then[0]
	}

	go func() {
		defer func() {
			if x := recover(); x != nil {
				err := reject(x)
				if err != nil {
					slog.Warn(fmt.Sprintf(`reject failed: %s`, err))
				}
			}
		}()

		result, err := async()
		enqueue(func() error {
			value, err := thenFun(result, err)
			if err != nil {
				return reject(err)
			} else {
				return resolve(value)
			}
		})
	}()

	return promise
}

// Reject with reason
func Reject(rt *sobek.Runtime, reason any) sobek.Value {
	promise, _, rejectFn := rt.NewPromise()
	_ = rejectFn(reason)
	return rt.ToValue(promise)
}
