package analyzer

import (
	"encoding/json"
	"testing"

	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/fetcher"
	. "github.com/shiroyk/cloudcat/meta"
	"github.com/shiroyk/cloudcat/parser"
)

var (
	schema = NewSchema(ObjectType).
		AddProperty("title", *NewSchema(StringType).
			AddRule(NewStep("gq", "title"))).
		AddProperty("href", *NewSchema(StringType).
			AddRule(NewStep("gq", `.body ul #a1 a -> href`))).
		AddProperty("home", *NewSchema(StringType).
			AddRule(NewStep("gq", `.body ul #a2 a -> href`))).
		AddProperty("object", *NewSchema(ObjectType).
			AddInit(NewStep("gq", "#main")).
			AddProperty("string", *NewSchema(StringType).
				AddRule(NewStep("gq", "#n")).
				AddOpRule(OperatorOr, NewStep("gq", "#n1"))).
			AddProperty("integer", *NewSchema(IntegerType).
				AddRule(NewStep("gq", "#n1"))).
			AddProperty("number", *NewSchema(NumberType).
				AddRule(NewStep("gq", "#n2"))).
			AddProperty("boolean", *NewSchema(BooleanType).
				AddRule(NewStep("gq", "#n1"))).
			AddProperty("array", *NewSchema(StringType, ArrayType).
				AddRule(NewStep("gq", "#n3"))).
			AddProperty("object", *NewSchema(StringType, ObjectType).
				AddRule(NewStep("gq", "#n4")))).
		AddProperty("object1", *NewSchema(ObjectType, NumberType).
			AddRule(NewStep("gq", `#main #n4`))).
		AddProperty("array", *NewSchema(ArrayType).
			AddInit(NewStep("gq", `#main div -> slice(0, 2)`)).
			AddProperty("n", *NewSchema(NumberType).
				AddRule(NewStep("gq", `div -> text`)))).
		AddProperty("array1", *NewSchema(ArrayType, NumberType).
			AddRule(NewStep("gq", `#main div -> slice(0, 2)`))).
		AddProperty("array2", *NewSchema(ArrayType, NumberType).
			AddRule(NewStep("gq", `#main #n3`), NewStep("json", `$.*`)))
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
	di.Provide(fetcher.NewFetcher(&fetcher.Options{}))
	ctx := parser.NewContext(&parser.Options{
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
