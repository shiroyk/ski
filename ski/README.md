# ski

## Install
```shell
go install github.com/shiroyk/ski/ski
```

## Run
```shell
cat << EOF | ski -
import http from "ski/http";
import { default as $, selector } from "ski/gq";

export default (data) => {
    const res = http.get('https://news.ycombinator.com/best');
    
    const index = selector('.rank');
    const title = selector('.titleline>:first-child');
    const by = selector('.hnuser');
    const age = selector('.age');
    const comments = selector('.subline>:last-child');
    
    const stories = $("#hnmain tbody", res.text()).eq(2)
      .find('tr:not(.spacer,.morespace,:last-child)').toArray();

    return stories?.reduce((acc, v, i, arr) => {
        if (i % 2 === 0) {
            let item = arr.slice(i, i + 2);
            acc.push({
                index: parseInt($(index, item).text().replace(/[^\d]+/g, ''), 10),
                title: $(title, item).text(),
                by: $(by, item).text(),
                age: $(age, item).text(),
                comments: parseInt($(comments, item).text().replace(/[^\d]+/g, ''), 10)
            });
        }
        return acc;
    }, []);
}
EOF
```