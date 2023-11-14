package gq

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/shiroyk/cloudcat/plugin"
	"github.com/shiroyk/cloudcat/plugin/parser"
	"github.com/stretchr/testify/assert"
	"log/slog"
)

var (
	gq      Parser
	ctx     *plugin.Context
	content = `<!DOCTYPE html>
<html lang="en">
  <head>
    <title>Tests for siblings</title>
  </head>
  <body>
    <div id="main">
      <div id="n1" class="one even row">1</div>
      <div id="n2" class="two odd row">2</div>
      <div id="n3" class="three even row">3</div>
      <div id="n4" class="four odd row">4</div>
      <div id="n5" class="five even row">5</div>
      <div id="n6" class="six odd row">6</div>
    </div>
	<div class="body">
        <ul id="url">
			<li id="a1"><a href="https://google.com" title="Google page">Google</a></li>
			<li id="a2"><a href="https://github.com" title="Github page">Github</a></li>
			<li id="a3" class="selected"><a href="https://go.dev" title="Golang page">Golang</a></li>
			<li id="a4"><a href="/home" title="Home page">Home</a></li>
		</ul>
	</div>
    <div id="foot">
      <div id="nf1" class="one even row">f1</div>
      <div id="nf2" class="two odd row">f2</div>
      <div id="nf3" class="three even row">f3</div>
      <div id="nf4" class="four odd row">f4</div>
      <div id="nf5" class="five even row odder">f5</div>
      <div id="nf6" class="six odd row">f6</div>
    </div>
	<script type="text/javascript">
		(function() {
		  const ga = document.createElement("script"); ga.type = "text/javascript"; ga.async = true;
		  ga.src = ("https:" === document.location.protocol ? "https://ssl" : "http://www") + ".google-analytics.com/ga.js";
		  const s = document.getElementsByTagName("script")[0]; s.parentNode.insertBefore(ga, s);
		})();
	</script>
  </body>
</html>
`
)

func TestMain(m *testing.M) {
	flag.Parse()
	ctx = plugin.NewContext(plugin.ContextOptions{
		URL: "https://localhost",
	})
	gq = Parser{parseFuncs: builtins()}
	code := m.Run()
	os.Exit(code)
}

func assertGetString(t *testing.T, arg string, expected string) {
	str, err := gq.GetString(ctx, content, arg)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, expected, str)
}

func assertGetStrings(t *testing.T, arg string, expected []string) {
	str, err := gq.GetStrings(ctx, content, arg)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, expected, str)
}

func assertGetElement(t *testing.T, arg string, expected string) {
	ele, err := gq.GetElement(ctx, content, arg)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, expected, ele)
}

func assertGetElements(t *testing.T, arg string, expected []string) {
	objs, err := gq.GetElements(ctx, content, arg)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, expected, objs)
}

func TestParser(t *testing.T) {
	t.Parallel()
	if _, ok := parser.GetParser(Key); !ok {
		t.Fatal("schema not registered")
	}

	if _, err := gq.GetString(ctx, 0, ``); err == nil {
		t.Fatal("expected empty error")
	}

	if _, err := gq.GetString(ctx, nil, ``); err != nil {
		t.Fatal(err)
	}

	if _, err := gq.GetString(ctx, []string{"<br>"}, ``); err != nil {
		t.Fatal(err)
	}

	if _, err := gq.GetString(ctx, `<a href="https://go.dev" title="Golang page">Golang</a>`, ``); err != nil {
		t.Fatal(err)
	}

	sel, _ := gq.GetElement(ctx, content, `#main .row`)
	if _, err := gq.GetString(ctx, sel, ``); err != nil {
		t.Fatal(err)
	}
}

func TestGetString(t *testing.T) {
	t.Parallel()
	assertGetString(t, `#main .row -> text`, "1\n2\n3\n4\n5\n6")

	assertGetString(t, `.body ul a -> parent(li) -> attr(id) -> join(-)`, "a1-a2-a3-a4")

	assertGetString(t, `script -> slice(0) -> attr(type)`, "text/javascript")
}

func TestGetStrings(t *testing.T) {
	t.Parallel()
	assertGetStrings(t, `.body ul li -> child(a) -> attr(title)`, []string{"Google page", "Github page", "Golang page", "Home page"})

	assertGetStrings(t, `.body ul a`, []string{"Google", "Github", "Golang", "Home"})
}

func TestGetElement(t *testing.T) {
	t.Parallel()
	assertGetElement(t, `.body ul a -> parents(li)`, `<li id="a1"><a href="https://google.com" title="Google page">Google</a></li>`)

	assertGetElement(t, `.body ul a -> slice(1) -> text`, `Github`)
}

func TestGetElements(t *testing.T) {
	t.Parallel()
	assertGetElements(t, `#foot div -> slice(0, 3)`, []string{
		`<div id="nf1" class="one even row">f1</div>`,
		`<div id="nf2" class="two odd row">f2</div>`,
		`<div id="nf3" class="three even row">f3</div>`,
	})

	assertGetElements(t, `#foot div -> slice(0, 3) -> text`, []string{"f1", "f2", "f3"})
}

func TestExternalFunc(t *testing.T) {
	{
		fun := func(logger *slog.Logger) GFunc {
			return func(_ *plugin.Context, content any, args ...string) (any, error) {
				logger.Info(fmt.Sprintf("result type was %T", content))
				return content, nil
			}
		}
		p := NewGoQueryParser(FuncMap{"logger": fun(slog.Default())})
		_, err := p.GetString(ctx, content, ".body ul a -> logger -> text")
		assert.NoError(t, err)
	}

	{
		fun := func(_ *plugin.Context, content any, args ...string) (any, error) {
			return nil, nil
		}
		p := NewGoQueryParser(FuncMap{"nil": fun})
		_, err := p.GetString(ctx, content, ".body ul a -> nil -> text")
		assert.NoError(t, err)
		_, err = p.GetStrings(ctx, content, ".body ul a -> nil -> text")
		assert.NoError(t, err)
		_, err = p.GetElement(ctx, content, ".body ul a -> nil -> text")
		assert.NoError(t, err)
		_, err = p.GetElements(ctx, content, ".body ul a -> nil -> text")
		assert.NoError(t, err)
	}
}
