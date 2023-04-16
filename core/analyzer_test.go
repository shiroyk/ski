package core

import (
	"fmt"
	"reflect"
	"testing"
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
  <script type="text/javascript">
    const url = "https://go.dev";
  </script>
</html>
`

//func TestAnalyzer(t *testing.T) {
//	ctx := plugin.NewContext(plugin.Options{
//		URL: "https://localhost",
//	})
//	testCases := []struct {
//		schema string
//		want   any
//	}{
//		{
//			`
//{ gq: '.body ul #a2 a -> href' }
//`, `"https://localhost/home"`,
//		},
//		{
//			`
//- gq: foo
//- or
//- gq: title
//`, `"Tests for Analyzer"`,
//		},
//		{
//			`
//- gq: script
//  js: |
//    eval(content + 'url;');
//`, `"https://go.dev"`,
//		},
//		{
//			`
//type: integer
//rule: { gq: '#main #n1' }
//`, 1,
//		},
//		{
//			`
//type: boolean
//rule: { gq: '#main #n1' }
//`, true,
//		},
//		{
//			`
//type: number
//rule: { gq: '#main #n2' }
//`, 2.1,
//		},
//		{
//			`
//type: array
//init:
//  - gq: '#main div'
//  - and
//  - gq: .body li
//properties:
//  item:
//    - gq: div, li -> attr(id)
//`, `[{"item":"n1"},{"item":"n2"},{"item":"n3"},{"item":"n4"},{"item":"a1"},{"item":"a2"}]`,
//		},
//		{
//			`
//type: array
//init: { gq: '#main div' }
//properties:
//  item:
//    - gq: foo
//    - or
//    - gq: div
//`, `[{"item":"1"},{"item":"2.1"},{"item":"[\"3\"]"},{"item":"{\"n4\":\"4.2\"}"}]`,
//		},
//		{
//			`
//type: object
//init: { gq: '#main' }
//properties:
//  string: { gq: '#n1' }
//  integer:
//    type: integer
//    rule: { gq: '#n1' }
//  number:
//    type: number
//    rule: { gq: '#n2' }
//  boolean:
//    type: boolean
//    rule: { gq: '#n1' }
//  array:
//    type: string
//    format: array
//    rule: { gq: '#n3' }
//  object:
//    type: object
//    rule: { gq: '#n4' }
//`, `{"array":["3"],"boolean":true,"integer":1,"number":2.1,"object":{"n4":"4.2"},"string":"1"}`,
//		},
//		{
//			`
//type: object
//format: number
//rule: { gq: '#main #n4' }
//`, `{"n4":4.2}`,
//		},
//		{
//			`
//type: array
//init: { gq: '#main div -> slice(0, 2)' }
//properties:
//  n:
//    type: number
//    rule: { gq: 'div' }
//`, `[{"n":1},{"n":2.1}]`,
//		},
//		{
//			`
//type: array
//format: number
//rule: { gq: '#main div -> slice(0, 2)' }
//`, `[1,2.1]`,
//		},
//		{
//			`
//type: array
//format: number
//rule:
//  - gq: '#main #n3'
//    json: $.*
//`, `[3]`,
//		},
//	}
//	for i, testCase := range testCases {
//		t.Run(strconv.Itoa(i), func(t *testing.T) {
//			s := new(Schema)
//			err := yaml.Unmarshal([]byte(testCase.schema), s)
//			if err != nil {
//				t.Fatal(err)
//			}
//			result := Analyze(ctx, s, content)
//			if want, ok := testCase.want.(string); ok {
//				bytes, err := json.Marshal(Analyze(ctx, s, content))
//				assert.NoError(t, err)
//				assert.JSONEq(t, want, string(bytes))
//				return
//			}
//			assert.Equal(t, testCase.want, result)
//		})
//	}
//}

func TestFormat(t *testing.T) {
	t.Parallel()
	formatter := new(defaultFormatHandler)
	testCases := []struct {
		data any
		typ  Type
		want any
	}{
		{"1", StringType, "1"},
		{"2.1", NumberType, 2.1},
		{"3", IntegerType, 3},
		{"1", BooleanType, true},
		{`{"k":"v"}`, ObjectType, map[string]any{"k": "v"}},
		{`[1,2]`, ArrayType, []any{1.0, 2.0}},
		{[]string{"1", "2"}, IntegerType, []any{1, 2}},
		{map[string]any{"k": "1"}, IntegerType, map[string]any{"k": 1}},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			got, err := formatter.Format(testCase.data, testCase.typ)
			if err != nil {
				t.Error(err)
			}
			if !reflect.DeepEqual(got, testCase.want) {
				t.Errorf("got %v, want %v", got, testCase.want)
			}
		})
	}
}
