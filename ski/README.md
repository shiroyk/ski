# ski

## Install
```shell
go install github.com/shiroyk/ski/ski
```

## Run executor
```shell
cat << 'EOF' | ski -m -
$fetch: https://news.ycombinator.com/best
$xpath.element: //*[@id="hnmain"]/tbody/tr[3]/td/table
$gq.elements: tr -> chunk(3) -> slice(0, 30)
$each:
  $map:
    index:
      $gq: .rank
      $regex.replace: /[^\d]/
      $kind: int
    title:
      $gq: .titleline>:first-child
    by:
      $gq: .hnuser
    age:
      $gq: .age
    comments:
      $gq: .subline>:last-child
      $regex.replace: /[^\d]/
      $kind: int
EOF
```

## Run script
```shell
cat << EOF | ski -s -
import http from "ski/http";
import gq from "executor/gq";

export default () => {
    let res = http.get('https://news.ycombinator.com/best');
    
    const index = gq('.rank');
    const title = gq('.titleline>:first-child');
    const by = gq('.hnuser');
    const age = gq('.age');
    const comments = gq('.subline>:last-child');
    
    const stories = gq.elements("#hnmain tbody -> slice(2) -> child('tr:not(.spacer,.morespace,:last-child)')").exec(res.text());
    return stories?.reduce((acc, v, i, arr) => {
        if (i % 2 === 0) {
            let item = arr.slice(i, i + 2);
            acc.push({
                index: parseInt(index.exec(item)?.replace(/[^\d]+/g, ''), 10),
                title: title.exec(item),
                by: by.exec(item),
                age: age.exec(item),
                comments: parseInt(comments.exec(item)?.replace(/[^\d]+/g, ''), 10)
            });
        }
        return acc;
    }, []);
}
EOF
```