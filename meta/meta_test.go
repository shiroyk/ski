package meta

import (
	"encoding/json"
	"testing"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

var yamlConfig = `
title: [ gq: title ]
href: [ gq: ".body ul #a1 a -> href" ]
home: [ gq: ".body ul #a2 a -> href" ]
object:
 type: object
 init: [ gq: "#main" ]
 properties:
  string: { gq: "#n1" }
  integer:
   type: integer
   rule: [ gq: "#n1" ]
  number:
   type: number
   rule: [ gq: "#n2" ]
  boolean:
   type: boolean
   rule: [ gq: "#n1" ]
  array:
   type: string
   format: array
   rule: [ gq: "#n3" ]
  object:
   type: object
   rule: [ gq: "#n4" ]
object1:
 type: object
 format: number
 rule:
  or:
   - gq: "#main #nn"
   - gq: "#main #n4"
array:
 type: array
 init: [ gq: "#main div -> slice(0, 2)" ]
 properties:
  n:
   type: number
   rule: [ gq: "div -> text" ]
array1:
 type: array
 format: number
 rule: [ gq: "#main div -> slice(0, 2)" ]
array2:
 type: array
 format: number
 rule:
  - gq: "#main #n3"
  - json: "$.*"
`

func TestYaml(t *testing.T) {
	r := new(Meta)

	err := yaml.Unmarshal([]byte(yamlConfig), r)
	if err != nil {
		t.Fatal(err)
	}
}

var jsonConfig = `
{
  "title": {
    "gq": "title"
  },
  "href": {
    "gq": ".body ul #a1 a -> href"
  },
  "home": {
    "gq": ".body ul #a2 a -> href"
  },
  "object": {
    "type": "object",
    "init": {
      "gq": "#main"
    },
    "properties": {
      "string": {
        "gq": "#n1"
      },
      "integer": {
        "type": "integer",
        "rule": {
          "gq": "#n1"
        }
      },
      "number": {
        "type": "number",
        "rule": {
          "gq": "#n2"
        }
      },
      "boolean": {
        "type": "boolean",
        "rule": {
          "gq": "#n1"
        }
      },
      "array": {
        "type": "string",
        "format": "array",
        "rule": {
          "gq": "#n3"
        }
      },
      "object": {
        "type": "object",
        "rule": {
          "gq": "#n4"
        }
      }
    }
  },
  "object1": {
    "type": "object",
    "format": "number",
    "rule": {
      "or": [
        {
          "gq": "#main #nn"
        },
        {
          "gq": "#main #n4"
        }
      ]
    }
  },
  "array": {
    "type": "array",
    "init": {
      "gq": "#main div -> slice(0, 2)"
    },
    "properties": {
      "n": {
        "type": "number",
        "rule": {
          "gq": "div -> text"
        }
      }
    }
  },
  "array1": {
    "type": "array",
    "format": "number",
    "rule": {
      "gq": "#main div -> slice(0, 2)"
    }
  },
  "array2": {
    "type": "array",
    "format": "number",
    "rule": [
      {
        "gq": "#main #n3"
      },
      {
        "json": "$.*"
      }
    ]
  }
}
`

func TestJSON(t *testing.T) {
	r := new(Meta)

	err := json.Unmarshal([]byte(jsonConfig), r)
	if err != nil {
		t.Fatal(err)
	}
}

var tomlConfig = `
[title]
gq = "title"

[href]
gq = ".body ul #a1 a -> href"

[home]
gq = ".body ul #a2 a -> href"

[object]
type = "object"

  [object.init]
  gq = "#main"

[object.properties.string]
gq = "#n1"

[object.properties.integer]
type = "integer"

  [object.properties.integer.rule]
  gq = "#n1"

[object.properties.number]
type = "number"

  [object.properties.number.rule]
  gq = "#n2"

[object.properties.boolean]
type = "boolean"

  [object.properties.boolean.rule]
  gq = "#n1"

[object.properties.array]
type = "string"
format = "array"

  [object.properties.array.rule]
  gq = "#n3"

[object.properties.object]
type = "object"

  [object.properties.object.rule]
  gq = "#n4"

[object1]
type = "object"
format = "number"

[[object1.rule.or]]
gq = "#main #nn"

[[object1.rule.or]]
gq = "#main #n4"

[array]
type = "array"

  [array.init]
  gq = "#main div -> slice(0, 2)"

[array.properties.n]
type = "number"

  [array.properties.n.rule]
  gq = "div -> text"

[array1]
type = "array"
format = "number"

  [array1.rule]
  gq = "#main div -> slice(0, 2)"

[array2]
type = "array"
format = "number"

  [[array2.rule]]
  gq = "#main #n3"

  [[array2.rule]]
  json = "$.*"
`

func TestToml(t *testing.T) {
	r := new(Meta)

	err := toml.Unmarshal([]byte(tomlConfig), r)
	if err != nil {
		t.Fatal(err)
	}
}
