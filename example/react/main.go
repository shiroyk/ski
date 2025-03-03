package main

import (
	"context"
	"errors"
	"os/signal"
	"syscall"

	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/modulestest"
	"github.com/shiroyk/ski/modules"
	_ "github.com/shiroyk/ski/modules/buffer"
	_ "github.com/shiroyk/ski/modules/encoding"
)

func init() {
	modules.Register("server", modules.ModuleFunc(modulestest.HttpServer))
	modules.Register("open", modules.ModuleFunc(openFile))
	js.Loader().SetFileLoader(fileLoader)

	// alias module from cdn
	source("node_modules/react", `export { default } from "https://esm.sh/react@18";`)
	source("node_modules/react-dom/server", `export * from "https://esm.sh/react-dom@18/server";`)
	source("node_modules/canvas-confetti", `export { default } from "https://esm.sh/canvas-confetti@1.6.0";`)

	source("index.html", `
<html>
<head>
    <title>React SSR Example</title>
    <style>
        #root {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', 'Oxygen',
            'Ubuntu', 'Cantarell', 'Fira Sans', 'Droid Sans', 'Helvetica Neue',
            sans-serif;
            display: flex; align-items: center; justify-content: center; height: 100%; text-align: center;
        }
    </style>
    <script type="importmap">
        {"imports":{
          "react":"https://esm.sh/react@18",
          "react-dom":"https://esm.sh/react-dom@18",
          "canvas-confetti":"https://esm.sh/canvas-confetti@1.6.0"
      }}
    </script>
    <script>window.__COMPILE__ = __TIME__;</script>
    <script type="module" src="/client.js" ></script>
</head>
<body>
<div id="root">__ROOT__</div>
</body>
</html>
`)

	source(`App.jsx`, `
import React from "react";
import confetti from "canvas-confetti";

const App = ({ compile }) => {
  const [count, setCount] = React.useState(0);

  const onClick = (e) => {
    confetti({
      particleCount: 100,
      spread: 70,
      origin: { y: 0.6 }
    });
    setCount(count + 1);
  }

  return (
    <div>
      <h1>React SSR Example</h1>
      <p>Server side rendered in {compile}ms</p>
      <div onClick={onClick}>Click me: {count}</div>
    </div>
  );
};

export default App;
`)
	source("client.jsx", `
import App from "./app.js";
import React from "react";
import ReactDOM from "react-dom";

ReactDOM.hydrate(React.createElement(App, { compile: window.__COMPILE__ ?? "-" }), document.getElementById("root"));
`)
	source("server.js", `
import App from "./App.jsx";
import React from "react";
import {renderToString} from "react-dom/server";
import createServer from "ski/server";
import open from "ski/open";

export default () => createServer("localhost:3000", (req, res) => {
  switch (req.path) {
    case "/":
      const html = open("index.html");
      const start = Date.now();
      const app = renderToString(React.createElement(App));
      res.end(html.replace("__ROOT__", app).replace("__TIME__", Date.now() - start));
      break;
    case "/client.js":
      res.setHeader("Content-Type", "text/javascript");
      res.end(open("client.jsx"));
      break;
    case "/app.js":
      res.setHeader("Content-Type", "text/javascript");
      res.end(open("App.jsx"));
      break;
    default:
      res.statusCode = 404;
      res.end("Not Found: "+req.path);
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
