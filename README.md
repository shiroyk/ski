# Cloudcat
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/shiroyk/cloudcat)
[![Go Report Card](https://goreportcard.com/badge/github.com/shiroyk/cloudcat)](https://goreportcard.com/report/github.com/shiroyk/cloudcat)
![GitHub](https://img.shields.io/github/license/shiroyk/cloudcat)<br/>
**Cloudcat** is a tool for extracting structured data from websites using YAML configuration and the syntax rule is extensible.<br/>
⚠️**This project is still in development**.
## CLI example
```shell
cat << EOF > HackerNews.yaml
source:
  name: HackerNews
  http: https://news.ycombinator.com/best
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

cloudcat run -m HackerNews.yaml
```
## Documentation
See [Wiki](https://github.com/shiroyk/cloudcat/wiki)
## License
cloudcat is distributed under the [AGPL-3.0 license](https://github.com/shiroyk/cloudcat/blob/master/LICENSE.md).
## Todo
1. [ ] REST API
2. [ ] Documentation