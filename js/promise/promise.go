package promise

import (
	"errors"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
)

// Callback is a function that receives a function to resolve/reject the promise.
//
// The function return:
//   - result (any): The value to resolve the promise with
//   - error: If non-nil, the promise will be rejected with this error
type Callback func(func() (any, error))

// New creates a sobek.Promise that wraps an asynchronous operation.
// The async function runs in a separate goroutine and can safely interact with JavaScript values
// through the provided callback. The callback ensures all JavaScript operations happen on the main goroutine.
//
// The callback return:
//   - result (any): The value to resolve the promise with
//   - error: If non-nil, the promise will be rejected with this error
//
// If a panic occurs in either the async function or callback, the promise will be rejected with the panic value.
//
// Example usage - implementing an async fetch function:
//
//	func main() {
//		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//			w.WriteHeader(http.StatusOK)
//			_, _ = w.Write([]byte(`{"foo":"bar"}`))
//		}))
//		defer server.Close()
//
//		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
//		defer cancel()
//
//		fetch := func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
//			return promise.New(rt, func(callback promise.Callback) {
//				res, err := http.Get(call.Argument(0).String())
//				callback(func() (any, error) {
//					if err != nil {
//						return nil, err
//					}
//					defer res.Body.Close()
//					data, err := io.ReadAll(res.Body)
//					if err != nil {
//						return nil, err
//					}
//					return string(data), nil
//				})
//			})
//		}
//
//		var (
//			value sobek.Value
//			err   error
//		)
//		err = js.Run(ctx, func(rt *sobek.Runtime) error {
//			_ = rt.Set("fetch", fetch)
//			value, err = rt.RunString(fmt.Sprintf(`fetch("%s")`, server.URL))
//			return err
//		})
//		if err != nil {
//			panic(err)
//		}
//		fmt.Println(value.Export().(*sobek.Promise).Result().Export())
//	}
func New(rt *sobek.Runtime, async func(callback Callback)) sobek.Value {
	enqueue := js.EnqueueJob(rt)
	promise, resolve, reject := rt.NewPromise()

	go func() {
		defer func() {
			if x := recover(); x != nil {
				enqueue(func() error { return reject(x) })
			}
		}()

		async(func(callback func() (any, error)) {
			enqueue(func() (err error) {
				defer func() {
					if x := recover(); x != nil {
						err = reject(x)
					}
				}()
				result, err := callback()
				if err != nil {
					return reject(err)
				} else {
					return resolve(result)
				}
			})
		})
	}()

	return rt.ToValue(promise)
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
