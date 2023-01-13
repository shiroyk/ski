package js

import (
	"testing"
)

func TestContext(t *testing.T) {
	_, err := testVM.RunString(`
		assert.equal(go.baseUrl, "http://localhost")
		assert.equal(go.redirectUrl, "http://localhost/home")
		go.setVar('v1', 114514)
		assert.equal(go.getVar('v1'), 114514)
		go.clearVar()
		assert(!go.getVar('v1'), "variable should be cleared")
		assert.equal(go.content, "1919810")
	`)
	if err != nil {
		t.Fatal(err)
	}
}
