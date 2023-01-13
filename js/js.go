package js

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	"github.com/labstack/gommon/log"
	"github.com/shiroyk/cloudcat/parser"
)

// CreateVMWithContext instance JS runtime with parser.Context
func CreateVMWithContext(ctx *parser.Context, content any) *goja.Runtime {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.UncapFieldNameMapper())

	req := new(require.Registry)
	req.Enable(vm)
	req.RegisterNativeModule("console", console.RequireWithPrinter(console.PrinterFunc(func(s string) {
		log.Info(s)
	})))
	console.Enable(vm)

	_ = vm.Set("FormData", NewFormData)
	_ = vm.Set("URLSearchParams", NewURLSearchParams)
	_ = vm.Set("go", jsContext{
		ctx:         ctx,
		BaseUrl:     ctx.BaseUrl(),
		RedirectUrl: ctx.RedirectUrl(),
		Content:     content,
		Http:        jsHttp{fetch: ctx.Fetcher()},
		Cache:       jsCache{cache: ctx.Cache()},
		Cookie:      jsCookie{cookie: ctx.Cookie()},
		Shortener:   jsShortener{shortener: ctx.Shortener()},
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
