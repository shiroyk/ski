package xpath

import (
	"flag"
	"os"
	"testing"

	"github.com/shiroyk/cloudcat/plugin"
	"github.com/shiroyk/cloudcat/plugin/parser"
	"github.com/stretchr/testify/assert"
)

var (
	xpath   Parser
	ctx     *plugin.Context
	content = `
<!DOCTYPE html>
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
	ctx = plugin.NewContext(plugin.ContextOptions{})
	code := m.Run()
	os.Exit(code)
}

func TestParser(t *testing.T) {
	t.Parallel()
	if _, ok := parser.GetParser(key); !ok {
		t.Fatal("schema not registered")
	}

	_, err := xpath.GetString(ctx, 1, ``)
	if err == nil {
		t.Fatal("error should not be nil")
	}

	_, err = xpath.GetString(ctx, `<a href="https://go.dev" title="Golang page">Golang</a>`, `//a`)
	if err != nil {
		t.Error(err)
	}

	sel, _ := xpath.GetElement(ctx, content, `//div[@class="body"]`)
	_, err = xpath.GetString(ctx, sel, `//a/text()`)
	if err != nil {
		t.Error(err)
	}
}

func TestGetString(t *testing.T) {
	t.Parallel()
	if o, _ := xpath.GetStrings(ctx, content, `///`); o != nil {
		t.Fatal("Unexpected type")
	}

	str1, err := xpath.GetString(ctx, content, `//div[@id="main"]/div[contains(@class, "row")]/text()`)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "1\n2\n3\n4\n5\n6", str1)

	str2, err := xpath.GetString(ctx, content, `//div[@class="body"]/ul/li/@id`)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "a1\na2\na3", str2)

	js, err := xpath.GetString(ctx, content, `//script[1]`)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotEmpty(t, js)
}

func TestGetStrings(t *testing.T) {
	t.Parallel()
	if o, _ := xpath.GetStrings(ctx, content, `//unknown`); o != nil {
		t.Fatal("Unexpected type")
	}

	str1, err := xpath.GetStrings(ctx, content, `//div[@class="body"]/ul//a/@title`)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []string{"Google page", "Github page", "Golang page"}, str1)

	str2, err := xpath.GetStrings(ctx, content, `//div[@class="body"]/ul//a`)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []string{"Google", "Github", "Golang"}, str2)
}

func TestGetElement(t *testing.T) {
	t.Parallel()
	if o, _ := xpath.GetElement(ctx, content, `//unknown`); o != "" {
		t.Fatal("Unexpected type")
	}

	object, err := xpath.GetElement(ctx, content, `//div[@class="body"]/ul//a/..`)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, `<li id="a1"><a href="https://google.com" title="Google page">Google</a></li>
<li id="a2"><a href="https://github.com" title="Github page">Github</a></li>
<li id="a3" class="selected"><a href="https://go.dev" title="Golang page">Golang</a></li>`, object)
}

func TestGetElements(t *testing.T) {
	t.Parallel()
	if o, _ := xpath.GetElements(ctx, content, `//unknown`); o != nil {
		t.Fatal("Unexpected type")
	}

	objects, err := xpath.GetElements(ctx, content, `//div[@id="foot"]/div/@class`)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []string{
		"<class>one even row</class>", "<class>two odd row</class>",
		"<class>three even row</class>", "<class>four odd row</class>",
		"<class>five even row odder</class>", "<class>six odd row</class>",
	}, objects)
}
