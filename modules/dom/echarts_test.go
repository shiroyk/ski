package dom

import (
	"testing"

	"github.com/shiroyk/ski/js/modulestest"
	"github.com/stretchr/testify/require"
)

func TestEchartsSSR(t *testing.T) {
	t.Parallel()
	vm := modulestest.New(t)
	_ = vm.Runtime().Set("global", map[string]any{
		"__DEV__": true,
	})

	source := `
import echarts from "https://unpkg.com/echarts@5/dist/echarts.js";

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

let svg = chart.renderToSVGString();
chart.dispose();
assert.regexp(svg, '<svg width="400" height="300"');
`
	_, err := vm.RunModule(t.Context(), source)
	require.NoError(t, err)
}
