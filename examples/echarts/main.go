package main

import (
	"context"
	"errors"
	"os/signal"
	"syscall"

	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/modules"
	_ "github.com/shiroyk/ski/modules/http"
	_ "github.com/shiroyk/ski/modules/timers"
)

func init() {
	modules.Register("open", modules.ModuleFunc(openFile))
	js.Loader().SetFileLoader(fileLoader)

	// alias module from cdn
	source("node_modules/echarts", `export { default } from "https://unpkg.com/echarts@5/dist/echarts.js";`)

	source("index.html", `
<html>
<head>
    <title>Vue SSR Example</title>
    <style>
        #app {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', 'Oxygen',
            'Ubuntu', 'Cantarell', 'Fira Sans', 'Droid Sans', 'Helvetica Neue',
            sans-serif;
            display: flex; align-items: center; justify-content: center; height: 100%; text-align: center;
        }
    </style>
</head>
<body>
<div id="app">__APP__</div>
</body>
</html>
`)

	source("server.js", `
import echarts from "echarts";
import serve from "ski/http/server";
import open from "ski/open";

globalThis.global = {
  __DEV__: true
};

let chart = echarts.init(null, null, {
  renderer: 'svg',
  ssr: true,
  width: 400,
  height: 300
});

export default () => serve(3000, async (req) => {
  switch (req.url) {
    case "/":
      const html = open("index.html");
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
      const app = chart.renderToSVGString();
      return new Response(html.replace("__APP__", app), {
        headers: {
          "content-type": "text/html",
        }
      });
    default:
      return new Response("Not Found: " + req.url, { status: 404, });
  }
});
`)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	_, err := js.RunString(ctx, `require("./server.js").default()`)
	if err != nil && !errors.Is(err, context.Canceled) {
		panic(err)
	}
}
