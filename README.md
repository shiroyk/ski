# Cloudcat
<pre style="text-align: center">
       .__                   .___             __   
  ____ |  |   ____  __ __  __| _/____ _____ _/  |_ 
_/ ___\|  |  /  _ \|  |  \/ __ |/ ___\\__  \\   __\
\  \___|  |_(  <_> )  |  / /_/ \  \___ / __ \|  |  
 \___  >____/\____/|____/\____ |\___  >____  /__|  
     \/                       \/    \/     \/   
</pre>

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
## License
cloudcat is distributed under the [AGPL-3.0 license](https://github.com/shiroyk/cloudcat/blob/master/LICENSE.md).
## Todo
1. [ ] REST API
2. [ ] Documentation