# ski
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/shiroyk/ski)
[![Go Report Card](https://goreportcard.com/badge/github.com/shiroyk/ski)](https://goreportcard.com/report/github.com/shiroyk/ski)
![GitHub](https://img.shields.io/github/license/shiroyk/ski)<br/>
**ski** is a tool written in Golang for extracting structured data.

## Usage
```go
package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/shiroyk/ski"
)

func main() {
	const source = `
$gq.elements: table[aria-labelledby="folders-and-files"] .react-directory-row-name-cell-large-screen .react-directory-filename-cell a
$each:
  $map:
    title:
      $gq: -> text
    href:
      $gq: -> href
    
`
	executor, err := ski.Compile(source)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	resp, err := http.Get("https://github.com/shiroyk/ski")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	result, err := executor.Exec(ctx, string(bytes))
	if err != nil {
		panic(err)
	}

	err = json.NewEncoder(os.Stdout).Encode(result)
	if err != nil {
		panic(err)
	}
}
```
## License
ski is distributed under the [**MIT license**](https://github.com/shiroyk/ski/blob/master/LICENSE.md).