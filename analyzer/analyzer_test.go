package analyzer

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/shiroyk/cloudcat/parser"
	"github.com/shiroyk/cloudcat/schema"
	"github.com/stretchr/testify/assert"
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
	ctx := parser.NewContext(parser.Options{
		URL: "https://localhost",
	})
	testCases := []struct {
		Schema *schema.Schema
		Result string
	}{
		{schema.NewSchema(schema.ArrayType).
			AddInit(schema.NewStep("gq", "#main div")).
			AddInitOp(schema.OperatorAnd).
			AddInit(schema.NewStep("gq", ".body li")).
			AddProperty("item", *schema.NewSchema(schema.StringType).
				AddRule(schema.NewStep("gq", "div, li -> attr(id)"))),
			`[{"item":"n1"},{"item":"n2"},{"item":"n3"},{"item":"n4"},{"item":"a1"},{"item":"a2"}]`,
		},
		{schema.NewSchema(schema.ObjectType).
			AddProperty("title", *schema.NewSchema(schema.StringType).
				AddRule(schema.NewStep("gq", "foo")).
				AddRuleOp(schema.OperatorOr).
				AddRule(schema.NewStep("gq", "title"))),
			`{"title":"Tests for Analyzer"}`,
		},
		{schema.NewSchema(schema.ArrayType).
			AddInit(schema.NewStep("gq", "#main div")).
			AddProperty("item", *schema.NewSchema(schema.StringType).
				AddRule(schema.NewStep("gq", "foo")).
				AddRuleOp(schema.OperatorOr).
				AddRule(schema.NewStep("gq", "div"))),
			`[{"item":"1"},{"item":"2.1"},{"item":"[\"3\"]"},{"item":"{\"n4\":\"4.2\"}"}]`,
		},
		{schema.NewSchema(schema.ObjectType).
			AddProperty("home", *schema.NewSchema(schema.StringType).
				AddRule(schema.NewStep("gq", `.body ul #a2 a -> href`))),
			`{"home":"https://localhost/home"}`,
		},
		{schema.NewSchema(schema.ObjectType).
			AddProperty("object", *schema.NewSchema(schema.ObjectType).
				AddInit(schema.NewStep("gq", "#main")).
				AddProperty("string", *schema.NewSchema(schema.StringType).
					AddRule(schema.NewStep("gq", "#n1"))).
				AddProperty("integer", *schema.NewSchema(schema.IntegerType).
					AddRule(schema.NewStep("gq", "#n1"))).
				AddProperty("number", *schema.NewSchema(schema.NumberType).
					AddRule(schema.NewStep("gq", "#n2"))).
				AddProperty("boolean", *schema.NewSchema(schema.BooleanType).
					AddRule(schema.NewStep("gq", "#n1"))).
				AddProperty("array", *schema.NewSchema(schema.StringType, schema.ArrayType).
					AddRule(schema.NewStep("gq", "#n3"))).
				AddProperty("object", *schema.NewSchema(schema.StringType, schema.ObjectType).
					AddRule(schema.NewStep("gq", "#n4")))),
			`{"object":{"array":["3"],"boolean":true,"integer":1,"number":2.1,"object":{"n4":"4.2"},"string":"1"}}`,
		},
		{schema.NewSchema(schema.ObjectType).
			AddProperty("object1", *schema.NewSchema(schema.ObjectType, schema.NumberType).
				AddRule(schema.NewStep("gq", `#main #n4`))),
			`{"object1":{"n4":4.2}}`,
		},
		{schema.NewSchema(schema.ObjectType).
			AddProperty("array", *schema.NewSchema(schema.ArrayType).
				AddInit(schema.NewStep("gq", `#main div -> slice(0, 2)`)).
				AddProperty("n", *schema.NewSchema(schema.NumberType).
					AddRule(schema.NewStep("gq", `div -> text`)))),
			`{"array":[{"n":1},{"n":2.1}]}`,
		},
		{schema.NewSchema(schema.ObjectType).
			AddProperty("array1", *schema.NewSchema(schema.ArrayType, schema.NumberType).
				AddRule(schema.NewStep("gq", `#main div -> slice(0, 2)`))),
			`{"array1":[1,2.1]}`,
		},
		{schema.NewSchema(schema.ObjectType).
			AddProperty("array2", *schema.NewSchema(schema.ArrayType, schema.NumberType).
				AddRule(schema.NewStep("gq", `#main #n3`), schema.NewStep("json", `$.*`))),
			`{"array2":[3]}`,
		},
	}
	a := NewAnalyzer()
	for i, testCase := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			bytes, err := json.Marshal(a.ExecuteSchema(ctx, testCase.Schema, content))
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, testCase.Result, string(bytes))
		})
	}
}
