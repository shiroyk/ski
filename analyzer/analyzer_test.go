package analyzer

import (
	"encoding/json"
	"testing"

	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/fetch"
	"github.com/shiroyk/cloudcat/parser"
)

var (
	schema = parser.NewSchema(parser.ObjectType).
		AddProperty("title", *parser.NewSchema(parser.StringType).
			AddRule(parser.NewStep("gq", "title"))).
		AddProperty("href", *parser.NewSchema(parser.StringType).
			AddRule(parser.NewStep("gq", `.body ul #a1 a -> href`))).
		AddProperty("home", *parser.NewSchema(parser.StringType).
			AddRule(parser.NewStep("gq", `.body ul #a2 a -> href`))).
		AddProperty("object", *parser.NewSchema(parser.ObjectType).
			AddInit(parser.NewStep("gq", "#main")).
			AddProperty("string", *parser.NewSchema(parser.StringType).
				AddRule(parser.NewStep("gq", "#n")).
				AddOpRule(parser.OperatorOr, parser.NewStep("gq", "#n1"))).
			AddProperty("integer", *parser.NewSchema(parser.IntegerType).
				AddRule(parser.NewStep("gq", "#n1"))).
			AddProperty("number", *parser.NewSchema(parser.NumberType).
				AddRule(parser.NewStep("gq", "#n2"))).
			AddProperty("boolean", *parser.NewSchema(parser.BooleanType).
				AddRule(parser.NewStep("gq", "#n1"))).
			AddProperty("array", *parser.NewSchema(parser.StringType, parser.ArrayType).
				AddRule(parser.NewStep("gq", "#n3"))).
			AddProperty("object", *parser.NewSchema(parser.StringType, parser.ObjectType).
				AddRule(parser.NewStep("gq", "#n4")))).
		AddProperty("object1", *parser.NewSchema(parser.ObjectType, parser.NumberType).
			AddRule(parser.NewStep("gq", `#main #n4`))).
		AddProperty("array", *parser.NewSchema(parser.ArrayType).
			AddInit(parser.NewStep("gq", `#main div -> slice(0, 2)`)).
			AddProperty("n", *parser.NewSchema(parser.NumberType).
				AddRule(parser.NewStep("gq", `div -> text`)))).
		AddProperty("array1", *parser.NewSchema(parser.ArrayType, parser.NumberType).
			AddRule(parser.NewStep("gq", `#main div -> slice(0, 2)`))).
		AddProperty("array2", *parser.NewSchema(parser.ArrayType, parser.NumberType).
			AddRule(parser.NewStep("gq", `#main #n3`), parser.NewStep("json", `$.*`)))
	content = `<!DOCTYPE html>
<html lang="en">
  <head>
    <title>Tests for Analyzer</title>
  </head>
  <body>
    <div id="main">
      <div id="n1">1</div>
      <div id="n2">2.1</div>
      <div id="n3">["3"]</div>
      <div id="n4">{"n4":"4"}</div>
    </div>
	<div class="body">
        <ul id="url">
			<li id="a1" class="selected"><a href="https://go.dev" title="Golang page">Golang</a></li>
			<li id="a2"><a href="/home" title="Home page">Home</a></li>
		</ul>
	</div>
  </body>
</html>
`
	result = "{" +
		`"array":[{"n":1},{"n":2.1}],` +
		`"array1":[1,2.1],` +
		`"array2":[3],` +
		`"home":"https://localhost/home",` +
		`"href":"https://go.dev",` +
		`"object":{"array":["3"],` +
		`"boolean":true,` +
		`"integer":1,` +
		`"number":2.1,` +
		`"object":{"n4":"4"},` +
		`"string":"1"},` +
		`"object1":{"n4":4},` +
		`"title":"Tests for Analyzer"` +
		"}"
)

func TestAnalyzer(t *testing.T) {
	di.Provide(fetch.NewFetcher(fetch.Options{}))
	ctx := parser.NewContext(parser.Options{
		Url: "https://localhost",
	})
	bytes, err := json.Marshal(NewAnalyzer().ExecuteSchema(ctx, schema, content))
	if err != nil {
		t.Fatal(err)
	}
	if string(bytes) != result {
		t.Fatalf("Unexpected string %s", bytes)
	}
}
