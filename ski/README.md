# ski

## Install
```shell
go install github.com/shiroyk/ski/ski
```

## Example
Render ECharts svg
```shell
cat << EOF | ski -
import echarts from "https://unpkg.com/echarts@5"

globalThis.global = {  __DEV__: true };

let chart = echarts.init(null, null, {
  renderer: 'svg',
  ssr: true,
  width: 400,
  height: 300
});

chart.setOption({
  title: {
    text: 'ECharts entry example'
  },
  backgroundColor: 'white',
  tooltip: {},
  legend: {
    data:['Sales']
  },
  xAxis: {
    data: ["shirt","cardign","chiffon shirt","pants","heels","socks"]
  },
  yAxis: {},
  series: [{
    name: 'Sales',
    type: 'bar',
    data: [5, 20, 36, 10, 10, 20]
  }]
});

export default () => {
  const svg = chart.renderToSVGString();
  chart.dispose();
  return svg;
};
EOF
```