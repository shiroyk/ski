package js

import (
	"testing"
)

func TestShortener(t *testing.T) {
	_, err := testVM.RunString(`
		id = go.shortener.set('http://localhost', {"cookie": "token=123456"});
		assert.equal(go.shortener.get(id).headers.cookie, "token=123456")
	`)
	if err != nil {
		t.Fatal(err)
	}
}
