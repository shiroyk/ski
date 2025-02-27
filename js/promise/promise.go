package promise

import (
	"errors"

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
				enqueue(func() error { return reject(x) })
			}
		}()

		result, err := async()
		enqueue(func() error {
			if x := recover(); x != nil {
				enqueue(func() error { return reject(x) })
			}
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

// Resolve with value
func Resolve(rt *sobek.Runtime, value any) sobek.Value {
	promise, resolve, _ := rt.NewPromise()
	_ = resolve(value)
	return rt.ToValue(promise)
}

// Result returns the promise result, if it not promise return origin value.
func Result(value sobek.Value) (any, error) {
	if value == nil {
		return nil, nil
	}
	v := value.Export()
	promise, ok := v.(*sobek.Promise)
	if !ok {
		return v, nil
	}
	switch promise.State() {
	case sobek.PromiseStateRejected:
		return nil, errors.New(promise.Result().String())
	case sobek.PromiseStateFulfilled:
		return promise.Result().Export(), nil
	default:
		return nil, errors.New("unexpected promise pending state")
	}
}
