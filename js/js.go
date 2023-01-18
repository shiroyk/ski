package js

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	"github.com/shiroyk/cloudcat/cache"
	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/fetcher"
	"github.com/shiroyk/cloudcat/parser"
	"golang.org/x/exp/slog"
)

// CreateVMWithContext instance JS runtime with parser.Context
func CreateVMWithContext(ctx *parser.Context, content any) *goja.Runtime {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.UncapFieldNameMapper())

	req := new(require.Registry)
	req.Enable(vm)
	req.RegisterNativeModule("console", console.RequireWithPrinter(console.PrinterFunc(func(s string) {
		slog.Info(s)
	})))
	console.Enable(vm)

	_ = vm.Set("FormData", NewFormData)
	_ = vm.Set("URLSearchParams", NewURLSearchParams)
	_ = vm.Set("go", jsContext{
		ctx:         ctx,
		BaseUrl:     ctx.BaseUrl(),
		RedirectUrl: ctx.RedirectUrl(),
		Content:     content,
		Http:        jsHttp{fetch: di.MustResolve[*fetcher.Fetcher]()},
		Cache:       jsCache{cache: di.MustResolveNamed[cache.Cache]("cache")},
		Cookie:      jsCookie{cookie: di.MustResolveNamed[cache.Cookie]("cookie")},
		Shortener:   jsShortener{shortener: di.MustResolveNamed[cache.Shortener]("shortener")},
	})

	go func() {
		// Wait for the context to be done
		<-ctx.Done()
		// Interrupt the JS runtime
		vm.Interrupt(ctx.Err())
	}()

	return vm
}

// UnWrapValue unwrap the goja.Value to the raw value
func UnWrapValue(value goja.Value) any {
	switch value := value.Export().(type) {
	default:
		return value
	case goja.ArrayBuffer:
		return value.Bytes()
	case *goja.Promise:
		switch value.State() {
		case goja.PromiseStateRejected:
			panic(value.Result().String())
		case goja.PromiseStateFulfilled:
			return value.Result().Export()
		default:
			panic(fmt.Errorf("unexpected promise state: %v", value.State()))
		}
	}
}
