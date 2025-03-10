package main

import (
	"fmt"
	"io"
	"net/http"
	urlpkg "net/url"
	"testing/fstest"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
)

var (
	sourceFS = make(fstest.MapFS)
)

func source(path, data string) {
	sourceFS[path] = &fstest.MapFile{Data: []byte(data)}
	fmt.Println("source file:", path)
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
