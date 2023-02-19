# Cloudcat
**Cloudcat** is a tool for extracting structured data from websites using YAML configuration and the syntax rule is extensible.
## CLI example
```shell
cat << EOF > HackerNews.yaml
source:
  name: HackerNews
  url: https://news.ycombinator.com/best
  timeout: 60s
schema:
  stories:
    type: array
    init:
      - gq: "#hnmain tbody -> slice(2) -> child('tr:not(.spacer,.morespace,:last-child)')"
        js: |
          items = content.length ? content : []
          items.reduce((acc, v, i, arr) => {
            if (i % 2 === 0) {
              acc.push(arr.slice(i, i + 2).join(''));
            }
            return acc;
          }, [])
    properties:
      index:
        type: integer
        rule:
          - gq: .rank
            regex: /[^\d]/
      title: { gq: .titleline>:first-child }
      by: { gq: .hnuser }
      age: { gq: .age }
      comments:
        type: integer
        rule:
          - gq: .subline>:last-child
            regex: /[^\d]/
EOF

cloudcat -m HackerNews.yaml
```
## Documentation
* [Model](#model)
* [Source](#source)
* [Schema](#schema)
  * [Type](#type)
  * [Format](#format)
  * [Init](#init)
  * [Rule](#rule)
  * [Properties](#properties)
  * [Example](#example)
    * [String](#string)
    * [Number](#number)
    * [Integer](#integer)
    * [Boolean](#boolean)
    * [Object](#object)
    * [Array](#array)
    * [Example](#example-1)
* [Parser](#parser)
  * [gq](#gq)
    * [Syntax](#syntax)
    * [Build in functions](#build-in-functions)
      * [Get](#get)
      * [Set](#set)
      * [Text](#text)
      * [Join](#join)
      * [Attr](#attr)
      * [Href](#href)
      * [Html](#html)
      * [Prev](#prev)
      * [Next](#next)
      * [Slice](#slice)
      * [Child](#child)
      * [Parent](#parent)
      * [Parents](#parents)
  * [Js](#js)
    * [Global Modules](#global-modules)
      * [Require](#require)
      * [Console](#console)
    * [Build in Modules](#build-in-modules)
      * [Cache](#cache)
      * [Cookie](#cookie)
      * [HTTP](#http)
      * [Shortener](#shortener)
    * [Environment variables](#environment-variables)
      * [Content](#content)
      * [Cat](#cat)
    * [Example](#example-2)
  * [JSON](#json)
    * [Example](#example-3)
  * [Regex](#regex)
    * [Syntax](#syntax-1)
    * [Example](#example-4)
  * [Xpath](#xpath)
    * [Example](#example-5)
* [Context](#context)
  * [Cancel](#cancel)
  * [Deadline](#deadline)
  * [Done](#done)
  * [Value](#value)
  * [GetValue](#getvalue)
  * [SetValue](#setvalue)
  * [ClearValue](#clearvalue)
  * [Logger](#logger)
  * [BaseURL](#baseurl)
  * [RedirectURL](#redirecturl)
* [License](#license)
* [Todo](#todo)
## Model
The **Model** consists of a **[Source](#source)** and a **[Schema](#schema)**.
## Source
The **source** is the configuration of an extract task.
```yaml
source:
  name: test # name of the model
  url: https://example.com # url of the source
  proxy:  # one or more proxy
    - https://example1.com
    - https://example2.com
  timeout: 60s # the extract task timeout
  header: # the request header
    user-agent: cloudcat
```
## Schema
Schema definitions a data structure.
A schema has the following attributes.
 - type
 - format
 - init
 - rule
 - properties
### Type
The **type** keyword defines the type of property.
Keyword must be one of the six primitive types string.
 - string
 - number
 - integer
 - boolean
 - object
 - array
### Format
The **format** keyword defines the format of property.
Keyword value is same as the **[type](#type)** keyword.
The default formatter converts the string to the specified format type.
If format type is object or array, the string will be deserialized using JSON..
### Init
The **init** keyword defines the initial extract rules.
If property **[type](#type)** isn't **object** or **array** can be omitted.
### Rule
The **rule** keyword defines the property's extract rules.
### Properties
The **properties** keyword defines the property's properties.
If property **[type](#type)** isn't **object** or **array** can be omitted.
### Example
#### String
This schema defines an object with the **string** type properties.
```yaml
title:
  type: string
  rule: { gq: foo }
```
The above configuration can be abbreviated. **Only support the property is string type**.
```yaml
title: { gq: foo }
```
#### Number
This schema defines an object with the **number** type properties.
```yaml
price:
  type: number
  rule: { gq: foo }
```
#### Integer
This schema defines an object with the **integer** type properties.
```yaml
index:
  type: integer
  rule: { gq: foo }
```
#### Boolean
This schema defines an object with the **boolean** type properties.
```yaml
starred:
  type: boolean
  rule: { gq: foo }
```
#### Object
This schema defines an object with the **object** type properties.
```yaml
user:
  type: object
  properties:
    name: { gq: foo }
    star:
      type: integer
      rule: { gq: foo }
```
#### Array
This schema defines an object with the **array** type properties.
```yaml
book:
  type: array
  properties:
    title: { gq: foo }
    price:
      type: number
      rule: { gq: foo }
```
#### Example
HackerNews
```yaml
stories:
  type: array
  init:
    - gq: "#hnmain tbody -> slice(2) -> child('tr:not(.spacer,.morespace,:last-child)')"
      js: |
        items = content.length ? content : []
        items.reduce((acc, v, i, arr) => {
          if (i % 2 === 0) {
            acc.push(arr.slice(i, i + 2).join(''));
          }
          return acc;
        }, [])
  properties:
    index:
      type: integer
      rule:
        - gq: .rank
          regex: /[^\d]/
    title: { gq: .titleline>:first-child }
    by: { gq: .hnuser }
    age: { gq: .age }
    comments:
      type: integer
      rule:
        - gq: .subline>:last-child
          regex: /[^\d]/
```
## Parser
**Parser** is used to parse the rules.
The following parsers are built in:
 - [gq](#gq)
 - [js](#js)
 - [json](#json)
 - [regex](#regex)
 - [xpath](#xpath)
### gq
**gq** depends on the [Goquery](https://github.com/PuerkitoBio/goquery) library.
#### Syntax
**gq** syntax consists of selectors and functions and is separated by **->**.
```yaml
body: { gq: '#hnmain tbody -> slice(2) -> join('')' }
```
#### Build in functions
##### Get
Get returns the value associated with this [context](#context) for key
##### Set
Set value associated with key to the [context](#context). 
The first argument is the key, and the second argument is value.
If the value is present will store the previous execute result.
##### Text
Text gets the combined text contents of each element in the set of matched
elements, including their descendants.
##### Join
Join the text with the separator, if not present separator uses the default separator ", ".
##### Attr
Attr gets the specified attribute's value for the first element in the Selection.
The first argument is the name of the attribute, the second is the default value.
##### Href
Href gets the href attribute's value, if URL is not absolute get the base URL from context
and return the absolute URL.
##### Html
Html the first argument is outer.
If true returns the outer HTML rendering of the first item in
the selection - that is, the HTML including the first element's
tag and attributes, or gets the HTML contents of the first element
in the set of matched elements. It includes text and comment nodes.
##### Prev
Prev gets the immediately preceding sibling of each element in the Selection.
If present selector gets all preceding siblings of each element up to but not
including the element matched by the selector.
##### Next
Next gets the immediately following sibling of each element in the Selection.
If present selector gets all following siblings of each element up to but not
including the element matched by the selector.
##### Slice
Slice reduces the set of matched elements to a subset specified by a range
of indices. The start index is 0-based and indicates the index of the first
element to select. The end index is 0-based and indicates the index at which
the elements stop being selected (the end index is not selected).

If the end index is not specified reduces the set of matched elements to the one at the
specified start index.

The indices may be negative, in which case they represent an offset from the
end of the selection.
##### Child
Child gets the child elements of each element in the Selection.
If present the selector will return filtered by the specified selector.
##### Parent
Parent gets the parent of each element in the Selection.
If present the selector will return filtered by a selector.
##### Parents
Parents get the ancestors of each element in the current Selection.
If present the selector will return filtered by a selector.
### Js
**js** depends on the [goja](https://github.com/dop251/goja) library.
#### Global Modules
 - [require](#require)
 - [console](#console)
##### Require
js module **require** implements the CommonJS modules require.
**require** can load under node_modules directory module files and remote modules.
```js
const localLodash = require("lodash");
console.log(localLodash.VERSION);
const remoteLodash = require("https://cdn.jsdelivr.net/npm/lodash@4.17.21/lodash.min.js");
console.log(remoteLodash.VERSION);
```
##### Console
console is a tool which is mainly used to log information.
```js
console.log("%s test", 1);
```
#### Build in Modules
These modules can be imported by cloudcat/*
 - [cache](#cache)
 - [cookie](#cookie)
 - [http](#http)
 - [shortener](#shortener)
##### Cache
**cache** is used to store data.
```js
const cache = require('cloudcat/cache');
cache.set("cache1", "1");
cache.del("cache1");
cache.get("cache2");
cache.setBytes("cache3", new Uint8Array([50]).buffer);
cache.getBytes("cache3");
```
##### Cookie
**cookie** manages storage and use of cookies in HTTP requests.
```js
const cookie = require('cloudcat/cookie');
cookie.set("https://github.com", "max-age=3600;");
cookie.get("https://github.com");
cookie.del("https://github.com");
```
##### HTTP
**http** fetch resources across the network and return a response object.
```js
const http = require('cloudcat/http');
const url = "http://localhost"
http.get(url);
http.get(url, { "User-Agent": "cloudcat" });
http.head(url);
http.post(url, {'dark': 'o'}); // application/json
http.post(url, new URLSearchParams({'key': 'foo'})); // application/x-www-form-url
http.post(url, new FormData({"name": "foo", "file": new Uint8Array([226, 153, 130, 239, 184, 142]).buffer})); // multipart/form-data
http.request('PUT', url, {"name": "bar"}, {"Authorization": "token 123456"})
```
##### Shortener
**shortener** is URL shortener to reduce a long http request.
```js
const shortener = require('cloudcat/shortener');
console.log(shortener.set(`POST http://localhost HTTP/2.0
Pragma: no-cache
Content-Type: application/octet-stream
Connection: close

{{ get "image" }}`));
```
#### Environment variables
##### Content
**content** is the result of the previous parser execution.
##### Cat
**cat** is the [context](#context) of the task, it has the following methods:
 - log
 - getVar
 - setVar
 - clearVar
 - getString
 - getStrings
 - getElement
 - getElements
#### Example
```yaml
body:
  type: string
  rule:
    - gq: "#hnmain tbody -> slice(2) -> child('tr:not(.spacer,.morespace,:last-child)')"
      js: |
        items = content.length ? content : []
        items.reduce((acc, v, i, arr) => {
          if (i % 2 === 0) {
            acc.push(arr.slice(i, i + 2).join(''));
          }
          return acc;
        }, [])
```
### JSON
**json** depends on the [ojg](https://github.com/ohler55/ojg) library.
#### Example
```yaml
author: { json: $.store.book[0].author }
```
### Regex
**regex** depends on the [regexp2](https://github.com/dlclark/regexp2) library.
#### Syntax
**regex** syntax consists of find and replace and optional flags, separated by **/**.
The following syntax is to find a to z replacing three characters with 1 starting from the third and ignore case.
```
/[a-z]/1/i{3,3}
```
The available flags are:
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
#### Example
```yaml
author: 
  rule: 
    - json: $.store.book[0].author
      regex: /[a-z]/1/i{3,3}
```
### Xpath
**xpath** depends on the [htmlquery](https://github.com/antchfx/htmlquery) and [xpath](https://github.com/antchfx/xpath) library.
#### Example
```yaml
title: { xpath: '//div[@id="main"]/div[contains(@class, "row")]/text()' }
```
## Context
Every extracting task has a unique **context**.
**Context** contains methods such as timeout control and temporary variable access.
### Cancel
Cancel this context releases resources associated with it.
### Deadline
Deadline returns the time when work done on behalf of this context should be canceled.
### Done
Done returns a channel that's closed when work done on behalf of this context should be canceled.
### Value
Value returns the value associated with this context for key, or nil.
### GetValue
GetValue same as Value.
### SetValue
SetValue value associated with key.
### ClearValue
ClearValue clean all values.
### Logger
Logger returns the logger.
### BaseURL
BaseURL returns the base URL string.
### RedirectURL
RedirectURL returns the redirect URL string
## License
cloudcat is distributed under the [AGPL-3.0 license](https://github.com/shiroyk/cloudcat/blob/master/LICENSE.md).
## Todo
1. [ ] REST API
2. [ ] Documentation