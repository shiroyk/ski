package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	urlpkg "net/url"
	"path/filepath"
	"sync"
	"testing/fstest"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
)

var (
	sourceFS = make(fstest.MapFS)
	// compiler compile jsx file
	compiler = sync.OnceValue(func() func(string) ([]byte, error) {
		vm := js.NewVM()
		module, err := js.CompileModule("compiler", `
import { transform } from "https://esm.sh/@babel/standalone@7";
export default (code) => transform(code, {presets: ["react"]}).code;
`)
		return func(data string) ([]byte, error) {
			if err != nil {
				return nil, fmt.Errorf("init compiler falied: %w", err)
			}
			value, err := vm.RunModule(context.Background(), module, data)
			if err != nil {
				return nil, fmt.Errorf("compile falied: %w", err)
			}
			return []byte(value.String()), nil
		}
	})
)

func source(path, data string) {
	sourceFS[path] = &fstest.MapFile{Data: []byte(data)}
}

func loadFile(path string) ([]byte, error) {
	data, err := sourceFS.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// compile jsx file
	if filepath.Ext(path) == ".jsx" {
		return compiler()(string(data))
	}
	return data, nil
}

func fileLoader(specifier *urlpkg.URL, _ string) ([]byte, error) {
	switch specifier.Scheme {
	case "http", "https":
		res, err := http.Get(specifier.String())
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		return io.ReadAll(res.Body)
	case "file":
		return loadFile(specifier.Path)
	}
	return nil, fmt.Errorf("scheme not supported %s", specifier.Scheme)
}

func openFile(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	name := call.Argument(0).String()
	data, err := loadFile(name)
	if err != nil {
		js.Throw(rt, err)
	}
	return rt.ToValue(string(data))
}
