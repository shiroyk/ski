# ski
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/shiroyk/ski)
[![Go Report Card](https://goreportcard.com/badge/github.com/shiroyk/ski)](https://goreportcard.com/report/github.com/shiroyk/ski)
![GitHub](https://img.shields.io/github/license/shiroyk/ski)<br/>
**ski** is a tool written in Golang for extracting structured data.<br/>


## Description
ski use YAML to define data-extracting Executors, which are executed sequentially like a pipeline. <br/>
Here's a simple example to extract the title and author of selected books from HTML document.
```yaml
$gq.elements: .books .select
$each:
  $map:
    title:
      $gq: .title
    author:
      $gq: .author
```
output：
```json
[{"title":"Book 1","author":"Author 1"},{"title":"Book 2","author":"Author 2"}]
```

## Executors
### Build in
#### fetch
`$fetch` fetches the resource from the network, default method is GET.
```yaml
$fetch: https://example.com
```
#### kind
`$kind` converts the argument the specified type.
```yaml
$raw: 123
$kind: int
```
#### list.of
`$list.of` returns a list of Executor result.
```yaml
$list.of:
  - 123
  - 456
```
#### str.join
`$str.join` joins strings with specified separator.
```yaml
$list.of:
  - 123
  - 456
$str.join: ~
```
#### str.split
`$str.split` splits string with specified separator.
```yaml
$raw: 123~456
$str.split: ~
```
#### map
`$map` returns a map of Executor result. [k1, v1, k2, v2, ...]
```yaml
$map:
  - 123
  - 456
```
#### each
`$each` loop the slice arg and execute the Executor.
```yaml
$list.of:
  - 123
  - 456
$each:
  $kind: int
```
#### or
`$or` executes a slice of Executor. return result if the Executor result is not nil.
```yaml
$or:
  - $raw:
  - 456
```
### Control flow
filter the string contains "2" and convert to int, output: [123, 234]
```yaml
$list.of:
  - 123
  - 234
  - 345
$each:
  $pipe:
    $if.contains: 2
    $kind: int
```
filter the string match "bar", output: {"bar": "some value"}
```yaml
$list.of:
  - foo
  - bar
  - baz
$map:
  $if.contains: bar
  $raw: some value
```
### Expression
- [gq](#gq): similar to jQuery expressions.
- [jq](#jq): JSONPath expressions.
- [js](#js): JavaScript expressions.
- [regex](#regex): regular expressions.
- [xpath](#xpath): XPath expressions.
#### gq
**gq** syntax consists of selectors and functions and is separated by **->**.<br/>
`$gq` returns the match element text of the selector. return the first if node length is 1.
```yaml
$gq: .books .title -> text
```
`$gq.element` returns the first element of the selector.
```yaml
$gq.element: .books .select
```
`$gq.elements` returns all elements of the selector.
```yaml
$gq.elements: .books
```
#### jq
`$jq` returns the value of the JSONPath expression.
```yaml
$jq: $.books[0].author
```
#### js
`$js` returns the value of the JavaScript expression.
```yaml
$js: export default (ctx) => ctx.get('content')
```
#### regex
available flags:
- i Ignore case
- m Multiple line
- n Explicit capture
- c Compiled
- s Single line
- x Ignore pattern whitespace
- r Right to left
- d Debug
- e ECMAScript
- u Unicode

`$regex.replace` `/expr/replace/flags{start,count}` replaces the pattern of the string.
```yaml
$regex.replace: /[^\d]/
```
`$regex.match` `/expr/flags{start,count}` returns the match of the pattern of the string.
```yaml
$regex.match: /\\//1
```
`$regex.assert` `/expr/message/flags` asserts the pattern of the string.
```yaml
$regex.assert: /\d+/number not found/
```
#### xpath
`$xpath` returns the match element text of the XPath expression. return the first if node length is 1.
```yaml
$xpath: div p
```
`$xpath.element` returns the first element of the XPath expression.
```yaml
$xpath.element: div p
```
`$xpath.elements` returns all elements of the XPath expression.
```yaml
$xpath.elements: div p
```

## Usage

```go
package main

import (
	"context"
	"fmt"

	"github.com/shiroyk/ski"
)

const content = `...`

const source = ``

func main() {
	executor, err := ski.Compile(source)
	if err != nil {
		panic(err)
	}

	result, err := executor.Exec(context.Background(), content)
	if err != nil {
		panic(err)
	}
	fmt.Println(result)
}
```
## License
ski is distributed under the [**MIT license**](https://github.com/shiroyk/ski/blob/master/LICENSE.md).