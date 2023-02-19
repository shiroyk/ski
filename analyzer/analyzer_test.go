package analyzer

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/shiroyk/cloudcat/parser"
	"github.com/shiroyk/cloudcat/schema"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
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
		Schema, Result string
	}{
		{`
items:
  type: array
  init:
    - gq: '#main div'
    - and
    - gq: .body li
  properties:
    item:
      - gq: div, li -> attr(id)
`, `{"items":[{"item":"n1"},{"item":"n2"},{"item":"n3"},{"item":"n4"},{"item":"a1"},{"item":"a2"}]}`,
		},
		{`
title:
  - gq: foo
  - or
  - gq: title
`, `{"title":"Tests for Analyzer"}`,
		},
		{`
items:
  type: array
  init: { gq: '#main div' }
  properties:
    item:
      - gq: foo
      - or
      - gq: div
`, `{"items":[{"item":"1"},{"item":"2.1"},{"item":"[\"3\"]"},{"item":"{\"n4\":\"4.2\"}"}]}`,
		},
		{`
home: { gq: '.body ul #a2 a -> href' }
`, `{"home":"https://localhost/home"}`,
		},
		{`
object:
  type: object
  init: { gq: '#main' }
  properties:
    string: { gq: '#n1' }
    integer:
      type: integer
      rule: { gq: '#n1' }
    number:
      type: number
      rule: { gq: '#n2' }
    boolean:
      type: boolean
      rule: { gq: '#n1' }
    array:
      type: string
      format: array
      rule: { gq: '#n3' }
    object:
      type: object
      rule: { gq: '#n4' }
`, `{"object":{"array":["3"],"boolean":true,"integer":1,"number":2.1,"object":{"n4":"4.2"},"string":"1"}}`,
		},
		{`
object1: 
  type: object
  format: number
  rule: { gq: '#main #n4' }
`, `{"object1":{"n4":4.2}}`,
		},
		{`
array:
  type: array
  init: { gq: '#main div -> slice(0, 2)' }
  properties:
    n:
      type: number
      rule: { gq: 'div' }
`, `{"array":[{"n":1},{"n":2.1}]}`,
		},
		{`
array1:
  type: array
  format: number
  rule: { gq: '#main div -> slice(0, 2)' }
`, `{"array1":[1,2.1]}`,
		},
		{`
array2:
  type: array
  format: number
  rule: 
    - gq: '#main #n3'
      json: $.*
`, `{"array2":[3]}`,
		},
	}
	for i, testCase := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			s := new(schema.Schema)
			err := yaml.Unmarshal([]byte(testCase.Schema), s)
			if err != nil {
				t.Fatal(err)
			}
			bytes, err := json.Marshal(Analyze(ctx, s, content))
			if err != nil {
				t.Error(err)
			}
			assert.JSONEq(t, testCase.Result, string(bytes))
		})
	}
}
