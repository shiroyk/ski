package cache

import (
	"fmt"
	"testing"

	"github.com/shiroyk/cloudcat/cache/memory"
)

func TestShortener(t *testing.T) {
	shortener := memory.NewShortener()

	req := `POST http://localhost
Content-Type: application/json

{"key":"foo"}`

	id := shortener.Set(req)

	h, ok := shortener.Get(id)
	if !ok {
		t.Fatal(fmt.Sprintf("not found: %v", id))
	}

	if req != h {
		t.Fatalf("want: %s, got %s", req, h)
	}
}
