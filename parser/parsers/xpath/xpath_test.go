package xpath

import (
	"flag"
	"os"
	"reflect"
	"testing"

	"github.com/shiroyk/cloudcat/parser"
)

var (
	xpath   Parser
	ctx     *parser.Context
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
	ctx = parser.NewContext(parser.Options{})
	code := m.Run()
	os.Exit(code)
}

func TestParser(t *testing.T) {
	_, ok := parser.GetParser(key)
	if !ok {
		t.Fatal("schema not registered")
	}

	_, err := xpath.GetString(ctx, 1, ``)
	if err == nil {
		t.Fatal("error should be nil")
	}

	node := `<a href="https://go.dev" title="Golang page">Golang</a>`
	_, err = xpath.GetString(ctx, &node, `//a`)
	if err != nil {
		t.Fatal(err)
	}

	sel, _ := xpath.GetElement(ctx, content, `//div[@class="body"]`)
	_, err = xpath.GetString(ctx, sel, `//a/text()`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetString(t *testing.T) {
	if o, _ := xpath.GetStrings(ctx, content, `///`); o != nil {
		t.Fatal("Unexpected type")
	}

	str1, err := xpath.GetString(ctx, content, `//div[@id="main"]/div[contains(@class, "row")]/text()`)
	if err != nil {
		t.Fatal(err)
	}
	if str1 != "1, 2, 3, 4, 5, 6" {
		t.Fatalf("Unexpected string %s", str1)
	}

	str2, err := xpath.GetString(ctx, content, `//div[@class="body"]/ul/li/@id`)
	if err != nil {
		t.Fatal(err)
	}
	if str2 != "a1, a2, a3" {
		t.Fatalf("Unexpected string %s", str2)
	}

	js, err := xpath.GetString(ctx, content, `//script[1]`)
	if err != nil {
		t.Fatal(err)
	}
	if len(js) == 0 {
		t.Fatalf("Unexpected string %s", js)
	}
}

func TestGetStrings(t *testing.T) {
	if o, _ := xpath.GetStrings(ctx, content, `//unknown`); o != nil {
		t.Fatal("Unexpected type")
	}

	str1, err := xpath.GetStrings(ctx, content, `//div[@class="body"]/ul//a/@title`)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(str1, []string{"Google page", "Github page", "Golang page"}) {
		t.Fatalf("Unexpected strings %s", str1)
	}

	str2, err := xpath.GetStrings(ctx, content, `//div[@class="body"]/ul//a`)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(str2, []string{"Google", "Github", "Golang"}) {
		t.Fatalf("Unexpected strings %s", str2)
	}
}

func TestGetElement(t *testing.T) {
	if o, _ := xpath.GetElement(ctx, content, `//unknown`); o != "" {
		t.Fatal("Unexpected type")
	}

	object, err := xpath.GetElement(ctx, content, `//div[@class="body"]/ul//a/..`)
	if err != nil {
		t.Fatal(err)
	}
	if object != `<li id="a1"><a href="https://google.com" title="Google page">Google</a></li>
<li id="a2"><a href="https://github.com" title="Github page">Github</a></li>
<li id="a3" class="selected"><a href="https://go.dev" title="Golang page">Golang</a></li>` {
		t.Fatalf("Unexpected object %s", object)
	}
}

func TestGetElements(t *testing.T) {
	if o, _ := xpath.GetElements(ctx, content, `//unknown`); o != nil {
		t.Fatal("Unexpected type")
	}

	objects, err := xpath.GetElements(ctx, content, `//div[@id="foot"]/div/@class`)
	if err != nil {
		t.Fatal(err)
	}

	if len(objects) != 6 {
		t.Fatalf("Unexpected node length %d", len(objects))
	}
}
