# Cloudcat

## Usage
run the **Model**
```shell
cat << EOF | ./cloudcat -d -m -
source:
  name: HackerNews
  http: https://news.ycombinator.com/best
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
        }, [])
  properties:
    index: !integer
      - gq: .rank
        regex: /[^\d]/
    title: { gq: .titleline>:first-child }
    by: { gq: .hnuser }
    age: { gq: .age }
    comments: !integer
      - gq: .subline>:last-child
        regex: /[^\d]/
EOF
```
run the **Script**
```shell
cat << EOF | ./cloudcat -d -s -
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