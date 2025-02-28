package promise

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromise(t *testing.T) {
	t.Parallel()
	vm := js.NewVM()

	t.Run("new", func(t *testing.T) {
		rt := vm.Runtime()

		t.Run("resolve", func(t *testing.T) {
			var v sobek.Value
			err := vm.Run(t.Context(), func() error {
				v = New(rt, func(callback Callback) {
					callback(func() (any, error) {
						return "resolve", nil
					})
				})
				return nil
			})
			require.NoError(t, err)
			result, err := Result(v)
			require.NoError(t, err)
			assert.Equal(t, "resolve", result)
		})

		t.Run("reject", func(t *testing.T) {
			var v sobek.Value
			err := vm.Run(t.Context(), func() error {
				v = New(rt, func(callback Callback) {
					callback(func() (any, error) {
						return nil, errors.New("reject")
					})
				})
				return nil
			})
			require.NoError(t, err)
			_, err = Result(v)
			assert.ErrorContains(t, err, "reject")
		})

		t.Run("panic on async", func(t *testing.T) {
			assert.NotPanics(t, func() {
				var v sobek.Value
				err := vm.Run(t.Context(), func() error {
					v = New(rt, func(callback Callback) {
						panic("reject")
					})
					return nil
				})
				require.NoError(t, err)
				_, err = Result(v)
				assert.ErrorContains(t, err, "reject")
			})
		})

		t.Run("panic on callback", func(t *testing.T) {
			assert.NotPanics(t, func() {
				var v sobek.Value
				err := vm.Run(t.Context(), func() error {
					v = New(rt, func(callback Callback) {
						callback(func() (any, error) {
							panic("reject")
						})
					})
					return nil
				})
				require.NoError(t, err)
				_, err = Result(v)
				assert.ErrorContains(t, err, "reject")
			})
		})

	})

	t.Run("example", func(t *testing.T) {
		result := `{"foo":"bar"}`
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(result))
		}))
		defer server.Close()

		fetch := func(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
			return New(rt, func(callback Callback) {
				res, err := http.Get(call.Argument(0).String())
				callback(func() (any, error) {
					if err != nil {
						return nil, err
					}
					defer res.Body.Close()
					data, err := io.ReadAll(res.Body)
					if err != nil {
						return nil, err
					}
					return string(data), nil
				})
			})
		}

		var (
			value sobek.Value
			err   error
		)
		err = vm.Run(t.Context(), func() error {
			_ = vm.Runtime().Set("fetch", fetch)
			value, err = vm.Runtime().RunString(fmt.Sprintf(`fetch("%s")`, server.URL))
			return err
		})
		if err != nil {
			panic(err)
		}
		v, err := Result(value)
		require.NoError(t, err)
		assert.Equal(t, result, v)
	})
}
