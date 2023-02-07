package parser

import (
	"testing"

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
