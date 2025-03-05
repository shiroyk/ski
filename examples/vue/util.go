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
	"time"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
)

var (
	sourceFS = make(fstest.MapFS)
	// compiler compile vue sfc file
	compiler = sync.OnceValue(func() func(bool, string, string) ([]byte, error) {
		vm := js.NewVM()
		module, err := js.CompileModule("compiler", `
import {compileScript, parse} from "https://esm.sh/vue@3/compiler-sfc";

export default (ssr, name, code) => {
  const { descriptor, errors } = parse(code);
  if (errors && errors.length) {
    throw new Error('parse sfc file error：' + errors);
  }

  if (descriptor.script || descriptor.scriptSetup) {
    const { content } = compileScript(descriptor, {
      id: "app",
      inlineTemplate: true,
      genDefaultAs: "__sfc__",
      templateOptions: { ssr: false, ssrCssVars: descriptor.cssVars, },
    });
    return content + "\n__sfc__.__file = '"+name+"';\nexport default __sfc__;";
  }
};
`)
		return func(ssr bool, name, data string) ([]byte, error) {
			if err != nil {
				return nil, fmt.Errorf("init compiler falied: %w", err)
			}
			value, err := vm.RunModule(context.Background(), module, ssr, name, data)
			if err != nil {
				return nil, fmt.Errorf("compile falied: %w", err)
			}
			return []byte(value.String()), nil
		}
	})
)

func source(path, data string) {
	if filepath.Ext(path) == ".vue" {
		// compile vue sfc file
		client, err := compiler()(false, filepath.Base(path), data)
		if err != nil {
			panic("compile vue sfc failed: " + err.Error())
		}
		sourceFS[path] = &fstest.MapFile{Data: client}

		ssr, err := compiler()(true, filepath.Base(path), data)
		if err != nil {
			panic("compile vue sfc failed: " + err.Error())
		}
		sourceFS[path+"?ssr"] = &fstest.MapFile{Data: ssr}

		fmt.Println("compile vue:", path)
	} else {
		sourceFS[path] = &fstest.MapFile{Data: []byte(data)}
		fmt.Println("source file:", path)
	}
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
		return sourceFS.ReadFile(specifier.Path)
	}
	return nil, fmt.Errorf("scheme not supported %s", specifier.Scheme)
}

func openFile(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	name := call.Argument(0).String()
	data, err := sourceFS.ReadFile(name)
	if err != nil {
		js.Throw(rt, err)
	}
	return rt.ToValue(string(data))
}

func now(_ sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(time.Now().UnixNano() / 1000)
}
