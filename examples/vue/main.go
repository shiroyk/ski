package main

import (
	"context"
	"errors"
	"os/signal"
	"syscall"

	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/modules"
	_ "github.com/shiroyk/ski/modules/buffer"
	_ "github.com/shiroyk/ski/modules/http"
)

func init() {
	modules.Register("open", modules.ModuleFunc(openFile))
	modules.Register("now", modules.ModuleFunc(now))
	js.Loader().SetFileLoader(fileLoader)

	// alias module from cdn
	source("node_modules/vue", `export * from "https://esm.sh/vue@3";`)
	source("node_modules/vue/server-renderer", `export * from "https://esm.sh/@vue/server-renderer@3";`)
	source("node_modules/canvas-confetti", `export { default } from "https://esm.sh/canvas-confetti@1.6.0";`)

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
    <script type="importmap">
        {"imports":{
            "vue":"https://esm.sh/vue@3",
            "vue/server-renderer":"https://esm.sh/@vue/server-renderer@3",
            "canvas-confetti":"https://esm.sh/canvas-confetti@1.6.0"
        }}
    </script>
    <script>window.__COMPILE__ = __TIME__;</script>
    <script type="module" src="/client.js" ></script>
</head>
<body>
<div id="app">__APP__</div>
</body>
</html>
`)

	source("App.vue", `
<script setup>
import { ref } from 'vue';
import confetti from "canvas-confetti";

const props = defineProps(['compile'])
const count = ref(0);

const onClick = (e) => {
  confetti({
    particleCount: 100,
    spread: 70,
    origin: { y: 0.6 }
  });
  count.value++;
}
</script>

<template>
  <div>
    <h1>Vue SSR Example</h1>
	<p>Server side rendered in {{props.compile}}ms</p>
    <div @click="onClick">Click me: {{count}}</div>
  </div>
</template>
`)

	source("client.js", `
import App from './app.js';
import { createSSRApp } from 'vue';
const app = createSSRApp(App, { compile: window.__COMPILE__ ?? "-" });
app.mount('#app');
`)

	source("server.js", `
import serve from "ski/http/server";
import open from "ski/open";
import now from "ski/now";
import { createSSRApp } from "vue";
import { renderToString } from "vue/server-renderer";
import App from "./App.vue?ssr";

export default () => serve(3000, async (req) => {
  switch (req.url) {
    case "/":
      const html = open("index.html");
      const start = now();
      const app = await renderToString(createSSRApp(App));
      const take = ((now() - start) / 1000).toFixed(2);
      return new Response(html.replace("__APP__", app).replace("__TIME__", take), {
        headers: {
          "content-type": "text/html",
        }
      });
    case "/client.js":
      return new Response(open("client.js"), {
        headers: {
          "content-type": "text/javascript",
        }
      });
    case "/app.js":
      return new Response(open("App.vue"), {
        headers: {
          "content-type": "text/javascript",
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
