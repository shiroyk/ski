package cache

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/shiroyk/cloudcat/cache/memory"
)

func TestShortener(t *testing.T) {
	shortener := memory.NewShortener()
	context.Background()

	url := "http://localhost"
	headers := map[string]string{
		"Referer": "http://localhost",
	}
	id := shortener.Set(url, headers)

	u, h, ok := shortener.Get(id)
	if !ok {
		t.Fatal(fmt.Sprintf("not found: %v", id))
	}

	if u != url {
		t.Fatalf("unexpected url: %s", u)
	}

	if !reflect.DeepEqual(headers, h) {
		t.Fatalf("unexpected headers: %v", headers)
	}
}
