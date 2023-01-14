package js

import (
	"testing"
)

func TestShortener(t *testing.T) {
	tpl := `POST http://localhost
Content-Type: application/json

{\"key\":\"foo\"}`
	_, err := testVM.RunString(
		"id = go.shortener.set(`" + tpl + "`);" +
			"assert.equal(go.shortener.get(id), `" + tpl + "`)",
	)
	if err != nil {
		t.Fatal(err)
	}
}
