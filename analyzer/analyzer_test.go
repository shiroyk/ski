package analyzer

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/fetch"
	"github.com/shiroyk/cloudcat/parser"
)

var content = `<!DOCTYPE html>
<html lang="en">
  <head>
    <title>Tests for Analyzer</title>
  </head>
  <body>
    <div id="main">
      <div id="n1">1</div>
      <div id="n2">2.1</div>
      <div id="n3">["3"]</div>
      <div id="n4">{"n4":"4.2"}</div>
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

func TestAnalyzer(t *testing.T) {
	di.Provide(fetch.NewFetcher(fetch.Options{}))
	ctx := parser.NewContext(parser.Options{
		URL: "https://localhost",
	})
	testCases := []struct {
		Schema *parser.Schema
		Result string
	}{
		{parser.NewSchema(parser.ObjectType).
			AddProperty("title", *parser.NewSchema(parser.StringType).
				AddRule(parser.NewStep("gq", "foo")).
				AddRuleOp(parser.OperatorOr).
				AddRule(parser.NewStep("gq", "title"))),
			`{"title":"Tests for Analyzer"}`,
		},
		{parser.NewSchema(parser.ArrayType).
			AddInit(parser.NewStep("gq", "#main div")).
			AddProperty("item", *parser.NewSchema(parser.StringType).
				AddRule(parser.NewStep("gq", "foo")).
				AddRuleOp(parser.OperatorOr).
				AddRule(parser.NewStep("gq", "div"))),
			`[{"item":"1"},{"item":"2.1"},{"item":"[\"3\"]"},{"item":"{\"n4\":\"4.2\"}"}]`,
		},
		{parser.NewSchema(parser.ArrayType).
			AddInit(parser.NewStep("gq", "#main div")).
			AddInitOp(parser.OperatorAnd).
			AddInit(parser.NewStep("gq", ".body li")).
			AddProperty("item", *parser.NewSchema(parser.StringType).
				AddRule(parser.NewStep("gq", "* -> attr(id)"))),
			`[{"item":"n1"},{"item":"n2"},{"item":"n3"},{"item":"n4"},{"item":"a1"},{"item":"a2"}]`,
		},
		{parser.NewSchema(parser.ObjectType).
			AddProperty("home", *parser.NewSchema(parser.StringType).
				AddRule(parser.NewStep("gq", `.body ul #a2 a -> href`))),
			`{"home":"https://localhost/home"}`,
		},
		{parser.NewSchema(parser.ObjectType).
			AddProperty("object", *parser.NewSchema(parser.ObjectType).
				AddInit(parser.NewStep("gq", "#main")).
				AddProperty("string", *parser.NewSchema(parser.StringType).
					AddRule(parser.NewStep("gq", "#n1"))).
				AddProperty("integer", *parser.NewSchema(parser.IntegerType).
					AddRule(parser.NewStep("gq", "#n1"))).
				AddProperty("number", *parser.NewSchema(parser.NumberType).
					AddRule(parser.NewStep("gq", "#n2"))).
				AddProperty("boolean", *parser.NewSchema(parser.BooleanType).
					AddRule(parser.NewStep("gq", "#n1"))).
				AddProperty("array", *parser.NewSchema(parser.StringType, parser.ArrayType).
					AddRule(parser.NewStep("gq", "#n3"))).
				AddProperty("object", *parser.NewSchema(parser.StringType, parser.ObjectType).
					AddRule(parser.NewStep("gq", "#n4")))),
			`{"object":{"array":["3"],"boolean":true,"integer":1,"number":2.1,"object":{"n4":"4.2"},"string":"1"}}`,
		},
		{parser.NewSchema(parser.ObjectType).
			AddProperty("object1", *parser.NewSchema(parser.ObjectType, parser.NumberType).
				AddRule(parser.NewStep("gq", `#main #n4`))),
			`{"object1":{"n4":4.2}}`,
		},
		{parser.NewSchema(parser.ObjectType).
			AddProperty("array", *parser.NewSchema(parser.ArrayType).
				AddInit(parser.NewStep("gq", `#main div -> slice(0, 2)`)).
				AddProperty("n", *parser.NewSchema(parser.NumberType).
					AddRule(parser.NewStep("gq", `div -> text`)))),
			`{"array":[{"n":1},{"n":2.1}]}`,
		},
		{parser.NewSchema(parser.ObjectType).
			AddProperty("array1", *parser.NewSchema(parser.ArrayType, parser.NumberType).
				AddRule(parser.NewStep("gq", `#main div -> slice(0, 2)`))),
			`{"array1":[1,2.1]}`,
		},
		{parser.NewSchema(parser.ObjectType).
			AddProperty("array2", *parser.NewSchema(parser.ArrayType, parser.NumberType).
				AddRule(parser.NewStep("gq", `#main #n3`), parser.NewStep("json", `$.*`))),
			`{"array2":[3]}`,
		},
	}
	for i, testCase := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			bytes, err := json.Marshal(NewAnalyzer().ExecuteSchema(ctx, testCase.Schema, content))
			if err != nil {
				t.Fatal(err)
			}
			if string(bytes) != testCase.Result {
				t.Fatalf("want %s, got %s", testCase.Result, bytes)
			}
		})
	}
}
