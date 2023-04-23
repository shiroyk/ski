# Cloudcat Ctl
**ctl** is a command line client for cloudcat.
## Usage
run the **Model**
```shell
cat << EOF | cloudcat run -m -
source:
  name: HackerNews
  http: https://news.ycombinator.com/best
  timeout: 60s
schema:
  type: array
  init:
    - gq: "#hnmain tbody -> slice(2) -> child('tr:not(.spacer,.morespace,:last-child)')"
      js: |
        content?.reduce((acc, v, i, arr) => {
          if (i % 2 === 0) {
            acc.push(arr.slice(i, i + 2).join(''));
          }
          return acc;
        }, []);
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
```
run the **JavaScript**
```shell
cat << EOF | cloudcat run -s -
const http = require('cloudcat/http');
let res = http.get('https://news.ycombinator.com/best');
let stories = cat.getElements('gq', "#hnmain tbody -> slice(2) -> child('tr:not(.spacer,.morespace,:last-child)')", res.string());
stories?.reduce((acc, v, i, arr) => {
    if (i % 2 === 0) {
        let item = arr.slice(i, i + 2).join('');
        let index = cat.getString('gq', '.rank', item);
        let title = cat.getString('gq', '.titleline>:first-child', item);
        let by = cat.getString('gq', '.hnuser', item);
        let age = cat.getString('gq', '.age', item);
        let comments = cat.getString('gq', '.subline>:last-child', item);
        acc.push({
            index: parseInt(index?.replace(/[^\d]+/g, ''), 10),
            title: title,
            by: by,
            age: age,
            comments: parseInt(comments?.replace(/[^\d]+/g, ''), 10)
        });
    }
    return acc;
}, []);
EOF
```
## API
If no secret is specified a random secret will be output.
```shell
cloudcat -s cloudcat
# Secret: cloudcat
# Service start http://localhost:8080
export SECRET=cloudcat
```
### /ping
Test if the API service is working.
```shell
curl --request GET \
  --url http://localhost:8080/ping \
  --header "Authorization: Bearer $SECRET"
```
### /v1/run
```shell
curl --request POST \
  --url http://localhost:8080/v1/run \
  --header "Authorization: Bearer $SECRET" \
  --header 'Content-Type: text/yaml' \
  --data 'source:
  name: HackerNews
  http: https://news.ycombinator.com/best
  timeout: 60s
schema:
  type: array
  init:
    - gq: "#hnmain tbody -> slice(2) -> child('\''tr:not(.spacer,.morespace,:last-child)'\'')"
      js: |
        items = content.length ? content : []
        items.reduce((acc, v, i, arr) => {
          if (i % 2 === 0) {
            acc.push(arr.slice(i, i + 2).join('\'''\''));
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
          regex: /[^\d]/'
```
Set application/javascript request header to run javascript.
```shell
curl --request POST \
  --url http://localhost:8080/v1/run \
  --header "Authorization: Bearer $SECRET" \
  --header 'Content-Type: application/javascript' \
  --data 'const http = require('\''cloudcat/http'\'');
let res = http.get('\''https://news.ycombinator.com/best'\'');
let stories = cat.getElements('\''gq'\'', res.string(), "#hnmain tbody -> slice(2) -> child('\''tr:not(.spacer,.morespace,:last-child)'\'')");
stories?.reduce((acc, v, i, arr) => {
    if (i % 2 === 0) {
        let item = arr.slice(i, i + 2).join('\'''\'');
        let index = cat.getString('\''gq'\'', item, '\''.rank'\'');
        let title = cat.getString('\''gq'\'', item, '\''.titleline>:first-child'\'');
        let by = cat.getString('\''gq'\'', item, '\''.hnuser'\'');
        let age = cat.getString('\''gq'\'', item, '\''.age'\'');
        let comments = cat.getString('\''gq'\'', item, '\''.subline>:last-child'\'');
        acc.push({
            index: parseInt(index?.replace(/[^\d]+/g, '\'''\''), 10),
            title: title,
            by: by,
            age: age,
            comments: parseInt(comments?.replace(/[^\d]+/g, '\'''\''), 10)
        });
    }
    return acc;
}, []);'
```
### /v1/debug
debug the parsing.