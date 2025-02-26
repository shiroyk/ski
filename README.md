# ski
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/shiroyk/ski)
[![Go Report Card](https://goreportcard.com/badge/github.com/shiroyk/ski)](https://goreportcard.com/report/github.com/shiroyk/ski)
![GitHub](https://img.shields.io/github/license/shiroyk/ski)<br/>

## Description
**ski** is a collection of Goja modules, provides a set of built-in modules for common tasks like caching, cryptography, encoding, fetch.

## Modules
Partial Node.js compatibility and web standard implementations.
- [buffer](#buffer)
- [encoding](#encoding)
- [fetch](#fetch)
- [stream](#stream)
- [timers](#timers)
- [url](#url)
### buffer
buffer module implements.
- Blob
- File
```js
export default async function () {
  const blob = new Blob(["hello world"], { type: "text/plain" });
  console.log(await blob.text());
}
```
### encoding
encoding module provides base64 decode/encode and TextDecoder/TextEncoder.
- base64
- TextDecoder
- TextEncoder
```js
export default function () {
  const encoder = new TextEncoder();
  const data = encoder.encode("hello");
  return encoding.base64.encode(data);
}
```
### fetch
fetch module provides HTTP client functionality. Web API implementations:
- fetch
- FormData
- Headers
- Request
- Response
- AbortController
- AbortSignal

Other:
- cookieJar
- http
```js
export default async () => {
  const res = await fetch("http://example.com", {
    headers: {
      "X-Custom": "custom value"
    }
  });
  console.log(await response.text());
}
```
### stream
stream module implements.
- ReadableStream
```js
export default async () => {
  const res = new Response("test");
  const reader = await res.body.getReader();
  // ...
}
```
### timers
timers module provides JavaScript timer functions.
```js
export default async () => {
  return await new Promise((resolve) => {
    let count = 0;
    const id = setInterval(() => {
      count++;
    }, 100);

    setTimeout(() => {
      clearInterval(id);
      resolve(count);
    }, 250);
  });
}
```
### url
url module implements [WHATWG URL Standard](https://url.spec.whatwg.org/).
- URL
- URLSearchParams
```js
export default async () => {
  console.log(new URL('http://example.com'));
}
```
### cache
cache module provides for store string or bytes.
```js
import cache from "ski/cache";

export default function () {
  cache.set("hello", "world");
}
```
### crypto
crypto module provides cryptographic functionality including hashing and encryption/decryption.

Supported algorithms:

- aes
- des
- md5
- hmac
- ripemd160
- tripleDes
- sha1
- sha256
- sha384
- sha512
- sha512_224
- sha512_256
```js
import crypto from "ski/crypto";

export default function () {
  return crypto.md5('hello').hex();
}
```

## Usage

```go
package main

import (
	"context"
	"fmt"

	"github.com/shiroyk/ski"
	"github.com/shiroyk/ski/js"

	_ "github.com/shiroyk/ski/modules/timers"
)

func main() {
	module, err := js.CompileModule(`module`, `
        import { default as $ } from "ski/gq";

	export default function () {
	    return $('<div><span>hello</ span></ div>').find('span').text();
	}
	`)
	if err != nil {
		panic(err)
	}

	result, err := ski.RunModule(context.Background(), module, 100)
	if err != nil {
		panic(err)
	}
	fmt.Println(result.Export())
}
```
## License
ski is distributed under the [**MIT license**](https://github.com/shiroyk/ski/blob/master/LICENSE.md).
