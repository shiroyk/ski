package js

import (
	"fmt"
	"testing"

	_ "github.com/shiroyk/cloudcat/parser/parsers/json"
)

func TestContext(t *testing.T) {
	testCase := []string{
		`assert.equal(go.baseUrl, "http://localhost")`,
		`assert.equal(go.redirectUrl, "http://localhost/home")`,
		`go.setVar('v1', 114514)`,
		`assert.equal(go.getVar('v1'), 114514)`,
		`go.clearVar()
		 assert(!go.getVar('v1'), "variable should be cleared")`,
		`assert.equal(go.content, "1919810")`,
		`assert.equal(go.getString('json', '{\"key\": \"foo\"}', '$.key'), 'foo')`,
		`assert.equal(go.getStrings('json', '{\"key\": [1, 2, 3]}', '$.key[1]')[0], '2')`,
		`assert.equal(go.getElement('json', '{\"key\": \"foo\"}', '$.key'), 'foo')`,
		`assert.equal(go.getElements('json', '{\"key\": [\"foo\"]}', '$.key[0]')[0], 'foo')`,
	}
	for i, s := range testCase {
		t.Run(fmt.Sprintf("Script%v", i), func(t *testing.T) {
			_, err := testVM.RunString(s)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
